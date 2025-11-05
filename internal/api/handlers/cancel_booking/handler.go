package cancel_booking

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/m04kA/SMC-BookingService/internal/api/handlers"
	"github.com/m04kA/SMC-BookingService/internal/service/bookings"
)

const (
	msgInvalidBookingID   = "некорректный ID бронирования"
	msgInvalidRequestBody = "некорректное тело запроса"
	msgNotFound           = "бронирование не найдено"
	msgForbidden          = "доступ запрещен"
	msgCannotCancel       = "бронирование не может быть отменено"
)

type Handler struct {
	service BookingService
	logger  Logger
}

func NewHandler(service BookingService, logger Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Handle PATCH /api/v1/bookings/{bookingId}/cancel
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем bookingId из URL
	vars := mux.Vars(r)
	bookingIDStr := vars["bookingId"]

	bookingID, err := strconv.ParseInt(bookingIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("PATCH /bookings/{id}/cancel - Invalid booking ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidBookingID)
		return
	}

	// Декодируем body
	var req CancelBookingRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		h.logger.Warn("PATCH /bookings/{id}/cancel - Invalid request body: %v", err)
		handlers.RespondBadRequest(w, msgInvalidRequestBody)
		return
	}

	// Конвертируем в модель сервиса
	serviceReq := req.ToServiceRequest()

	// Отменяем бронирование
	err = h.service.Cancel(r.Context(), bookingID, serviceReq)
	if err != nil {
		switch {
		case errors.Is(err, bookings.ErrBookingNotFound):
			h.logger.Warn("PATCH /bookings/{id}/cancel - Booking not found: booking_id=%d", bookingID)
			handlers.RespondNotFound(w, msgNotFound)

		case errors.Is(err, bookings.ErrAccessDenied):
			h.logger.Warn("PATCH /bookings/{id}/cancel - Access denied: booking_id=%d, user_id=%d",
				bookingID, req.UserID)
			handlers.RespondForbidden(w, msgForbidden)

		case errors.Is(err, bookings.ErrCannotCancel):
			h.logger.Warn("PATCH /bookings/{id}/cancel - Cannot cancel: booking_id=%d", bookingID)
			handlers.RespondBadRequest(w, msgCannotCancel)

		default:
			h.logger.Error("PATCH /bookings/{id}/cancel - Failed to cancel booking: booking_id=%d, error=%v",
				bookingID, err)
			handlers.RespondInternalError(w)
		}
		return
	}

	h.logger.Info("PATCH /bookings/{id}/cancel - Booking cancelled successfully: booking_id=%d, user_id=%d",
		bookingID, req.UserID)
	handlers.RespondJSON(w, http.StatusOK, nil)
}

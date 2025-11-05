package get_booking

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/m04kA/SMC-BookingService/internal/api/handlers"
	"github.com/m04kA/SMC-BookingService/internal/api/middleware"
	"github.com/m04kA/SMC-BookingService/internal/service/bookings"
)

const (
	msgInvalidBookingID = "некорректный ID бронирования"
	msgNotFound         = "бронирование не найдено"
	msgMissingUserID    = "отсутствует ID пользователя"
	msgForbidden        = "доступ запрещен"
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

// Handle GET /api/v1/bookings/{bookingId}
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем bookingId из URL
	vars := mux.Vars(r)
	bookingIDStr := vars["bookingId"]

	bookingID, err := strconv.ParseInt(bookingIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("GET /bookings/{id} - Invalid booking ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidBookingID)
		return
	}

	// Получаем userID из контекста (через middleware Auth)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.logger.Warn("GET /bookings/{id} - Missing user ID")
		handlers.RespondUnauthorized(w, msgMissingUserID)
		return
	}

	// Получаем бронирование (сервис сам проверит права доступа)
	booking, err := h.service.GetByID(r.Context(), bookingID, userID)
	if err != nil {
		switch {
		case errors.Is(err, bookings.ErrBookingNotFound):
			h.logger.Warn("GET /bookings/{id} - Booking not found: booking_id=%d", bookingID)
			handlers.RespondNotFound(w, msgNotFound)

		case errors.Is(err, bookings.ErrAccessDenied):
			h.logger.Warn("GET /bookings/{id} - Access denied: booking_id=%d, user_id=%d", bookingID, userID)
			handlers.RespondForbidden(w, msgForbidden)

		default:
			h.logger.Error("GET /bookings/{id} - Failed to get booking: booking_id=%d, error=%v", bookingID, err)
			handlers.RespondInternalError(w)
		}
		return
	}

	h.logger.Info("GET /bookings/{id} - Booking retrieved successfully: booking_id=%d, user_id=%d",
		bookingID, userID)
	handlers.RespondJSON(w, http.StatusOK, booking)
}

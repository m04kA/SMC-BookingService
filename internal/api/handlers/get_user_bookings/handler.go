package get_user_bookings

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/m04kA/SMC-BookingService/internal/api/handlers"
	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

const (
	msgInvalidUserID = "некорректный ID пользователя"
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

// Handle GET /api/v1/users/{userId}/bookings
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем userId из URL
	vars := mux.Vars(r)
	userIDStr := vars["userId"]

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("GET /users/{userId}/bookings - Invalid user ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidUserID)
		return
	}

	// Получаем status из query параметров (опционально)
	status := r.URL.Query().Get("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	// Формируем запрос к сервису
	serviceReq := &models.GetUserBookingsRequest{
		UserID: userID,
		Status: statusPtr,
	}

	// Получаем бронирования пользователя
	result, err := h.service.GetUserBookings(r.Context(), serviceReq)
	if err != nil {
		h.logger.Error("GET /users/{userId}/bookings - Failed to get bookings: user_id=%d, error=%v",
			userID, err)
		handlers.RespondInternalError(w)
		return
	}

	h.logger.Info("GET /users/{userId}/bookings - Bookings retrieved successfully: user_id=%d, count=%d",
		userID, len(result.Bookings))
	handlers.RespondJSON(w, http.StatusOK, result.Bookings)
}

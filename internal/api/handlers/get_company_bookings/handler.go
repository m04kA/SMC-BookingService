package get_company_bookings

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
	msgInvalidCompanyID = "некорректный ID компании"
	msgMissingUserID    = "отсутствует ID пользователя"
	msgInvalidParams    = "некорректные параметры запроса"
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

// Handle GET /api/v1/companies/{companyId}/bookings
// Query params: addressId, status, date, includeInactive (опционально)
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем companyId из URL
	vars := mux.Vars(r)
	companyIDStr := vars["companyId"]

	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/bookings - Invalid company ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidCompanyID)
		return
	}

	// Получаем userID из контекста (через middleware Auth)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.logger.Warn("GET /companies/{id}/bookings - Missing user ID")
		handlers.RespondUnauthorized(w, msgMissingUserID)
		return
	}

	// Получаем опциональные query параметры
	addressIDStr := r.URL.Query().Get("addressId")
	statusStr := r.URL.Query().Get("status")
	dateStr := r.URL.Query().Get("date")
	includeInactiveStr := r.URL.Query().Get("includeInactive")

	// Формируем запрос к сервису
	serviceReq, err := ToServiceRequest(companyID, userID, addressIDStr, statusStr, dateStr, includeInactiveStr)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/bookings - Invalid parameters: %v", err)
		handlers.RespondBadRequest(w, msgInvalidParams)
		return
	}

	// Получаем бронирования компании (сервис сам проверит права менеджера)
	result, err := h.service.GetCompanyBookings(r.Context(), serviceReq)
	if err != nil {
		switch {
		case errors.Is(err, bookings.ErrAccessDenied):
			h.logger.Warn("GET /companies/{id}/bookings - Access denied: company_id=%d, user_id=%d",
				companyID, userID)
			handlers.RespondForbidden(w, msgForbidden)

		default:
			h.logger.Error("GET /companies/{id}/bookings - Failed to get bookings: company_id=%d, error=%v",
				companyID, err)
			handlers.RespondInternalError(w)
		}
		return
	}

	h.logger.Info("GET /companies/{id}/bookings - Bookings retrieved successfully: company_id=%d, count=%d",
		companyID, len(result.Bookings))
	handlers.RespondJSON(w, http.StatusOK, result.Bookings)
}

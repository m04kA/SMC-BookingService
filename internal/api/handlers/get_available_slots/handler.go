package get_available_slots

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/m04kA/SMC-BookingService/internal/api/handlers"
	getAvailableSlots "github.com/m04kA/SMC-BookingService/internal/usecase/get_available_slots"
)

const (
	msgInvalidCompanyID    = "некорректный ID компании"
	msgInvalidAddressID    = "некорректный ID адреса"
	msgInvalidServiceID    = "некорректный ID услуги"
	msgMissingServiceID    = "ID услуги обязателен"
	msgMissingDate         = "дата обязательна"
	msgInvalidDate         = "некорректный формат даты, ожидается YYYY-MM-DD"
	msgCompanyNotFound     = "компания не найдена"
	msgAddressNotFound     = "адрес не найден"
	msgServiceNotFound     = "услуга не найдена"
	msgServiceNotAvailable = "услуга недоступна на выбранном адресе"
)

type Handler struct {
	useCase GetAvailableSlotsUseCase
	logger  Logger
}

func NewHandler(useCase GetAvailableSlotsUseCase, logger Logger) *Handler {
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Handle GET /api/v1/companies/{companyId}/addresses/{addressId}/available-slots
// Query params: serviceId (required), date (required, YYYY-MM-DD)
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Извлекаем companyId из URL
	companyIDStr := vars["companyId"]
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Invalid company ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidCompanyID)
		return
	}

	// Извлекаем addressId из URL
	addressIDStr := vars["addressId"]
	addressID, err := strconv.ParseInt(addressIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Invalid address ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidAddressID)
		return
	}

	// Извлекаем serviceId из query параметров
	serviceIDStr := r.URL.Query().Get("serviceId")
	if serviceIDStr == "" {
		h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Missing service ID")
		handlers.RespondBadRequest(w, msgMissingServiceID)
		return
	}

	serviceID, err := strconv.ParseInt(serviceIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Invalid service ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidServiceID)
		return
	}

	// Извлекаем date из query параметров
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Missing date")
		handlers.RespondBadRequest(w, msgMissingDate)
		return
	}

	// Формируем запрос к use case (с парсингом даты)
	useCaseReq, err := ToUseCaseRequest(companyID, addressID, serviceID, dateStr)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Invalid date format: %v", err)
		handlers.RespondBadRequest(w, msgInvalidDate)
		return
	}

	// Вызываем use case
	result, err := h.useCase.Execute(r.Context(), useCaseReq)
	if err != nil {
		// Обработка ошибок use case
		switch {
		case errors.Is(err, getAvailableSlots.ErrCompanyNotFound):
			h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Company not found: company_id=%d", companyID)
			handlers.RespondNotFound(w, msgCompanyNotFound)

		case errors.Is(err, getAvailableSlots.ErrAddressNotFound):
			h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Address not found: company_id=%d, address_id=%d",
				companyID, addressID)
			handlers.RespondNotFound(w, msgAddressNotFound)

		case errors.Is(err, getAvailableSlots.ErrServiceNotFound):
			h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Service not found: company_id=%d, service_id=%d",
				companyID, serviceID)
			handlers.RespondNotFound(w, msgServiceNotFound)

		case errors.Is(err, getAvailableSlots.ErrServiceNotAvailableAtAddress):
			h.logger.Warn("GET /companies/{id}/addresses/{id}/available-slots - Service not available at address: company_id=%d, address_id=%d, service_id=%d",
				companyID, addressID, serviceID)
			handlers.RespondBadRequest(w, msgServiceNotAvailable)

		default:
			h.logger.Error("GET /companies/{id}/addresses/{id}/available-slots - Failed to get slots: company_id=%d, address_id=%d, service_id=%d, error=%v",
				companyID, addressID, serviceID, err)
			handlers.RespondInternalError(w)
		}
		return
	}

	// Формируем HTTP ответ
	response := FromUseCaseResponse(result)

	h.logger.Info("GET /companies/{id}/addresses/{id}/available-slots - Slots retrieved successfully: company_id=%d, address_id=%d, service_id=%d, slots_count=%d",
		companyID, addressID, serviceID, len(result.Slots))
	handlers.RespondJSON(w, http.StatusOK, response)
}

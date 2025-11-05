package get_company_config

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/m04kA/SMC-BookingService/internal/api/handlers"
	"github.com/m04kA/SMC-BookingService/internal/service/config"
)

const (
	msgInvalidCompanyID = "некорректный ID компании"
	msgInvalidParams    = "некорректные параметры запроса"
)

type Handler struct {
	service ConfigService
	logger  Logger
}

func NewHandler(service ConfigService, logger Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Handle GET /api/v1/companies/{companyId}/config
// Query params: addressId, serviceId (опционально)
// Публичный endpoint - без авторизации
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем companyId из URL
	vars := mux.Vars(r)
	companyIDStr := vars["companyId"]

	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/config - Invalid company ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidCompanyID)
		return
	}

	// Получаем опциональные query параметры
	addressIDStr := r.URL.Query().Get("addressId")
	serviceIDStr := r.URL.Query().Get("serviceId")

	// Формируем запрос к сервису
	serviceReq, err := ToServiceRequest(companyID, addressIDStr, serviceIDStr)
	if err != nil {
		h.logger.Warn("GET /companies/{id}/config - Invalid parameters: %v", err)
		handlers.RespondBadRequest(w, msgInvalidParams)
		return
	}

	// Получаем конфигурацию с иерархическим поиском
	result, err := h.service.GetWithHierarchy(r.Context(), serviceReq)
	if err != nil {
		// Если конфигурация не найдена - возвращаем дефолтные значения
		if errors.Is(err, config.ErrConfigNotFound) {
			h.logger.Info("GET /companies/{id}/config - Config not found, returning defaults: company_id=%d",
				companyID)
			defaultConfig := GetDefaultConfigResponse(companyID)
			handlers.RespondJSON(w, http.StatusOK, defaultConfig)
			return
		}

		h.logger.Error("GET /companies/{id}/config - Failed to get config: company_id=%d, error=%v",
			companyID, err)
		handlers.RespondInternalError(w)
		return
	}

	h.logger.Info("GET /companies/{id}/config - Config retrieved successfully: company_id=%d, config_id=%d",
		companyID, result.ID)
	handlers.RespondJSON(w, http.StatusOK, result)
}

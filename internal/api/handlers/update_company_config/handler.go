package update_company_config

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/m04kA/SMC-BookingService/internal/api/handlers"
	"github.com/m04kA/SMC-BookingService/internal/service/config"
)

const (
	msgInvalidCompanyID   = "некорректный ID компании"
	msgInvalidRequestBody = "некорректное тело запроса"
	msgNotFound           = "конфигурация не найдена"
	msgForbidden          = "доступ запрещен"
	msgInvalidData        = "некорректные данные конфигурации"
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

// Handle PUT /api/v1/companies/{companyId}/config
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Извлекаем companyId из URL
	vars := mux.Vars(r)
	companyIDStr := vars["companyId"]

	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("PUT /companies/{id}/config - Invalid company ID: %v", err)
		handlers.RespondBadRequest(w, msgInvalidCompanyID)
		return
	}

	// Декодируем body
	var req UpdateCompanyConfigRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		h.logger.Warn("PUT /companies/{id}/config - Invalid request body: %v", err)
		handlers.RespondBadRequest(w, msgInvalidRequestBody)
		return
	}

	// Ищем существующую конфигурацию по (companyId, addressId, serviceId)
	getReq := ToGetConfigRequest(companyID, req.AddressID, req.ServiceID)
	existingConfig, err := h.service.GetWithHierarchy(r.Context(), getReq)
	if err != nil {
		if errors.Is(err, config.ErrConfigNotFound) {
			h.logger.Warn("PUT /companies/{id}/config - Config not found: company_id=%d, address_id=%v, service_id=%v",
				companyID, req.AddressID, req.ServiceID)
			handlers.RespondNotFound(w, msgNotFound)
			return
		}

		h.logger.Error("PUT /companies/{id}/config - Failed to get config: company_id=%d, error=%v",
			companyID, err)
		handlers.RespondInternalError(w)
		return
	}

	// Конвертируем в модель сервиса для обновления
	updateReq := req.ToServiceRequest()

	// Обновляем конфигурацию (сервис сам проверит права менеджера)
	result, err := h.service.Update(r.Context(), existingConfig.ID, updateReq)
	if err != nil {
		switch {
		case errors.Is(err, config.ErrConfigNotFound):
			h.logger.Warn("PUT /companies/{id}/config - Config not found during update: config_id=%d",
				existingConfig.ID)
			handlers.RespondNotFound(w, msgNotFound)

		case errors.Is(err, config.ErrAccessDenied):
			h.logger.Warn("PUT /companies/{id}/config - Access denied: company_id=%d, user_id=%d",
				companyID, req.UserID)
			handlers.RespondForbidden(w, msgForbidden)

		case errors.Is(err, config.ErrInvalidInput):
			h.logger.Warn("PUT /companies/{id}/config - Invalid data: company_id=%d, error=%v",
				companyID, err)
			handlers.RespondBadRequest(w, msgInvalidData)

		default:
			h.logger.Error("PUT /companies/{id}/config - Failed to update config: company_id=%d, error=%v",
				companyID, err)
			handlers.RespondInternalError(w)
		}
		return
	}

	h.logger.Info("PUT /companies/{id}/config - Config updated successfully: company_id=%d, config_id=%d",
		companyID, result.ID)
	handlers.RespondJSON(w, http.StatusOK, result)
}

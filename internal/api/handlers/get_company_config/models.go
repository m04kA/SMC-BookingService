package get_company_config

import (
	"strconv"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/internal/service/config/models"
)

// ToServiceRequest формирует запрос к сервису из URL и query параметров
func ToServiceRequest(companyID int64, addressIDStr string, serviceIDStr string) (*models.GetConfigRequest, error) {
	req := &models.GetConfigRequest{
		CompanyID: companyID,
		AddressID: nil, // nil означает отсутствие addressID
		ServiceID: nil, // nil означает отсутствие serviceID
	}

	// Парсим addressId если указан
	if addressIDStr != "" {
		addressID, err := strconv.ParseInt(addressIDStr, 10, 64)
		if err != nil {
			return nil, err
		}
		req.AddressID = &addressID
	}

	// Парсим serviceId если указан
	if serviceIDStr != "" {
		serviceID, err := strconv.ParseInt(serviceIDStr, 10, 64)
		if err != nil {
			return nil, err
		}
		req.ServiceID = &serviceID
	}

	return req, nil
}

// GetDefaultConfigResponse возвращает дефолтную конфигурацию
func GetDefaultConfigResponse(companyID int64) *models.ConfigResponse {
	return &models.ConfigResponse{
		ID:                      0,  // 0 означает, что это не из БД
		CompanyID:               companyID,
		AddressID:               nil,
		ServiceID:               nil,
		SlotDurationMinutes:     domain.DefaultSlotDurationMinutes,
		MaxConcurrentBookings:   domain.DefaultMaxConcurrentBookings,
		AdvanceBookingDays:      domain.DefaultAdvanceBookingDays,
		MinBookingNoticeMinutes: domain.DefaultMinBookingNoticeMinutes,
	}
}

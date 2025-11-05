package update_company_config

import (
	"github.com/m04kA/SMC-BookingService/internal/service/config/models"
)

// UpdateCompanyConfigRequest HTTP request model
type UpdateCompanyConfigRequest struct {
	UserID                  int64  `json:"userId"`
	AddressID               *int64 `json:"addressId,omitempty"`
	ServiceID               *int64 `json:"serviceId,omitempty"`
	SlotDurationMinutes     *int   `json:"slotDurationMinutes,omitempty"`
	MaxConcurrentBookings   *int   `json:"maxConcurrentBookings,omitempty"`
	AdvanceBookingDays      *int   `json:"advanceBookingDays,omitempty"`
	MinBookingNoticeMinutes *int   `json:"minBookingNoticeMinutes,omitempty"`
}

// ToServiceRequest конвертирует HTTP request в модель сервиса
func (r *UpdateCompanyConfigRequest) ToServiceRequest() *models.UpdateConfigRequest {
	return &models.UpdateConfigRequest{
		UserID:                  r.UserID,
		SlotDurationMinutes:     r.SlotDurationMinutes,
		MaxConcurrentBookings:   r.MaxConcurrentBookings,
		AdvanceBookingDays:      r.AdvanceBookingDays,
		MinBookingNoticeMinutes: r.MinBookingNoticeMinutes,
	}
}

// ToGetConfigRequest создаёт запрос для поиска конфигурации
func ToGetConfigRequest(companyID int64, addressID *int64, serviceID *int64) *models.GetConfigRequest {
	return &models.GetConfigRequest{
		CompanyID: companyID,
		AddressID: addressID,
		ServiceID: serviceID,
	}
}

package models

import (
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
)

// Request модели

// CreateConfigRequest запрос на создание конфигурации слотов
type CreateConfigRequest struct {
	UserID                  int64  `json:"userId"`
	CompanyID               int64  `json:"companyId"`
	AddressID               *int64 `json:"addressId,omitempty"`               // NULL = для всех адресов
	ServiceID               *int64 `json:"serviceId,omitempty"`               // NULL = для всех услуг
	SlotDurationMinutes     int    `json:"slotDurationMinutes"`               // 15, 30, 60, etc.
	MaxConcurrentBookings   int    `json:"maxConcurrentBookings"`             // Количество одновременных бронирований
	AdvanceBookingDays      int    `json:"advanceBookingDays"`                // 0 = без ограничений
	MinBookingNoticeMinutes int    `json:"minBookingNoticeMinutes"`           // Минимальное время до бронирования
}

// UpdateConfigRequest запрос на обновление конфигурации слотов
// Все поля опциональны - обновляются только переданные значения
type UpdateConfigRequest struct {
	UserID                  int64 `json:"userId"`
	SlotDurationMinutes     *int  `json:"slotDurationMinutes,omitempty"`
	MaxConcurrentBookings   *int  `json:"maxConcurrentBookings,omitempty"`
	AdvanceBookingDays      *int  `json:"advanceBookingDays,omitempty"`
	MinBookingNoticeMinutes *int  `json:"minBookingNoticeMinutes,omitempty"`
}

// GetConfigRequest запрос на получение конфигурации (для иерархического поиска)
// AddressID и ServiceID могут быть nil для иерархического поиска
type GetConfigRequest struct {
	CompanyID int64  `json:"companyId"`
	AddressID *int64 `json:"addressId,omitempty"` // nil означает любой адрес
	ServiceID *int64 `json:"serviceId,omitempty"` // nil означает любая услуга
}

// DeleteConfigRequest запрос на удаление конфигурации
type DeleteConfigRequest struct {
	UserID    int64  `json:"userId"`
	CompanyID int64  `json:"companyId"`
	AddressID *int64 `json:"addressId,omitempty"`
	ServiceID *int64 `json:"serviceId,omitempty"`
}

// Response модели

// ConfigResponse ответ с данными конфигурации слотов
type ConfigResponse struct {
	ID                      int64     `json:"id"`
	CompanyID               int64     `json:"companyId"`
	AddressID               *int64    `json:"addressId,omitempty"`
	ServiceID               *int64    `json:"serviceId,omitempty"`
	SlotDurationMinutes     int       `json:"slotDurationMinutes"`
	MaxConcurrentBookings   int       `json:"maxConcurrentBookings"`
	AdvanceBookingDays      int       `json:"advanceBookingDays"`
	MinBookingNoticeMinutes int       `json:"minBookingNoticeMinutes"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

// ConfigListResponse ответ со списком конфигураций
type ConfigListResponse struct {
	Configs []ConfigResponse `json:"configs"`
}

// Методы конвертации

// FromDomainConfig конвертирует domain модель в DTO
func FromDomainConfig(c *domain.CompanySlotsConfig) *ConfigResponse {
	if c == nil {
		return nil
	}

	return &ConfigResponse{
		ID:                      c.ID,
		CompanyID:               c.CompanyID,
		AddressID:               c.AddressID,
		ServiceID:               c.ServiceID,
		SlotDurationMinutes:     c.SlotDurationMinutes,
		MaxConcurrentBookings:   c.MaxConcurrentBookings,
		AdvanceBookingDays:      c.AdvanceBookingDays,
		MinBookingNoticeMinutes: c.MinBookingNoticeMinutes,
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
	}
}

// FromDomainConfigList конвертирует список domain моделей в DTO
func FromDomainConfigList(configs []*domain.CompanySlotsConfig) *ConfigListResponse {
	if configs == nil {
		return &ConfigListResponse{
			Configs: []ConfigResponse{},
		}
	}

	resp := &ConfigListResponse{
		Configs: make([]ConfigResponse, len(configs)),
	}

	for i, config := range configs {
		if configResp := FromDomainConfig(config); configResp != nil {
			resp.Configs[i] = *configResp
		}
	}

	return resp
}

// ToDomainConfig конвертирует CreateConfigRequest в domain модель
func (r *CreateConfigRequest) ToDomainConfig() *domain.CompanySlotsConfig {
	return &domain.CompanySlotsConfig{
		CompanyID:               r.CompanyID,
		AddressID:               r.AddressID,
		ServiceID:               r.ServiceID,
		SlotDurationMinutes:     r.SlotDurationMinutes,
		MaxConcurrentBookings:   r.MaxConcurrentBookings,
		AdvanceBookingDays:      r.AdvanceBookingDays,
		MinBookingNoticeMinutes: r.MinBookingNoticeMinutes,
	}
}

// ApplyToConfig применяет обновления к существующей конфигурации
// Обновляются только непустые (not nil) поля из request
func (r *UpdateConfigRequest) ApplyToConfig(config *domain.CompanySlotsConfig) {
	if r.SlotDurationMinutes != nil {
		config.SlotDurationMinutes = *r.SlotDurationMinutes
	}
	if r.MaxConcurrentBookings != nil {
		config.MaxConcurrentBookings = *r.MaxConcurrentBookings
	}
	if r.AdvanceBookingDays != nil {
		config.AdvanceBookingDays = *r.AdvanceBookingDays
	}
	if r.MinBookingNoticeMinutes != nil {
		config.MinBookingNoticeMinutes = *r.MinBookingNoticeMinutes
	}
}

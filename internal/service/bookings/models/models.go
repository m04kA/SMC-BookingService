package models

import (
	"errors"
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
)

var (
	// ErrInvalidStatus возвращается при некорректном статусе
	ErrInvalidStatus = errors.New("invalid booking status")
)

// Request модели

// CancelBookingRequest запрос на отмену бронирования
type CancelBookingRequest struct {
	UserID             int64  `json:"userId"`
	CancellationReason string `json:"cancellationReason"`
}

// UpdateStatusRequest запрос на обновление статуса бронирования
type UpdateStatusRequest struct {
	UserID int64  `json:"userId"`
	Status string `json:"status"`
}

// GetUserBookingsRequest запрос на получение бронирований пользователя
type GetUserBookingsRequest struct {
	UserID int64   `json:"userId"`
	Status *string `json:"status,omitempty"`
}

// GetCompanyBookingsRequest запрос на получение бронирований компании
type GetCompanyBookingsRequest struct {
	UserID          int64      `json:"userId"`
	CompanyID       int64      `json:"companyId"`
	AddressID       *int64     `json:"addressId,omitempty"`       // Фильтр по адресу (опционально)
	StartDate       *time.Time `json:"startDate,omitempty"`       // Начало периода (опционально)
	EndDate         *time.Time `json:"endDate,omitempty"`         // Конец периода (опционально)
	Status          *string    `json:"status,omitempty"`          // Фильтр по статусу (опционально)
	IncludeInactive bool       `json:"includeInactive,omitempty"` // Включить отменённые бронирования
}

// ToDomainFilter конвертирует request в domain фильтр
func (r *GetCompanyBookingsRequest) ToDomainFilter() (domain.CompanyBookingsFilter, error) {
	filter := domain.CompanyBookingsFilter{
		CompanyID:       r.CompanyID,
		AddressID:       r.AddressID,
		StartDate:       r.StartDate,
		EndDate:         r.EndDate,
		IncludeInactive: r.IncludeInactive,
	}

	// Конвертируем статус если указан
	if r.Status != nil {
		status, err := ToDomainBookingStatus(*r.Status)
		if err != nil {
			return filter, err
		}
		filter.Status = &status
	}

	return filter, nil
}

// Response модели

// BookingResponse ответ с данными бронирования
type BookingResponse struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"userId"`
	CompanyID       int64   `json:"companyId"`
	AddressID       int64   `json:"addressId"`
	ServiceID       int64   `json:"serviceId"`
	CarID           int64   `json:"carId"`
	BookingDate     string  `json:"bookingDate"`     // "2025-10-15"
	StartTime       string  `json:"startTime"`       // "10:00"
	DurationMinutes int     `json:"durationMinutes"`
	Status          string  `json:"status"`

	// Денормализованные данные
	ServiceName     string   `json:"serviceName"`
	ServicePrice    float64  `json:"servicePrice"`
	CarBrand        *string  `json:"carBrand,omitempty"`
	CarModel        *string  `json:"carModel,omitempty"`
	CarLicensePlate *string  `json:"carLicensePlate,omitempty"`
	Notes           *string  `json:"notes,omitempty"`

	CancellationReason *string `json:"cancellationReason,omitempty"`
	CancelledAt        *string `json:"cancelledAt,omitempty"` // ISO 8601 format

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BookingListResponse ответ со списком бронирований
type BookingListResponse struct {
	Bookings []BookingResponse `json:"bookings"`
}

// Методы конвертации

// FromDomainBooking конвертирует domain модель в DTO
func FromDomainBooking(b *domain.Booking) *BookingResponse {
	if b == nil {
		return nil
	}

	resp := &BookingResponse{
		ID:              b.ID,
		UserID:          b.UserID,
		CompanyID:       b.CompanyID,
		AddressID:       b.AddressID,
		ServiceID:       b.ServiceID,
		CarID:           b.CarID,
		BookingDate:     b.BookingDate.Format(domain.DateFormat),
		StartTime:       b.StartTime.String(),
		DurationMinutes: b.DurationMinutes,
		Status:          string(b.Status),
		ServiceName:     b.ServiceName,
		ServicePrice:    b.ServicePrice,
		CarBrand:        b.CarBrand,
		CarModel:        b.CarModel,
		CarLicensePlate: b.CarLicensePlate,
		Notes:           b.Notes,
		CancellationReason: b.CancellationReason,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}

	// Конвертируем CancelledAt в строку ISO 8601
	if b.CancelledAt != nil {
		cancelledStr := b.CancelledAt.Format(time.RFC3339)
		resp.CancelledAt = &cancelledStr
	}

	return resp
}

// FromDomainBookingList конвертирует список domain моделей в DTO
func FromDomainBookingList(bookings []*domain.Booking) *BookingListResponse {
	if bookings == nil {
		return &BookingListResponse{
			Bookings: []BookingResponse{},
		}
	}

	resp := &BookingListResponse{
		Bookings: make([]BookingResponse, len(bookings)),
	}

	for i, booking := range bookings {
		if bookingResp := FromDomainBooking(booking); bookingResp != nil {
			resp.Bookings[i] = *bookingResp
		}
	}

	return resp
}

// ToDomainBookingStatus конвертирует строку в domain.BookingStatus с валидацией
func ToDomainBookingStatus(status string) (domain.BookingStatus, error) {
	s := domain.BookingStatus(status)

	// Валидируем статус
	validStatuses := []domain.BookingStatus{
		domain.StatusPending,
		domain.StatusConfirmed,
		domain.StatusInProgress,
		domain.StatusCompleted,
		domain.StatusCancelledByUser,
		domain.StatusCancelledByCompany,
		domain.StatusNoShow,
	}

	for _, valid := range validStatuses {
		if s == valid {
			return s, nil
		}
	}

	return "", ErrInvalidStatus
}

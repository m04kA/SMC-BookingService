package domain

import (
	"time"

	"github.com/m04kA/SMC-BookingService/pkg/types"
)

// BookingStatus represents the status of a booking
type BookingStatus string

const (
	StatusPending            BookingStatus = "pending"
	StatusConfirmed          BookingStatus = "confirmed"
	StatusInProgress         BookingStatus = "in_progress"
	StatusCompleted          BookingStatus = "completed"
	StatusCancelledByUser    BookingStatus = "cancelled_by_user"
	StatusCancelledByCompany BookingStatus = "cancelled_by_company"
	StatusNoShow             BookingStatus = "no_show"
)

// Booking represents a service booking in the system
type Booking struct {
	ID              int64
	UserID          int64
	CompanyID       int64
	AddressID       int64 // ID адреса компании (компания может иметь несколько точек обслуживания)
	ServiceID       int64
	CarID           int64
	BookingDate     time.Time
	StartTime       types.TimeString
	DurationMinutes int
	Status          BookingStatus

	// Denormalized data for history
	ServiceName     string
	ServicePrice    float64
	CarBrand        *string
	CarModel        *string
	CarLicensePlate *string
	Notes           *string

	CancellationReason *string
	CancelledAt        *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsActive returns true if the booking is in an active state
func (b *Booking) IsActive() bool {
	return b.Status != StatusCancelledByUser &&
		b.Status != StatusCancelledByCompany &&
		b.Status != StatusNoShow
}

// CanBeCancelled returns true if the booking can be cancelled
func (b *Booking) CanBeCancelled() bool {
	return b.Status == StatusPending || b.Status == StatusConfirmed
}

// CanBeUpdated returns true if the booking can be updated
func (b *Booking) CanBeUpdated() bool {
	return b.Status == StatusPending || b.Status == StatusConfirmed
}

// IsCancelled returns true if the booking has been cancelled
func (b *Booking) IsCancelled() bool {
	return b.Status == StatusCancelledByUser || b.Status == StatusCancelledByCompany
}

// IsCompleted returns true if the booking is completed or was a no-show
func (b *Booking) IsCompleted() bool {
	return b.Status == StatusCompleted || b.Status == StatusNoShow
}

// CompanyBookingsFilter фильтр для получения бронирований компании
type CompanyBookingsFilter struct {
	CompanyID       int64          // Обязательный параметр
	AddressID       *int64         // Фильтр по адресу (опционально, если nil - все адреса)
	StartDate       *time.Time     // Начало периода (опционально, если nil - без ограничения)
	EndDate         *time.Time     // Конец периода (опционально, если nil - без ограничения)
	Status          *BookingStatus // Фильтр по статусу (опционально)
	IncludeInactive bool           // Включать ли неактивные бронирования (отмененные, no-show)
}

package create_booking

import (
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	createBooking "github.com/m04kA/SMC-BookingService/internal/usecase/create_booking"
	"github.com/m04kA/SMC-BookingService/pkg/types"
)

// CreateBookingRequest HTTP request model
type CreateBookingRequest struct {
	UserID      int64   `json:"userId"`
	CompanyID   int64   `json:"companyId"`
	AddressID   int64   `json:"addressId"`
	ServiceID   int64   `json:"serviceId"`
	BookingDate string  `json:"bookingDate"` // "2025-10-15"
	StartTime   string  `json:"startTime"`   // "10:00"
	Notes       *string `json:"notes,omitempty"`
}

// BookingResponse HTTP response model
type BookingResponse struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"userId"`
	CompanyID       int64   `json:"companyId"`
	AddressID       int64   `json:"addressId"`
	ServiceID       int64   `json:"serviceId"`
	CarID           int64   `json:"carId"`
	BookingDate     string  `json:"bookingDate"`
	StartTime       string  `json:"startTime"`
	DurationMinutes int     `json:"durationMinutes"`
	Status          string  `json:"status"`
	ServiceName     string  `json:"serviceName"`
	ServicePrice    float64 `json:"servicePrice"`
	CarBrand        *string `json:"carBrand,omitempty"`
	CarModel        *string `json:"carModel,omitempty"`
	CarLicensePlate *string `json:"carLicensePlate,omitempty"`
	Notes           *string `json:"notes,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

// ToUseCaseRequest конвертирует HTTP запрос в модель use case
func (r *CreateBookingRequest) ToUseCaseRequest() (*createBooking.Request, error) {
	// Парсим дату
	bookingDate, err := time.Parse(domain.DateFormat, r.BookingDate)
	if err != nil {
		return nil, err
	}

	// Парсим время
	startTime, err := types.NewTimeStringFromString(r.StartTime)
	if err != nil {
		return nil, err
	}

	return &createBooking.Request{
		UserID:    r.UserID,
		CompanyID: r.CompanyID,
		AddressID: r.AddressID,
		ServiceID: r.ServiceID,
		Date:      bookingDate,
		StartTime: startTime,
		Notes:     r.Notes,
	}, nil
}

// FromUseCaseResponse конвертирует ответ use case в HTTP response
func FromUseCaseResponse(resp *createBooking.Response) *BookingResponse {
	return &BookingResponse{
		ID:              resp.ID,
		UserID:          resp.UserID,
		CompanyID:       resp.CompanyID,
		AddressID:       resp.AddressID,
		ServiceID:       resp.ServiceID,
		CarID:           resp.CarID,
		BookingDate:     resp.BookingDate.Format(domain.DateFormat),
		StartTime:       resp.StartTime.String(),
		DurationMinutes: resp.DurationMinutes,
		Status:          resp.Status,
		ServiceName:     resp.ServiceName,
		ServicePrice:    resp.ServicePrice,
		CarBrand:        resp.CarBrand,
		CarModel:        resp.CarModel,
		CarLicensePlate: resp.CarLicensePlate,
		Notes:           resp.Notes,
		CreatedAt:       resp.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       resp.UpdatedAt.Format(time.RFC3339),
	}
}

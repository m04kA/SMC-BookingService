package get_company_bookings

import (
	"context"

	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

type BookingService interface {
	GetCompanyBookings(ctx context.Context, req *models.GetCompanyBookingsRequest) (*models.BookingListResponse, error)
}

type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

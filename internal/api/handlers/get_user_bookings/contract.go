package get_user_bookings

import (
	"context"

	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

type BookingService interface {
	GetUserBookings(ctx context.Context, req *models.GetUserBookingsRequest) (*models.BookingListResponse, error)
}

type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

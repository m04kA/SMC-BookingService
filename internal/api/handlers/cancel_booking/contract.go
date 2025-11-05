package cancel_booking

import (
	"context"

	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

type BookingService interface {
	Cancel(ctx context.Context, bookingID int64, req *models.CancelBookingRequest) error
}

type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

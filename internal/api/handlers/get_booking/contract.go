package get_booking

import (
	"context"

	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

type BookingService interface {
	GetByID(ctx context.Context, id int64, userID int64) (*models.BookingResponse, error)
}

type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

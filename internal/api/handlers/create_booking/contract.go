package create_booking

import (
	"context"

	createBooking "github.com/m04kA/SMC-BookingService/internal/usecase/create_booking"
)

type CreateBookingUseCase interface {
	Execute(ctx context.Context, req *createBooking.Request) (*createBooking.Response, error)
}

type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

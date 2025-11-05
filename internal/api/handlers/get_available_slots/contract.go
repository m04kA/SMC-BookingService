package get_available_slots

import (
	"context"

	getAvailableSlots "github.com/m04kA/SMC-BookingService/internal/usecase/get_available_slots"
)

type GetAvailableSlotsUseCase interface {
	Execute(ctx context.Context, req *getAvailableSlots.Request) (*getAvailableSlots.Response, error)
}

type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

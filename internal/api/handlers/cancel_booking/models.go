package cancel_booking

import (
	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

// CancelBookingRequest HTTP request model
type CancelBookingRequest struct {
	UserID             int64   `json:"userId"`
	CancellationReason *string `json:"cancellationReason,omitempty"`
}

// ToServiceRequest конвертирует HTTP request в модель сервиса
func (r *CancelBookingRequest) ToServiceRequest() *models.CancelBookingRequest {
	reason := ""
	if r.CancellationReason != nil {
		reason = *r.CancellationReason
	}

	return &models.CancelBookingRequest{
		UserID:             r.UserID,
		CancellationReason: reason,
	}
}

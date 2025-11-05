package get_available_slots

import (
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	getAvailableSlots "github.com/m04kA/SMC-BookingService/internal/usecase/get_available_slots"
)

// AvailableSlotsResponse HTTP response model
type AvailableSlotsResponse struct {
	Date      string          `json:"date"`
	CompanyID int64           `json:"companyId"`
	AddressID int64           `json:"addressId"`
	ServiceID int64           `json:"serviceId"`
	Slots     []AvailableSlot `json:"slots"`
}

// AvailableSlot модель временного слота
type AvailableSlot struct {
	StartTime       string `json:"startTime"`
	DurationMinutes int    `json:"durationMinutes"`
	AvailableSpots  int    `json:"availableSpots"`
	TotalSpots      int    `json:"totalSpots"`
}

// FromUseCaseResponse конвертирует ответ use case в HTTP response
func FromUseCaseResponse(resp *getAvailableSlots.Response) *AvailableSlotsResponse {
	slots := make([]AvailableSlot, len(resp.Slots))
	for i, slot := range resp.Slots {
		slots[i] = AvailableSlot{
			StartTime:       slot.StartTime.String(),
			DurationMinutes: slot.DurationMinutes,
			AvailableSpots:  slot.AvailableSpots,
			TotalSpots:      slot.TotalSpots,
		}
	}

	return &AvailableSlotsResponse{
		Date:      resp.Date.Format(domain.DateFormat),
		CompanyID: resp.CompanyID,
		AddressID: resp.AddressID,
		ServiceID: resp.ServiceID,
		Slots:     slots,
	}
}

// ToUseCaseRequest создает запрос use case из query параметров
func ToUseCaseRequest(companyID, addressID, serviceID int64, dateStr string) (*getAvailableSlots.Request, error) {
	// Парсим дату
	date, err := time.Parse(domain.DateFormat, dateStr)
	if err != nil {
		return nil, err
	}

	return &getAvailableSlots.Request{
		CompanyID: companyID,
		AddressID: addressID,
		ServiceID: serviceID,
		Date:      date,
	}, nil
}

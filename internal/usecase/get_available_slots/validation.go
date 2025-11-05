package get_available_slots

import (
	"fmt"
	"time"

	"github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
)

// validateRequest валидирует входные данные запроса
func validateRequest(req *Request) error {
	if req.CompanyID <= 0 {
		return fmt.Errorf("%w: companyID must be positive", ErrInvalidInput)
	}

	if req.AddressID <= 0 {
		return fmt.Errorf("%w: addressID must be positive", ErrInvalidInput)
	}

	if req.ServiceID <= 0 {
		return fmt.Errorf("%w: serviceID must be positive", ErrInvalidInput)
	}

	// Проверяем, что дата не является нулевой
	if req.Date.IsZero() {
		return fmt.Errorf("%w: date is required", ErrInvalidInput)
	}

	return nil
}

// validateDate проверяет, что дата подходит для бронирования
func validateDate(requestDate time.Time, now time.Time, advanceBookingDays int) error {
	// Проверяем, что дата не в прошлом
	if isDateInPast(requestDate, now) {
		return ErrInvalidDate
	}

	// Если advanceBookingDays = 0, нет ограничений на дату
	if advanceBookingDays == 0 {
		return nil
	}

	// Проверяем, что дата не превышает ограничение advanceBookingDays
	maxDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, advanceBookingDays)

	requestDateOnly := time.Date(requestDate.Year(), requestDate.Month(), requestDate.Day(), 0, 0, 0, 0, requestDate.Location())

	if requestDateOnly.After(maxDate) {
		return fmt.Errorf("%w: can only book %d days in advance", ErrDateTooFarInFuture, advanceBookingDays)
	}

	return nil
}

// validateAddressExists проверяет, что адрес существует в компании
func validateAddressExists(company *sellerservice.Company, addressID int64) error {
	for _, addr := range company.Addresses {
		if addr.ID == addressID {
			return nil
		}
	}
	return ErrAddressNotFound
}

// validateServiceAtAddress проверяет, что услуга доступна на указанном адресе
func validateServiceAtAddress(service *sellerservice.Service, addressID int64) error {
	for _, addrID := range service.AddressIDs {
		if addrID == addressID {
			return nil
		}
	}
	return ErrServiceNotAvailableAtAddress
}

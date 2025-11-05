package get_available_slots

import "errors"

var (
	// ErrCompanyNotFound возвращается, когда компания не найдена
	ErrCompanyNotFound = errors.New("company not found")

	// ErrAddressNotFound возвращается, когда адрес не найден в компании
	ErrAddressNotFound = errors.New("address not found")

	// ErrServiceNotFound возвращается, когда услуга не найдена
	ErrServiceNotFound = errors.New("service not found")

	// ErrServiceNotAvailableAtAddress возвращается, когда услуга недоступна на указанном адресе
	ErrServiceNotAvailableAtAddress = errors.New("service is not available at this address")

	// ErrInvalidDate возвращается при некорректной дате бронирования
	ErrInvalidDate = errors.New("invalid booking date")

	// ErrDateTooFarInFuture возвращается, когда дата превышает ограничение advanceBookingDays
	ErrDateTooFarInFuture = errors.New("date is too far in the future")

	// ErrCompanyClosed возвращается, когда компания закрыта в указанную дату
	ErrCompanyClosed = errors.New("company is closed on this date")

	// ErrInvalidInput возвращается при некорректных входных данных
	ErrInvalidInput = errors.New("invalid input data")

	// ErrInternal возвращается при внутренних ошибках usecase
	ErrInternal = errors.New("usecase: internal error")
)

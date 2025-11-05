package bookings

import "errors"

var (
	// ErrBookingNotFound возвращается, когда бронирование не найдено
	ErrBookingNotFound = errors.New("booking not found")

	// ErrCompanyNotFound возвращается, когда компания не найдена
	ErrCompanyNotFound = errors.New("company not found")

	// ErrServiceNotFound возвращается, когда услуга не найдена
	ErrServiceNotFound = errors.New("service not found")

	// ErrCarNotFound возвращается, когда у пользователя нет выбранного автомобиля
	ErrCarNotFound = errors.New("user has no selected car")

	// ErrAccessDenied возвращается, когда у пользователя нет прав доступа
	ErrAccessDenied = errors.New("access denied")

	// ErrCannotCancel возвращается, когда бронирование не может быть отменено
	ErrCannotCancel = errors.New("booking cannot be cancelled")

	// ErrInvalidStatus возвращается при попытке установить недопустимый статус
	ErrInvalidStatus = errors.New("invalid booking status")

	// ErrInvalidInput возвращается при некорректных входных данных
	ErrInvalidInput = errors.New("invalid input data")

	// ErrInvalidBookingDate возвращается при некорректной дате бронирования
	ErrInvalidBookingDate = errors.New("invalid booking date")

	// ErrInvalidTimeRange возвращается при некорректном временном диапазоне
	ErrInvalidTimeRange = errors.New("invalid time range")

	// ErrInternal возвращается при внутренних ошибках сервиса
	ErrInternal = errors.New("service: internal error")
)

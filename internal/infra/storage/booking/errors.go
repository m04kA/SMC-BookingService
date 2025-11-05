package booking

import "errors"

var (
	// ErrBookingNotFound возвращается, когда бронирование не найдено
	ErrBookingNotFound = errors.New("booking.repository: booking not found")

	// ErrSlotNotAvailable возвращается, когда слот недоступен для бронирования
	ErrSlotNotAvailable = errors.New("booking.repository: slot not available")

	// ErrTransaction возвращается при ошибках работы с транзакцией
	ErrTransaction = errors.New("booking.repository: transaction error")

	// ErrBuildQuery возвращается при ошибке построения SQL запроса
	ErrBuildQuery = errors.New("booking.repository: failed to build query")

	// ErrExecQuery возвращается при ошибке выполнения SQL запроса
	ErrExecQuery = errors.New("booking.repository: failed to execute query")

	// ErrScanRow возвращается при ошибке сканирования результата запроса
	ErrScanRow = errors.New("booking.repository: failed to scan row")

	// ErrInvalidStatus возвращается при попытке установить недопустимый статус
	ErrInvalidStatus = errors.New("booking.repository: invalid booking status")

	// ErrCannotCancel возвращается, когда бронирование не может быть отменено
	ErrCannotCancel = errors.New("booking.repository: booking cannot be cancelled")
)

package userservice

import "errors"

var (
	// ErrCarNotFound возвращается, когда у пользователя нет выбранного автомобиля
	ErrCarNotFound = errors.New("user has no selected car")

	// ErrInternal возвращается при внутренних ошибках клиента
	ErrInternal = errors.New("userservice client: internal error")

	// ErrInvalidResponse возвращается при некорректном ответе от сервиса
	ErrInvalidResponse = errors.New("userservice client: invalid response")

	// ErrServiceDegraded возвращается при применении graceful degradation
	// Указывает, что UserService недоступен и следует использовать базовые цены
	ErrServiceDegraded = errors.New("userservice unavailable: graceful degradation applied")
)

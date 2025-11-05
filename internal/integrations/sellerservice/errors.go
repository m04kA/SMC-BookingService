package sellerservice

import "errors"

var (
	// ErrCompanyNotFound возвращается, когда компания не найдена
	ErrCompanyNotFound = errors.New("company not found")

	// ErrServiceNotFound возвращается, когда услуга не найдена
	ErrServiceNotFound = errors.New("service not found")

	// ErrInternal возвращается при внутренних ошибках клиента
	ErrInternal = errors.New("sellerservice client: internal error")

	// ErrInvalidResponse возвращается при некорректном ответе от сервиса
	ErrInvalidResponse = errors.New("sellerservice client: invalid response")
)

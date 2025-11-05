package config

import "errors"

var (
	// ErrConfigNotFound возвращается, когда конфигурация не найдена
	ErrConfigNotFound = errors.New("config not found")

	// ErrCompanyNotFound возвращается, когда компания не найдена
	ErrCompanyNotFound = errors.New("company not found")

	// ErrAddressNotFound возвращается, когда адрес не найден
	ErrAddressNotFound = errors.New("address not found")

	// ErrServiceNotFound возвращается, когда услуга не найдена
	ErrServiceNotFound = errors.New("service not found")

	// ErrAccessDenied возвращается, когда у пользователя нет прав доступа
	ErrAccessDenied = errors.New("access denied")

	// ErrInvalidInput возвращается при некорректных входных данных
	ErrInvalidInput = errors.New("invalid input data")

	// ErrConfigAlreadyExists возвращается при попытке создать дублирующую конфигурацию
	ErrConfigAlreadyExists = errors.New("config already exists")

	// ErrInternal возвращается при внутренних ошибках сервиса
	ErrInternal = errors.New("service: internal error")
)

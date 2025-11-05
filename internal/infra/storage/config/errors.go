package config

import "errors"

var (
	// ErrConfigNotFound возвращается, когда конфигурация не найдена
	ErrConfigNotFound = errors.New("config.repository: config not found")

	// ErrTransaction возвращается при ошибках работы с транзакцией
	ErrTransaction = errors.New("config.repository: transaction error")

	// ErrBuildQuery возвращается при ошибке построения SQL запроса
	ErrBuildQuery = errors.New("config.repository: failed to build query")

	// ErrExecQuery возвращается при ошибке выполнения SQL запроса
	ErrExecQuery = errors.New("config.repository: failed to execute query")

	// ErrScanRow возвращается при ошибке сканирования результата запроса
	ErrScanRow = errors.New("config.repository: failed to scan row")

	// ErrDuplicateConfig возвращается при попытке создать дубликат конфигурации
	ErrDuplicateConfig = errors.New("config.repository: duplicate config for company and service")

	// ErrInvalidSlotDuration возвращается при недопустимой длительности слота
	ErrInvalidSlotDuration = errors.New("config.repository: invalid slot duration")

	// ErrInvalidMaxConcurrent возвращается при недопустимом количестве параллельных бронирований
	ErrInvalidMaxConcurrent = errors.New("config.repository: invalid max concurrent bookings")
)

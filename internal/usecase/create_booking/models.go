package create_booking

import (
	"time"

	"github.com/m04kA/SMC-BookingService/pkg/types"
)

// Request модель запроса на создание бронирования
type Request struct {
	UserID    int64            // ID пользователя (Telegram ID)
	CompanyID int64            // ID компании
	AddressID int64            // ID адреса компании
	ServiceID int64            // ID услуги
	Date      time.Time        // Дата бронирования (без времени)
	StartTime types.TimeString // Время начала слота (например, "10:00")
	Notes     *string          // Дополнительные заметки (опционально)
}

// Response модель ответа с созданным бронированием
type Response struct {
	ID              int64            // ID созданного бронирования
	UserID          int64            // ID пользователя
	CompanyID       int64            // ID компании
	AddressID       int64            // ID адреса
	ServiceID       int64            // ID услуги
	CarID           int64            // ID автомобиля
	BookingDate     time.Time        // Дата бронирования
	StartTime       types.TimeString // Время начала
	DurationMinutes int              // Длительность в минутах
	Status          string           // Статус бронирования

	// Денормализованные данные
	ServiceName     string  // Название услуги
	ServicePrice    float64 // Цена услуги
	CarBrand        *string // Марка автомобиля
	CarModel        *string // Модель автомобиля
	CarLicensePlate *string // Госномер
	Notes           *string // Заметки

	CreatedAt time.Time // Время создания
	UpdatedAt time.Time // Время обновления
}

package get_available_slots

import (
	"time"

	"github.com/m04kA/SMC-BookingService/pkg/types"
)

// Request модель запроса на получение доступных слотов
type Request struct {
	UserID    int64     // ID пользователя (для логирования, не влияет на результат)
	CompanyID int64     // ID компании
	AddressID int64     // ID адреса компании
	ServiceID int64     // ID услуги
	Date      time.Time // Дата для получения слотов (без времени)
}

// Response модель ответа со списком доступных слотов
type Response struct {
	Date      time.Time // Дата, на которую запрашивались слоты
	CompanyID int64     // ID компании
	AddressID int64     // ID адреса
	ServiceID int64     // ID услуги
	Slots     []Slot    // Список доступных слотов
}

// Slot модель временного слота
type Slot struct {
	StartTime       types.TimeString // Время начала слота (например, "10:00")
	DurationMinutes int              // Длительность слота в минутах
	AvailableSpots  int              // Количество свободных мест
	TotalSpots      int              // Общее количество мест
}

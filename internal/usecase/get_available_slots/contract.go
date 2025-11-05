package get_available_slots

import (
	"context"
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
)

// BookingRepository интерфейс репозитория бронирований
type BookingRepository interface {
	// GetByCompanyAndDateAndAddress получает все бронирования компании на конкретную дату и адрес
	GetByCompanyWithFilter(ctx context.Context, filter domain.CompanyBookingsFilter) ([]*domain.Booking, error)
}

// ConfigRepository интерфейс репозитория конфигурации слотов
type ConfigRepository interface {
	// GetConfigWithHierarchy получает конфигурацию с учетом иерархии приоритетов
	GetConfigWithHierarchy(ctx context.Context, companyID int64, addressID *int64, serviceID *int64) (*domain.CompanySlotsConfig, error)
}

// SellerServiceClient интерфейс клиента для SellerService
type SellerServiceClient interface {
	GetCompany(ctx context.Context, companyID int64) (*sellerservice.Company, error)
	GetService(ctx context.Context, companyID, serviceID int64) (*sellerservice.Service, error)
}

// TimeProvider интерфейс для получения текущего времени (для тестирования)
type TimeProvider interface {
	Now() time.Time
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

// RealTimeProvider реальный провайдер времени для production
type RealTimeProvider struct{}

// Now возвращает текущее время
func (p *RealTimeProvider) Now() time.Time {
	return time.Now()
}

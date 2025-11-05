package bookings

import (
	"context"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	"github.com/m04kA/SMC-BookingService/internal/integrations/userservice"
)

// BookingRepository интерфейс репозитория бронирований
type BookingRepository interface {
	Create(ctx context.Context, booking *domain.Booking) (*domain.Booking, error)
	GetByID(ctx context.Context, id int64) (*domain.Booking, error)
	GetByUserID(ctx context.Context, userID int64, status *domain.BookingStatus) ([]*domain.Booking, error)
	GetByCompanyWithFilter(ctx context.Context, filter domain.CompanyBookingsFilter) ([]*domain.Booking, error)
	UpdateStatus(ctx context.Context, id int64, status domain.BookingStatus) error
	Cancel(ctx context.Context, id int64, status domain.BookingStatus, reason string) error
}

// ConfigRepository интерфейс репозитория конфигурации слотов
type ConfigRepository interface {
	GetByCompanyAndService(ctx context.Context, companyID int64, serviceID *int64) (*domain.CompanySlotsConfig, error)
	GetAllByCompany(ctx context.Context, companyID int64) ([]*domain.CompanySlotsConfig, error)
}

// UserServiceClient интерфейс клиента для UserService
type UserServiceClient interface {
	GetSelectedCar(ctx context.Context, tgUserID int64) (*userservice.Car, error)
}

// SellerServiceClient интерфейс клиента для SellerService
type SellerServiceClient interface {
	GetCompany(ctx context.Context, companyID int64) (*sellerservice.Company, error)
	GetService(ctx context.Context, companyID, serviceID int64) (*sellerservice.Service, error)
}

// TransactionManager интерфейс для управления транзакциями
type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
	DoSerializable(ctx context.Context, fn func(ctx context.Context) error) error
	DoReadOnly(ctx context.Context, fn func(ctx context.Context) error) error
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

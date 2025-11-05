package config

import (
	"context"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
)

// ConfigRepository интерфейс репозитория конфигурации слотов
type ConfigRepository interface {
	Create(ctx context.Context, config *domain.CompanySlotsConfig) (*domain.CompanySlotsConfig, error)
	GetByID(ctx context.Context, id int64) (*domain.CompanySlotsConfig, error)
	GetByCompanyAddressAndService(ctx context.Context, companyID int64, addressID *int64, serviceID *int64) (*domain.CompanySlotsConfig, error)
	GetConfigWithHierarchy(ctx context.Context, companyID int64, addressID *int64, serviceID *int64) (*domain.CompanySlotsConfig, error)
	GetAllByCompany(ctx context.Context, companyID int64) ([]*domain.CompanySlotsConfig, error)
	Update(ctx context.Context, id int64, config *domain.CompanySlotsConfig) (*domain.CompanySlotsConfig, error)
	Delete(ctx context.Context, id int64) error
	DeleteByCompanyAddressAndService(ctx context.Context, companyID int64, addressID *int64, serviceID *int64) error
}

// SellerServiceClient интерфейс клиента для SellerService
type SellerServiceClient interface {
	GetCompany(ctx context.Context, companyID int64) (*sellerservice.Company, error)
	GetService(ctx context.Context, companyID, serviceID int64) (*sellerservice.Service, error)
}

// Logger интерфейс для логирования
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

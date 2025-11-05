package update_company_config

import (
	"context"

	"github.com/m04kA/SMC-BookingService/internal/service/config/models"
)

type ConfigService interface {
	GetWithHierarchy(ctx context.Context, req *models.GetConfigRequest) (*models.ConfigResponse, error)
	Update(ctx context.Context, id int64, req *models.UpdateConfigRequest) (*models.ConfigResponse, error)
}

type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

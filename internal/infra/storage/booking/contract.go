package booking

import (
	"context"
	"database/sql"

	"github.com/m04kA/SMC-BookingService/pkg/dbmetrics"
)

// Переиспользуем интерфейсы из dbmetrics для работы с БД
type DBExecutor = dbmetrics.DBExecutor
type TxExecutor = dbmetrics.TxExecutor

// TxBeginner интерфейс для начала транзакций
// Поддерживает *sql.DB и *dbmetrics.DB
type TxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (TxExecutor, error)
}

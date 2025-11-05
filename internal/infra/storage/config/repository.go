package config

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/pkg/dbmetrics"
	"github.com/m04kA/SMC-BookingService/pkg/psqlbuilder"
)

// Repository репозиторий для работы с конфигурацией слотов
type Repository struct {
	db DBExecutor
}

// NewRepository создает новый экземпляр репозитория конфигурации слотов
func NewRepository(db DBExecutor) *Repository {
	return &Repository{db: db}
}

// Create создает новую конфигурацию слотов
// Если в контексте передана активная транзакция, использует её
func (r *Repository) Create(ctx context.Context, config *domain.CompanySlotsConfig) (*domain.CompanySlotsConfig, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Insert("company_slots_config").
		Columns(
			"company_id",
			"address_id",
			"service_id",
			"slot_duration_minutes",
			"max_concurrent_bookings",
			"advance_booking_days",
			"min_booking_notice_minutes",
		).
		Values(
			config.CompanyID,
			config.AddressID,
			config.ServiceID,
			config.SlotDurationMinutes,
			config.MaxConcurrentBookings,
			config.AdvanceBookingDays,
			config.MinBookingNoticeMinutes,
		).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: Create - build insert query: %v", ErrBuildQuery, err)
	}

	var createdAt, updatedAt sql.NullTime
	err = executor.QueryRowContext(ctx, query, args...).Scan(
		&config.ID,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("%w: Create - execute insert: %v", ErrExecQuery, err)
	}

	config.CreatedAt = createdAt.Time
	config.UpdatedAt = updatedAt.Time

	return config, nil
}

// GetByID получает конфигурацию по ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*domain.CompanySlotsConfig, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Select(
		"id",
		"company_id",
		"address_id",
		"service_id",
		"slot_duration_minutes",
		"max_concurrent_bookings",
		"advance_booking_days",
		"min_booking_notice_minutes",
		"created_at",
		"updated_at",
	).
		From("company_slots_config").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetByID - build select query: %v", ErrBuildQuery, err)
	}

	var config domain.CompanySlotsConfig
	var createdAt, updatedAt sql.NullTime

	err = executor.QueryRowContext(ctx, query, args...).Scan(
		&config.ID,
		&config.CompanyID,
		&config.AddressID,
		&config.ServiceID,
		&config.SlotDurationMinutes,
		&config.MaxConcurrentBookings,
		&config.AdvanceBookingDays,
		&config.MinBookingNoticeMinutes,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: GetByID - scan config: %v", ErrScanRow, err)
	}

	config.CreatedAt = createdAt.Time
	config.UpdatedAt = updatedAt.Time

	return &config, nil
}

// GetByCompanyAddressAndService получает конфигурацию для компании, адреса и услуги
// Поддерживает иерархическую систему конфигурации:
// 1. Если addressID и serviceID заданы - ищет конфигурацию для конкретной услуги на конкретном адресе
// 2. Если только addressID задан - ищет конфигурацию для всех услуг на конкретном адресе
// 3. Если только serviceID задан - ищет конфигурацию для конкретной услуги на всех адресах
// 4. Если оба nil - ищет глобальную конфигурацию компании
func (r *Repository) GetByCompanyAddressAndService(ctx context.Context, companyID int64, addressID *int64, serviceID *int64) (*domain.CompanySlotsConfig, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	selectBuilder := psqlbuilder.Select(
		"id",
		"company_id",
		"address_id",
		"service_id",
		"slot_duration_minutes",
		"max_concurrent_bookings",
		"advance_booking_days",
		"min_booking_notice_minutes",
		"created_at",
		"updated_at",
	).
		From("company_slots_config").
		Where(squirrel.Eq{"company_id": companyID})

	// Фильтрация по address_id (NULL или конкретное значение)
	if addressID == nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"address_id": nil})
	} else {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"address_id": *addressID})
	}

	// Фильтрация по service_id (NULL или конкретное значение)
	if serviceID == nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"service_id": nil})
	} else {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"service_id": *serviceID})
	}

	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: GetByCompanyAddressAndService - build select query: %v", ErrBuildQuery, err)
	}

	var config domain.CompanySlotsConfig
	var createdAt, updatedAt sql.NullTime

	err = executor.QueryRowContext(ctx, query, args...).Scan(
		&config.ID,
		&config.CompanyID,
		&config.AddressID,
		&config.ServiceID,
		&config.SlotDurationMinutes,
		&config.MaxConcurrentBookings,
		&config.AdvanceBookingDays,
		&config.MinBookingNoticeMinutes,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: GetByCompanyAddressAndService - scan config: %v", ErrScanRow, err)
	}

	config.CreatedAt = createdAt.Time
	config.UpdatedAt = updatedAt.Time

	return &config, nil
}

// GetConfigWithHierarchy получает конфигурацию с учетом иерархии приоритетов
// Приоритет применения конфигурации:
// 1. Конфигурация для конкретной услуги на конкретном адресе (addressID, serviceID)
// 2. Конфигурация для всех услуг на конкретном адресе (addressID, NULL)
// 3. Конфигурация для конкретной услуги на всех адресах (NULL, serviceID)
// 4. Глобальная конфигурация компании (NULL, NULL)
//
// Если конфигурация не найдена ни на одном уровне, возвращает ErrConfigNotFound
func (r *Repository) GetConfigWithHierarchy(ctx context.Context, companyID int64, addressID *int64, serviceID *int64) (*domain.CompanySlotsConfig, error) {
	// 1. Пробуем получить конфигурацию для конкретной услуги на конкретном адресе (если оба указаны)
	if addressID != nil && serviceID != nil {
		config, err := r.GetByCompanyAddressAndService(ctx, companyID, addressID, serviceID)
		if err == nil {
			return config, nil
		}
		if err != ErrConfigNotFound {
			return nil, fmt.Errorf("%w: GetConfigWithHierarchy - level 1 (address+service): %v", ErrExecQuery, err)
		}
	}

	// 2. Пробуем получить конфигурацию для всех услуг на конкретном адресе (если адрес указан)
	if addressID != nil {
		config, err := r.GetByCompanyAddressAndService(ctx, companyID, addressID, nil)
		if err == nil {
			return config, nil
		}
		if err != ErrConfigNotFound {
			return nil, fmt.Errorf("%w: GetConfigWithHierarchy - level 2 (address only): %v", ErrExecQuery, err)
		}
	}

	// 3. Пробуем получить конфигурацию для конкретной услуги на всех адресах (если услуга указана)
	if serviceID != nil {
		config, err := r.GetByCompanyAddressAndService(ctx, companyID, nil, serviceID)
		if err == nil {
			return config, nil
		}
		if err != ErrConfigNotFound {
			return nil, fmt.Errorf("%w: GetConfigWithHierarchy - level 3 (service only): %v", ErrExecQuery, err)
		}
	}

	// 4. Пробуем получить глобальную конфигурацию компании
	config, err := r.GetByCompanyAddressAndService(ctx, companyID, nil, nil)
	if err == nil {
		return config, nil
	}
	if err != ErrConfigNotFound {
		return nil, fmt.Errorf("%w: GetConfigWithHierarchy - level 4 (global): %v", ErrExecQuery, err)
	}

	// Если конфигурация не найдена ни на одном уровне
	return nil, ErrConfigNotFound
}

// GetAllByCompany получает все конфигурации компании (глобальную, для адресов и услуг)
func (r *Repository) GetAllByCompany(ctx context.Context, companyID int64) ([]*domain.CompanySlotsConfig, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Select(
		"id",
		"company_id",
		"address_id",
		"service_id",
		"slot_duration_minutes",
		"max_concurrent_bookings",
		"advance_booking_days",
		"min_booking_notice_minutes",
		"created_at",
		"updated_at",
	).
		From("company_slots_config").
		Where(squirrel.Eq{"company_id": companyID}).
		OrderBy("address_id ASC NULLS FIRST, service_id ASC NULLS FIRST"). // Глобальная конфигурация первой
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetAllByCompany - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: GetAllByCompany - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	configs := make([]*domain.CompanySlotsConfig, 0)

	for rows.Next() {
		var config domain.CompanySlotsConfig
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&config.ID,
			&config.CompanyID,
			&config.AddressID,
			&config.ServiceID,
			&config.SlotDurationMinutes,
			&config.MaxConcurrentBookings,
			&config.AdvanceBookingDays,
			&config.MinBookingNoticeMinutes,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("%w: GetAllByCompany - scan row: %v", ErrScanRow, err)
		}

		config.CreatedAt = createdAt.Time
		config.UpdatedAt = updatedAt.Time

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: GetAllByCompany - rows error: %v", ErrScanRow, err)
	}

	return configs, nil
}

// Update обновляет конфигурацию слотов
func (r *Repository) Update(ctx context.Context, id int64, config *domain.CompanySlotsConfig) (*domain.CompanySlotsConfig, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("company_slots_config").
		Set("slot_duration_minutes", config.SlotDurationMinutes).
		Set("max_concurrent_bookings", config.MaxConcurrentBookings).
		Set("advance_booking_days", config.AdvanceBookingDays).
		Set("min_booking_notice_minutes", config.MinBookingNoticeMinutes).
		Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING created_at, updated_at").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: Update - build update query: %v", ErrBuildQuery, err)
	}

	var createdAt, updatedAt sql.NullTime
	err = executor.QueryRowContext(ctx, query, args...).Scan(&createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: Update - execute update: %v", ErrExecQuery, err)
	}

	config.ID = id
	config.CreatedAt = createdAt.Time
	config.UpdatedAt = updatedAt.Time

	return config, nil
}

// Delete удаляет конфигурацию слотов
func (r *Repository) Delete(ctx context.Context, id int64) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Delete("company_slots_config").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("%w: Delete - build delete query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: Delete - execute delete: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: Delete - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrConfigNotFound
	}

	return nil
}

// DeleteByCompanyAddressAndService удаляет конфигурацию по company_id, address_id и service_id
func (r *Repository) DeleteByCompanyAddressAndService(ctx context.Context, companyID int64, addressID *int64, serviceID *int64) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	deleteBuilder := psqlbuilder.Delete("company_slots_config").
		Where(squirrel.Eq{"company_id": companyID})

	// Фильтрация по address_id (NULL или конкретное значение)
	if addressID == nil {
		deleteBuilder = deleteBuilder.Where(squirrel.Eq{"address_id": nil})
	} else {
		deleteBuilder = deleteBuilder.Where(squirrel.Eq{"address_id": *addressID})
	}

	// Фильтрация по service_id (NULL или конкретное значение)
	if serviceID == nil {
		deleteBuilder = deleteBuilder.Where(squirrel.Eq{"service_id": nil})
	} else {
		deleteBuilder = deleteBuilder.Where(squirrel.Eq{"service_id": *serviceID})
	}

	query, args, err := deleteBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("%w: DeleteByCompanyAddressAndService - build delete query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: DeleteByCompanyAddressAndService - execute delete: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: DeleteByCompanyAddressAndService - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrConfigNotFound
	}

	return nil
}

// Helper methods

// BeginTx начинает новую транзакцию и возвращает контекст с ней
func (r *Repository) BeginTx(ctx context.Context, opts *sql.TxOptions) (context.Context, TxExecutor, error) {
	// Пытаемся привести к TxBeginner интерфейсу (dbmetrics.DB реализует этот интерфейс)
	if txBeginner, ok := r.db.(TxBeginner); ok {
		tx, err := txBeginner.BeginTx(ctx, opts)
		if err != nil {
			return ctx, nil, fmt.Errorf("%w: BeginTx: %v", ErrTransaction, err)
		}
		return dbmetrics.WithTx(ctx, tx), tx, nil
	}

	// Fallback для обычного *sql.DB
	if db, ok := r.db.(*sql.DB); ok {
		tx, err := db.BeginTx(ctx, opts)
		if err != nil {
			return ctx, nil, fmt.Errorf("%w: BeginTx: %v", ErrTransaction, err)
		}
		wrappedTx := &dbmetrics.SqlTxWrapper{Tx: tx}
		return dbmetrics.WithTx(ctx, wrappedTx), wrappedTx, nil
	}

	return ctx, nil, fmt.Errorf("%w: db type not supported", ErrTransaction)
}

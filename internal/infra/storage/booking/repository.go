package booking

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/pkg/dbmetrics"
	"github.com/m04kA/SMC-BookingService/pkg/psqlbuilder"
)

// Repository репозиторий для работы с бронированиями
type Repository struct {
	db DBExecutor
}

// NewRepository создает новый экземпляр репозитория бронирований
func NewRepository(db DBExecutor) *Repository {
	return &Repository{db: db}
}

// Create создает новое бронирование
// Если в контексте передана активная транзакция (через context.Value), использует её.
// Иначе выполняет обычный запрос без транзакции.
//
// Когда использовать транзакцию:
// - При создании бронирования с проверкой доступности слота (для предотвращения race condition)
// - При пакетном создании нескольких бронирований
// - При создании бронирования с обновлением связанных данных
//
// Когда можно без транзакции:
// - При простом создании бронирования без дополнительных проверок
// - При импорте данных (если не критична консистентность в моменте)
func (r *Repository) Create(ctx context.Context, booking *domain.Booking) (*domain.Booking, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Insert("bookings").
		Columns(
			"user_id",
			"company_id",
			"address_id",
			"service_id",
			"car_id",
			"booking_date",
			"start_time",
			"duration_minutes",
			"status",
			"service_name",
			"service_price",
			"car_brand",
			"car_model",
			"car_license_plate",
			"notes",
		).
		Values(
			booking.UserID,
			booking.CompanyID,
			booking.AddressID,
			booking.ServiceID,
			booking.CarID,
			booking.BookingDate,
			booking.StartTime,
			booking.DurationMinutes,
			booking.Status,
			booking.ServiceName,
			booking.ServicePrice,
			booking.CarBrand,
			booking.CarModel,
			booking.CarLicensePlate,
			booking.Notes,
		).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: Create - build insert query: %v", ErrBuildQuery, err)
	}

	var createdAt, updatedAt sql.NullTime
	err = executor.QueryRowContext(ctx, query, args...).Scan(
		&booking.ID,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("%w: Create - execute insert: %v", ErrExecQuery, err)
	}

	booking.CreatedAt = createdAt.Time
	booking.UpdatedAt = updatedAt.Time

	return booking, nil
}

// GetByID получает бронирование по ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Select(
		"id",
		"user_id",
		"company_id",
		"address_id",
		"service_id",
		"car_id",
		"booking_date",
		"start_time",
		"duration_minutes",
		"status",
		"service_name",
		"service_price",
		"car_brand",
		"car_model",
		"car_license_plate",
		"notes",
		"cancellation_reason",
		"cancelled_at",
		"created_at",
		"updated_at",
	).
		From("bookings").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetByID - build select query: %v", ErrBuildQuery, err)
	}

	var booking domain.Booking
	var createdAt, updatedAt sql.NullTime

	err = executor.QueryRowContext(ctx, query, args...).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.CompanyID,
		&booking.AddressID,
		&booking.ServiceID,
		&booking.CarID,
		&booking.BookingDate,
		&booking.StartTime,
		&booking.DurationMinutes,
		&booking.Status,
		&booking.ServiceName,
		&booking.ServicePrice,
		&booking.CarBrand,
		&booking.CarModel,
		&booking.CarLicensePlate,
		&booking.Notes,
		&booking.CancellationReason,
		&booking.CancelledAt,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: GetByID - scan booking: %v", ErrScanRow, err)
	}

	booking.CreatedAt = createdAt.Time
	booking.UpdatedAt = updatedAt.Time

	return &booking, nil
}

// GetByUserID получает список бронирований пользователя
// Опционально фильтрует по статусу
func (r *Repository) GetByUserID(ctx context.Context, userID int64, status *domain.BookingStatus) ([]*domain.Booking, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	selectBuilder := psqlbuilder.Select(
		"id",
		"user_id",
		"company_id",
		"address_id",
		"service_id",
		"car_id",
		"booking_date",
		"start_time",
		"duration_minutes",
		"status",
		"service_name",
		"service_price",
		"car_brand",
		"car_model",
		"car_license_plate",
		"notes",
		"cancellation_reason",
		"cancelled_at",
		"created_at",
		"updated_at",
	).
		From("bookings").
		Where(squirrel.Eq{"user_id": userID}).
		OrderBy("booking_date DESC, start_time DESC")

	// Фильтрация по статусу, если указан
	if status != nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"status": *status})
	}

	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: GetByUserID - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: GetByUserID - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	return r.scanBookings(rows)
}

// GetByCompanyWithFilter получает бронирования компании с гибкой фильтрацией
// Поддерживает фильтрацию по:
// - Периоду (StartDate, EndDate) - опционально
// - Статусу (Status) - опционально
// - Включению неактивных бронирований (IncludeInactive)
//
// Примеры использования:
//
// 1. Все активные бронирования компании:
//    filter := domain.CompanyBookingsFilter{CompanyID: 123}
//
// 2. Бронирования на конкретную дату:
//    date := time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC)
//    filter := domain.CompanyBookingsFilter{CompanyID: 123, StartDate: &date, EndDate: &date}
//
// 3. Бронирования за период:
//    start := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
//    end := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)
//    filter := domain.CompanyBookingsFilter{CompanyID: 123, StartDate: &start, EndDate: &end}
//
// 4. Только подтвержденные бронирования:
//    status := domain.StatusConfirmed
//    filter := domain.CompanyBookingsFilter{CompanyID: 123, Status: &status}
//
// 5. Все бронирования включая отменённые:
//    filter := domain.CompanyBookingsFilter{CompanyID: 123, IncludeInactive: true}
func (r *Repository) GetByCompanyWithFilter(ctx context.Context, filter domain.CompanyBookingsFilter) ([]*domain.Booking, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	selectBuilder := psqlbuilder.Select(
		"id",
		"user_id",
		"company_id",
		"address_id",
		"service_id",
		"car_id",
		"booking_date",
		"start_time",
		"duration_minutes",
		"status",
		"service_name",
		"service_price",
		"car_brand",
		"car_model",
		"car_license_plate",
		"notes",
		"cancellation_reason",
		"cancelled_at",
		"created_at",
		"updated_at",
	).
		From("bookings").
		Where(squirrel.Eq{"company_id": filter.CompanyID})

	// Фильтрация по адресу (если указан)
	if filter.AddressID != nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"address_id": *filter.AddressID})
	}

	// Фильтрация по периоду
	if filter.StartDate != nil {
		selectBuilder = selectBuilder.Where(squirrel.GtOrEq{"booking_date": *filter.StartDate})
	}
	if filter.EndDate != nil {
		selectBuilder = selectBuilder.Where(squirrel.LtOrEq{"booking_date": *filter.EndDate})
	}

	// Фильтрация по статусу
	if filter.Status != nil {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"status": *filter.Status})
	} else if !filter.IncludeInactive {
		// Если не указан конкретный статус и не нужны неактивные - исключаем их
		inactiveStatusStrings := make([]string, len(domain.InactiveStatuses))
		for i, s := range domain.InactiveStatuses {
			inactiveStatusStrings[i] = string(s)
		}
		selectBuilder = selectBuilder.Where(squirrel.NotEq{"status": inactiveStatusStrings})
	}

	// Определяем сортировку в зависимости от фильтра
	if filter.StartDate != nil && filter.EndDate != nil && filter.StartDate.Equal(*filter.EndDate) {
		// Для конкретной даты сортируем по времени начала (ASC)
		selectBuilder = selectBuilder.OrderBy("start_time ASC")
	} else {
		// Для периода или всех бронирований сортируем по дате и времени (DESC - сначала новые)
		selectBuilder = selectBuilder.OrderBy("booking_date DESC, start_time DESC")
	}

	// Если используется транзакция, добавляем FOR UPDATE для блокировки
	// (только для конкретной даты - для usecase создания бронирования)
	if dbmetrics.IsInTransaction(ctx) && filter.StartDate != nil && filter.EndDate != nil && filter.StartDate.Equal(*filter.EndDate) {
		selectBuilder = selectBuilder.Suffix("FOR UPDATE")
	}

	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: GetByCompanyWithFilter - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: GetByCompanyWithFilter - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	return r.scanBookings(rows)
}

// GetUserIDsByCompanyID получает список всех пользователей, которые когда-либо бронировали услуги компании
// Используется для рассылки уведомлений, аналитики и маркетинга
func (r *Repository) GetUserIDsByCompanyID(ctx context.Context, companyID int64) ([]int64, error) {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Select("DISTINCT user_id").
		From("bookings").
		Where(squirrel.Eq{"company_id": companyID}).
		OrderBy("user_id ASC").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%w: GetUserIDsByCompanyID - build select query: %v", ErrBuildQuery, err)
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: GetUserIDsByCompanyID - execute query: %v", ErrExecQuery, err)
	}
	defer rows.Close()

	userIDs := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("%w: GetUserIDsByCompanyID - scan user_id: %v", ErrScanRow, err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: GetUserIDsByCompanyID - rows error: %v", ErrScanRow, err)
	}

	return userIDs, nil
}

// UpdateStatus обновляет статус бронирования
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status domain.BookingStatus) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("bookings").
		Set("status", status).
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("%w: UpdateStatus - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: UpdateStatus - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: UpdateStatus - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrBookingNotFound
	}

	return nil
}

// Cancel отменяет бронирование с указанием причины
func (r *Repository) Cancel(ctx context.Context, id int64, status domain.BookingStatus, reason string) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Update("bookings").
		Set("status", status).
		Set("cancellation_reason", reason).
		Set("cancelled_at", "NOW()").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("%w: Cancel - build update query: %v", ErrBuildQuery, err)
	}

	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: Cancel - execute update: %v", ErrExecQuery, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: Cancel - get rows affected: %v", ErrExecQuery, err)
	}

	if rowsAffected == 0 {
		return ErrBookingNotFound
	}

	return nil
}

// Delete удаляет бронирование (физическое удаление, использовать осторожно)
// Рекомендуется использовать Cancel вместо физического удаления для сохранения истории
func (r *Repository) Delete(ctx context.Context, id int64) error {
	executor := dbmetrics.GetExecutor(ctx, r.db)

	query, args, err := psqlbuilder.Delete("bookings").
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
		return ErrBookingNotFound
	}

	return nil
}

// scanBookings сканирует результаты запроса в слайс бронирований
func (r *Repository) scanBookings(rows *sql.Rows) ([]*domain.Booking, error) {
	bookings := make([]*domain.Booking, 0)

	for rows.Next() {
		var booking domain.Booking
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.CompanyID,
			&booking.AddressID,
			&booking.ServiceID,
			&booking.CarID,
			&booking.BookingDate,
			&booking.StartTime,
			&booking.DurationMinutes,
			&booking.Status,
			&booking.ServiceName,
			&booking.ServicePrice,
			&booking.CarBrand,
			&booking.CarModel,
			&booking.CarLicensePlate,
			&booking.Notes,
			&booking.CancellationReason,
			&booking.CancelledAt,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("%w: scanBookings - scan row: %v", ErrScanRow, err)
		}

		booking.CreatedAt = createdAt.Time
		booking.UpdatedAt = updatedAt.Time

		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: scanBookings - rows error: %v", ErrScanRow, err)
	}

	return bookings, nil
}

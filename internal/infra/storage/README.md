# Storage Layer - Repository Pattern

Слой репозиториев для работы с базой данных. Реализован в соответствии с принципами Clean Architecture.

## Структура

```
storage/
├── booking/           # Репозиторий для бронирований
│   ├── contract.go   # Внешние зависимости (DBExecutor, TxExecutor)
│   ├── repository.go # CRUD методы
│   └── errors.go     # Специфичные ошибки
└── config/           # Репозиторий для конфигурации слотов
    ├── contract.go   # Внешние зависимости
    ├── repository.go # CRUD методы
    └── errors.go     # Специфичные ошибки
```

## Принципы

### 1. Без бизнес-логики
Repository слой содержит **только** CRUD операции. Вся бизнес-логика вынесена в service слой.

**Правильно:**
```go
// Repository - только сохранение данных
func (r *Repository) Create(ctx context.Context, booking *domain.Booking) (*domain.Booking, error) {
    query, args, _ := psqlbuilder.Insert("bookings").
        Columns("user_id", "company_id", ...).
        Values(booking.UserID, booking.CompanyID, ...).
        ToSql()

    err := executor.QueryRowContext(ctx, query, args...).Scan(&booking.ID)
    return booking, err
}
```

**Неправильно:**
```go
// Бизнес-логика в repository - НЕ ДЕЛАТЬ ТАК!
func (r *Repository) Create(ctx context.Context, booking *domain.Booking) (*domain.Booking, error) {
    // Валидация - это бизнес-логика, должна быть в service!
    if booking.StartTime.IsInPast() {
        return nil, errors.New("cannot book in the past")
    }

    // Проверка доступности - это бизнес-логика, должна быть в service!
    available := r.checkAvailability(...)
    if !available {
        return nil, errors.New("slot not available")
    }

    // ...
}
```

### 2. Работа с транзакциями через контекст

Все методы репозитория автоматически определяют, используется ли транзакция через контекст:

```go
// Без транзакции - обычный запрос
booking, err := repo.Create(ctx, booking)

// С транзакцией - автоматически используется из контекста
ctx, tx, err := repo.BeginTx(ctx, nil)
defer tx.Rollback()

booking, err := repo.Create(ctx, booking)  // Использует транзакцию!
bookings, err := repo.GetByCompanyAndDate(ctx, companyID, date)  // Добавляет FOR UPDATE!

if err := tx.Commit(); err != nil {
    return err
}
```

**Как это работает:**
1. `BeginTx` добавляет транзакцию в контекст через `dbmetrics.WithTx`
2. Все методы используют `dbmetrics.GetExecutor(ctx, r.db)` для получения executor'а
3. Если в контексте есть транзакция - используется она, иначе - обычный db
4. При наличии транзакции автоматически добавляется `FOR UPDATE` где нужно

### 3. Dependency Injection через интерфейсы

Все зависимости описаны через интерфейсы в `contract.go`:

```go
// contract.go
type DBExecutor = dbmetrics.DBExecutor   // Интерфейс для выполнения запросов
type TxExecutor = dbmetrics.TxExecutor   // Интерфейс для транзакций

type TxBeginner interface {
    BeginTx(ctx context.Context, opts *sql.TxOptions) (TxExecutor, error)
}
```

Это позволяет:
- Легко мокать зависимости в тестах
- Переиспользовать интерфейсы из `pkg/dbmetrics`
- Автоматически собирать метрики через wrapper

### 4. Специфичные ошибки для каждого слоя

Каждый репозиторий имеет свои ошибки с понятным префиксом:

```go
// booking/errors.go
var ErrBookingNotFound = errors.New("booking.repository: booking not found")
var ErrSlotNotAvailable = errors.New("booking.repository: slot not available")

// config/errors.go
var ErrConfigNotFound = errors.New("config.repository: config not found")
```

**Префиксы:**
- `booking.repository:` - ошибки booking репозитория
- `config.repository:` - ошибки config репозитория
- `booking.service:` - будет использоваться в service слое
- `booking.handler:` - будет использоваться в handler слое

Это помогает при логировании и отладке понять, на каком уровне произошла ошибка.

## Booking Repository

### Методы

#### Create
```go
func (r *Repository) Create(ctx context.Context, booking *domain.Booking) (*domain.Booking, error)
```

Создает новое бронирование. Если в контексте есть транзакция - использует её.

**Когда использовать транзакцию:**
- При создании бронирования с проверкой доступности слота (для предотвращения race condition)
- При пакетном создании нескольких бронирований
- При создании бронирования с обновлением связанных данных

**Когда можно без транзакции:**
- При простом создании бронирования без дополнительных проверок
- При импорте данных (если не критична консистентность в моменте)

#### GetByID
```go
func (r *Repository) GetByID(ctx context.Context, id int64) (*domain.Booking, error)
```

Получает бронирование по ID. Возвращает `ErrBookingNotFound` если не найдено.

#### GetByUserID
```go
func (r *Repository) GetByUserID(ctx context.Context, userID int64, status *domain.BookingStatus) ([]*domain.Booking, error)
```

Получает список бронирований пользователя. Опционально фильтрует по статусу.

**Примеры:**
```go
// Все бронирования пользователя
bookings, _ := repo.GetByUserID(ctx, userID, nil)

// Только подтвержденные
status := domain.StatusConfirmed
bookings, _ := repo.GetByUserID(ctx, userID, &status)
```

#### GetByCompanyAndDate
```go
func (r *Repository) GetByCompanyAndDate(ctx context.Context, companyID int64, date time.Time) ([]*domain.Booking, error)
```

Получает все **активные** бронирования компании на определенную дату.
Автоматически фильтрует неактивные статусы (`InactiveStatuses`).

**Важно:** Если в контексте есть транзакция, автоматически добавляет `FOR UPDATE` для блокировки строк.

**Использование:**
```go
// Без блокировки - для просмотра
bookings, _ := repo.GetByCompanyAndDate(ctx, companyID, date)

// С блокировкой - при создании нового бронирования
ctx, tx, _ := repo.BeginTx(ctx, nil)
bookings, _ := repo.GetByCompanyAndDate(ctx, companyID, date)  // Добавляет FOR UPDATE!
```

#### GetUserIDsByCompanyID
```go
func (r *Repository) GetUserIDsByCompanyID(ctx context.Context, companyID int64) ([]int64, error)
```

Получает список всех пользователей, которые когда-либо бронировали услуги компании.

**Применение:**
- Рассылка уведомлений о новых акциях
- Аналитика и маркетинг
- Формирование базы клиентов

#### UpdateStatus
```go
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status domain.BookingStatus) error
```

Обновляет только статус бронирования.

#### Cancel
```go
func (r *Repository) Cancel(ctx context.Context, id int64, status domain.BookingStatus, reason string) error
```

Отменяет бронирование с указанием причины и временем отмены.

**Устанавливает:**
- `status` - новый статус (обычно `StatusCancelledByUser` или `StatusCancelledByCompany`)
- `cancellation_reason` - причина отмены
- `cancelled_at` - время отмены (NOW())

#### Delete
```go
func (r *Repository) Delete(ctx context.Context, id int64) error
```

**Физическое удаление** бронирования. Использовать осторожно!

**Рекомендация:** Используйте `Cancel` вместо `Delete` для сохранения истории.

#### BeginTx
```go
func (r *Repository) BeginTx(ctx context.Context, opts *sql.TxOptions) (context.Context, TxExecutor, error)
```

Начинает транзакцию и возвращает новый контекст с ней.

**Использование:**
```go
ctx, tx, err := repo.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
})
if err != nil {
    return err
}
defer tx.Rollback()

// Все методы теперь используют транзакцию
booking, _ := repo.Create(ctx, booking)
bookings, _ := repo.GetByCompanyAndDate(ctx, companyID, date)

if err := tx.Commit(); err != nil {
    return err
}
```

## Config Repository

### Методы

#### Create
```go
func (r *Repository) Create(ctx context.Context, config *domain.CompanySlotsConfig) (*domain.CompanySlotsConfig, error)
```

Создает новую конфигурацию слотов.

#### GetByID
```go
func (r *Repository) GetByID(ctx context.Context, id int64) (*domain.CompanySlotsConfig, error)
```

Получает конфигурацию по ID.

#### GetByCompanyAndService
```go
func (r *Repository) GetByCompanyAndService(ctx context.Context, companyID int64, serviceID *int64) (*domain.CompanySlotsConfig, error)
```

Получает конфигурацию для компании и услуги.

**Логика:**
- Если `serviceID == nil` → возвращает глобальную конфигурацию компании
- Если `serviceID != nil` → возвращает конфигурацию для конкретной услуги

**Примеры:**
```go
// Глобальная конфигурация
config, _ := repo.GetByCompanyAndService(ctx, companyID, nil)

// Конфигурация для услуги
serviceID := int64(456)
config, _ := repo.GetByCompanyAndService(ctx, companyID, &serviceID)
```

#### GetAllByCompany
```go
func (r *Repository) GetAllByCompany(ctx context.Context, companyID int64) ([]*domain.CompanySlotsConfig, error)
```

Получает все конфигурации компании (глобальную и для каждой услуги).

**Сортировка:** Глобальная конфигурация (`service_id IS NULL`) всегда первая.

#### Update
```go
func (r *Repository) Update(ctx context.Context, id int64, config *domain.CompanySlotsConfig) (*domain.CompanySlotsConfig, error)
```

Обновляет конфигурацию слотов.

#### Delete
```go
func (r *Repository) Delete(ctx context.Context, id int64) error
```

Удаляет конфигурацию по ID.

#### DeleteByCompanyAndService
```go
func (r *Repository) DeleteByCompanyAndService(ctx context.Context, companyID int64, serviceID *int64) error
```

Удаляет конфигурацию по естественному ключу (company_id, service_id).

## Константы статусов

Все статусы бронирований вынесены в константы в `domain/constants.go`:

```go
// Неактивные статусы - используются для фильтрации при подсчёте доступных слотов
var InactiveStatuses = []BookingStatus{
    StatusCancelledByUser,
    StatusCancelledByCompany,
    StatusNoShow,
}

// Активные статусы - используются для фильтрации активных бронирований
var ActiveStatuses = []BookingStatus{
    StatusPending,
    StatusConfirmed,
    StatusInProgress,
    StatusCompleted,
}
```

**Применение:**
```go
// В repository
inactiveStatusStrings := make([]string, len(domain.InactiveStatuses))
for i, status := range domain.InactiveStatuses {
    inactiveStatusStrings[i] = string(status)
}

selectBuilder.Where(squirrel.NotEq{"status": inactiveStatusStrings})
```

## Transaction Context Helpers

Все helper функции для работы с транзакциями находятся в `pkg/dbmetrics`:

### WithTx
```go
func WithTx(ctx context.Context, tx TxExecutor) context.Context
```

Добавляет транзакцию в контекст. Используется внутри `BeginTx`.

### GetExecutor
```go
func GetExecutor(ctx context.Context, db DBExecutor) DBExecutor
```

Извлекает executor из контекста. Если есть транзакция - возвращает её, иначе - db.

**Использование в repository:**
```go
func (r *Repository) Create(ctx context.Context, booking *domain.Booking) (*domain.Booking, error) {
    executor := dbmetrics.GetExecutor(ctx, r.db)  // Авто-определяет транзакцию

    err := executor.QueryRowContext(ctx, query, args...).Scan(&booking.ID)
    return booking, err
}
```

### IsInTransaction
```go
func IsInTransaction(ctx context.Context) bool
```

Проверяет наличие транзакции в контексте.

**Использование:**
```go
// Автоматически добавляем FOR UPDATE если в транзакции
if dbmetrics.IsInTransaction(ctx) {
    selectBuilder = selectBuilder.Suffix("FOR UPDATE")
}
```

## Примеры использования

### Простое создание бронирования
```go
booking := &domain.Booking{
    UserID:    123,
    CompanyID: 1,
    // ...
}

created, err := repo.Create(ctx, booking)
if err != nil {
    return err
}
```

### Создание с проверкой доступности (с транзакцией)
```go
// Начинаем транзакцию
ctx, tx, err := bookingRepo.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Получаем бронирования с блокировкой (авто-добавляет FOR UPDATE)
existingBookings, err := bookingRepo.GetByCompanyAndDate(ctx, companyID, date)
if err != nil {
    return err
}

// Проверяем доступность (бизнес-логика в service)
config, _ := configRepo.GetByCompanyAndService(ctx, companyID, &serviceID)
if len(existingBookings) >= config.MaxConcurrentBookings {
    return ErrSlotNotAvailable
}

// Создаем бронирование (использует ту же транзакцию)
booking, err := bookingRepo.Create(ctx, booking)
if err != nil {
    return err
}

// Коммитим
if err := tx.Commit(); err != nil {
    return err
}
```

### Получение истории пользователя
```go
// Все бронирования
allBookings, _ := repo.GetByUserID(ctx, userID, nil)

// Только активные
activeStatus := domain.StatusConfirmed
activeBookings, _ := repo.GetByUserID(ctx, userID, &activeStatus)
```

### Работа с конфигурацией
```go
// Глобальная конфигурация компании
globalConfig, err := configRepo.GetByCompanyAndService(ctx, companyID, nil)

// Конфигурация для конкретной услуги
serviceID := int64(456)
serviceConfig, err := configRepo.GetByCompanyAndService(ctx, companyID, &serviceID)

// Все конфигурации компании
allConfigs, err := configRepo.GetAllByCompany(ctx, companyID)
// allConfigs[0] - всегда глобальная (service_id IS NULL)
// allConfigs[1:] - конфигурации для услуг
```

## Автоматический сбор метрик

Все запросы автоматически собирают метрики через `pkg/dbmetrics`:

**Метрики:**
- `db_queries_total{service, operation, table, status}` - количество запросов
- `db_query_duration_seconds{service, operation, table}` - длительность
- `db_errors_total{service, operation, table, error_type}` - ошибки
- `db_connections_*` - статистика connection pool

**Не требует дополнительного кода** - всё работает автоматически через wrapper!

## Best Practices

1. **Никакой бизнес-логики** - только CRUD операции
2. **Используйте транзакции** при создании бронирований для предотвращения race conditions
3. **Используйте константы** для статусов вместо хардкода
4. **Обрабатывайте ошибки** - проверяйте `ErrBookingNotFound`, `ErrConfigNotFound` и т.д.
5. **Передавайте `time.Time`** вместо string для дат
6. **Документируйте методы** - укажите когда использовать транзакции
7. **Используйте `Cancel`** вместо `Delete` для сохранения истории

## Связь со слоями

```
Handler Layer (API)
    ↓
Service Layer (бизнес-логика)
    ↓
Repository Layer (CRUD) ← ВЫ ЗДЕСЬ
    ↓
Database (PostgreSQL)
```

**Важно:** Repository **НЕ знает** о handler'ах и service'ах. Он знает только о:
- Domain моделях (`domain.Booking`, `domain.CompanySlotsConfig`)
- Database executor'ах (`DBExecutor`, `TxExecutor`)
- SQL запросах

Это обеспечивает чистоту архитектуры и тестируемость.

# BookingService - Сервис бронирования для автомоек, СТО и детейлингов

Микросервис для онлайн-записи в Telegram mini-app. Часть экосистемы SMC (Service Management Complex).

## Описание проекта

BookingService предоставляет функционал бронирования времени и услуг с поддержкой:
- Просмотра свободных слотов с гибким шагом (30, 60, 90 минут и т.д.)
- Создания бронирований с автоматическим выбором автомобиля пользователя
- **Поддержки нескольких адресов компании** (компания может иметь несколько точек обслуживания)
- Параллельного бронирования (несколько боксов одновременно на одном адресе)
- Гибкой конфигурации слотов (глобально для компании/адреса или для конкретной услуги)
- Истории бронирований пользователя и компании

## Архитектура экосистемы

BookingService взаимодействует с другими микросервисами:

```
┌─────────────────┐      ┌──────────────────┐      ┌─────────────────┐
│  UserService    │      │  SellerService   │      │  PriceService   │
│  (port 8080)    │      │  (port 8081)     │      │  (port 8082)    │
│                 │      │                  │      │                 │
│ - Пользователи  │      │ - Компании       │      │ - Цены услуг    │
│ - Автомобили    │      │ - Адреса         │      │ - Классы авто   │
│ - Выбранное авто│      │ - Услуги         │      │ - Множители     │
│                 │      │ - Рабочие часы   │      │                 │
└────────┬────────┘      └────────┬─────────┘      └────────┬────────┘
         │                        │                         │
         │                        │                         │
         └────────────────────────┼─────────────────────────┘
                                  │
                         ┌────────▼────────┐
                         │ BookingService  │
                         │  (port 8083)    │
                         │                 │
                         │ - Бронирования  │
                         │ - Свободные     │
                         │   слоты         │
                         │ - Конфигурация  │
                         │   слотов        │
                         └─────────────────┘
```

## Ключевые концепции

### 1. Поддержка множественных адресов компании

**Новое в архитектуре:** Компания может иметь несколько точек обслуживания (адресов).

```
Автомойка "Премиум"
├─ Адрес 1 (ID: 100): Москва, ул. Тверская, 10
│  └─ Бронирование с address_id = 100
├─ Адрес 2 (ID: 101): Москва, ул. Ленина, 5
│  └─ Бронирование с address_id = 101
└─ Услуга может быть доступна на одном или нескольких адресах
   (addressIds в SellerService.Service)
```

**Реализация:**
- В таблице `bookings` добавлено поле `address_id` (BIGINT NOT NULL)
- При создании бронирования клиент должен указать конкретный адрес
- Конфигурация слотов может быть задана для каждого адреса отдельно
- Валидация: адрес должен быть в списке адресов компании, где доступна услуга

### 2. Параллельные бронирования

Система поддерживает несколько одновременных бронирований на один временной слот через `maxConcurrentBookings`:

```
Автомойка на Тверской с 4 боксами → maxConcurrentBookings = 4
├─ 10:00-10:30: 4/4 свободных мест
├─ 10:30-11:00: 2/4 свободных мест (2 уже забронированы)
└─ 11:00-11:30: 0/4 свободных мест (все заняты)
```

**Реализация:**
- При создании бронирования проверяется количество активных записей в слоте на конкретном адресе
- Используется транзакционная обработка через `TransactionManager`
- Транзакции с уровнем изоляции `Serializable` для критических операций

### 3. Гибкая конфигурация слотов

Настройки можно задавать на трёх уровнях (с приоритетом):

**1. Для конкретной услуги на конкретном адресе:**
```sql
INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
VALUES (123, 100, 456, 2);  -- Услуга 456 на адресе 100 использует 2 бокса
```

**2. Глобально для адреса:**
```sql
INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
VALUES (123, 100, NULL, 4);  -- NULL service_id = настройка для всего адреса
```

**3. Глобально для компании:**
```sql
INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
VALUES (123, NULL, NULL, 3);  -- NULL address_id и service_id = глобальная настройка
```

**Приоритет применения (от высшего к низшему):**
1. Настройка для конкретной услуги на конкретном адресе
2. Настройка для адреса (все услуги на этом адресе)
3. Глобальная настройка компании (все адреса и услуги)
4. Дефолтные значения из приложения (см. `domain/constants.go`)

### 4. Денормализация данных

При создании бронирования сохраняются снимки данных:
- **Из SellerService**: название услуги, цена
- **Из UserService**: марка, модель, госномер автомобиля

**Зачем:**
- Сохранение истории даже при изменении/удалении данных в других сервисах
- Независимость от доступности других сервисов при чтении истории
- Юридическая значимость (пользователь видит ту цену, которую он согласился платить)

### 5. Ограничения бронирования

```go
type CompanySlotsConfig struct {
    SlotDurationMinutes     int  // Длительность слота (30, 60, 90 минут)
    AdvanceBookingDays      int  // 0 = без ограничений, 30 = только на 30 дней вперёд
    MinBookingNoticeMinutes int  // 60 = нельзя записаться менее чем за час
    MaxConcurrentBookings   int  // Количество боксов на адресе
}
```

**Примеры:**
- `advanceBookingDays = 0` → можно бронировать на любую дату в будущем
- `advanceBookingDays = 14` → можно бронировать только на 14 дней вперёд
- `minBookingNoticeMinutes = 120` → минимум за 2 часа до записи

## База данных

### Таблицы

#### bookings
Основная таблица бронирований с денормализованными данными и **поддержкой адресов**.

```sql
CREATE TABLE bookings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,              -- Telegram ID
    company_id BIGINT NOT NULL,
    address_id BIGINT NOT NULL,           -- ID адреса из SellerService
    service_id BIGINT NOT NULL,
    car_id BIGINT NOT NULL,

    booking_date DATE NOT NULL,
    start_time TIME NOT NULL,
    duration_minutes INT NOT NULL,        -- Длительность услуги (30, 60, 90, 120 и т.д.)

    status VARCHAR(30) NOT NULL,          -- confirmed, completed, cancelled_by_user, etc.

    -- Денормализованные данные для истории
    service_name VARCHAR(200) NOT NULL,
    service_price DECIMAL(10,2) NOT NULL,
    car_brand VARCHAR(100),
    car_model VARCHAR(100),
    car_license_plate VARCHAR(20),

    notes TEXT,
    cancellation_reason TEXT,
    cancelled_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Индексы:**
- `idx_bookings_user_id` - для истории пользователя
- `idx_bookings_company_date` - для списка бронирований компании
- `idx_bookings_company_address_date_time` - для проверки доступности слотов на адресе
- `idx_bookings_availability` (partial) - для проверки свободных слотов (исключает отменённые)

#### company_slots_config
Конфигурация слотов для компаний, адресов и услуг (с поддержкой иерархии).

```sql
CREATE TABLE company_slots_config (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    address_id BIGINT,                    -- NULL = настройка для всех адресов
    service_id BIGINT,                    -- NULL = настройка для всех услуг

    slot_duration_minutes INT NOT NULL DEFAULT 30,
    max_concurrent_bookings INT NOT NULL DEFAULT 1,
    advance_booking_days INT NOT NULL DEFAULT 0,
    min_booking_notice_minutes INT NOT NULL DEFAULT 60,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_company_address_service UNIQUE (company_id, address_id, service_id)
);
```

**Примеры настройки:**
```sql
-- Глобально для компании 123
INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
VALUES (123, NULL, NULL, 3);

-- Для адреса 100 компании 123
INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
VALUES (123, 100, NULL, 4);

-- Для услуги 456 на адресе 100 компании 123
INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
VALUES (123, 100, 456, 2);
```

### Миграции

Используется [golang-migrate/migrate](https://github.com/golang-migrate/migrate):

```
migrations/
├── 000001_create_bookings_table.up.sql
├── 000001_create_bookings_table.down.sql
├── 000002_create_company_slots_config_table.up.sql
├── 000002_create_company_slots_config_table.down.sql
├── 000003_create_triggers.up.sql
├── 000003_create_triggers.down.sql
└── fixtures/
    ├── 001_company_configs.sql
    ├── 002_bookings.sql
    └── README.md
```

**Применение:**
```bash
make migrate-up    # Применить миграции
make fixtures      # Загрузить тестовые данные
make db-reset      # Сбросить БД и применить миграции заново
```

## API Endpoints

### Авторизация
- **GET запросы** с проверкой прав: `X-User-ID` header
- **POST/PATCH/PUT запросы**: `userId` в body
- **Публичные endpoints**: без авторизации

### Endpoints

#### 1. Создать бронирование
```http
POST /api/v1/bookings
Content-Type: application/json

{
  "userId": 123456789,
  "companyId": 1,
  "addressId": 100,          // НОВОЕ: ID конкретного адреса
  "serviceId": 3,
  "bookingDate": "2025-10-15",
  "startTime": "10:00",
  "notes": "Пожалуйста, уделите внимание дискам"
}
```

**Валидация:**
- Компания с `companyId` должна существовать в SellerService
- Адрес с `addressId` должен быть в списке адресов компании
- Услуга с `serviceId` должна быть доступна на указанном адресе (проверка `addressIds` в Service)
- У пользователя должен быть выбранный автомобиль в UserService

#### 2. Получить бронирование
```http
GET /api/v1/bookings/{id}
X-User-ID: 123456789
```

#### 3. Отменить бронирование
```http
PATCH /api/v1/bookings/{id}/cancel
Content-Type: application/json

{
  "userId": 123456789,
  "cancellationReason": "Изменились планы"
}
```

#### 4. История пользователя
```http
GET /api/v1/users/{userId}/bookings?status=confirmed
```

#### 5. Свободные слоты (публичный)
```http
GET /api/v1/companies/{companyId}/addresses/{addressId}/available-slots?serviceId=3&date=2025-10-15
```

**ВАЖНО:** Теперь слоты запрашиваются для конкретного адреса!

**Response:**
```json
{
  "date": "2025-10-15",
  "companyId": 1,
  "addressId": 100,
  "serviceId": 3,
  "slots": [
    {
      "startTime": "10:00",
      "durationMinutes": 30,
      "availableSpots": 2,
      "totalSpots": 4
    }
  ]
}
```

#### 6. Бронирования компании (для менеджеров)
```http
GET /api/v1/companies/{companyId}/bookings?addressId=100&date=2025-10-15&status=confirmed
X-User-ID: 987654321
```

**Фильтры:**
- `addressId` (optional) - бронирования на конкретном адресе
- `date` (optional) - бронирования на конкретную дату
- `startDate` и `endDate` (optional) - бронирования за период
- `status` (optional) - фильтр по статусу
- `includeInactive` (optional, boolean) - включить отменённые

#### 7. Конфигурация слотов (публичный)
```http
GET /api/v1/companies/{companyId}/config?addressId=100&serviceId=3
```

**Query параметры:**
- `addressId` (optional) - для получения конфигурации конкретного адреса
- `serviceId` (optional) - для получения конфигурации конкретной услуги

**Response учитывает приоритет:**
1. Сначала ищется конфигурация для (companyId, addressId, serviceId)
2. Затем для (companyId, addressId, NULL)
3. Затем для (companyId, NULL, NULL)
4. Возвращаются дефолтные значения

#### 8. Обновить конфигурацию (для менеджеров)
```http
PUT /api/v1/companies/{companyId}/config
Content-Type: application/json

{
  "userId": 987654321,
  "addressId": 100,          // Опционально: для конкретного адреса
  "serviceId": 3,            // Опционально: для конкретной услуги
  "slotDurationMinutes": 30,
  "maxConcurrentBookings": 4,
  "advanceBookingDays": 0,
  "minBookingNoticeMinutes": 60
}
```

### Статусы бронирований

| Статус | Описание |
|--------|----------|
| `pending` | Ожидает подтверждения |
| `confirmed` | Подтверждена |
| `in_progress` | В процессе выполнения |
| `completed` | Выполнена |
| `cancelled_by_user` | Отменена пользователем |
| `cancelled_by_company` | Отменена компанией |
| `no_show` | Клиент не явился |

## Архитектура кода

Проект следует Clean Architecture и разделён на слои:

```
├── cmd/                    # Точки входа приложения
│   └── main.go
├── internal/              # Внутренняя бизнес-логика
│   ├── api/              # HTTP handlers и middleware
│   │   ├── handlers/
│   │   │   ├── create_booking/        # POST /bookings
│   │   │   ├── get_booking/           # GET /bookings/{id}
│   │   │   ├── cancel_booking/        # PATCH /bookings/{id}/cancel
│   │   │   ├── get_user_bookings/     # GET /users/{userId}/bookings
│   │   │   ├── get_available_slots/   # GET /companies/{id}/addresses/{id}/available-slots
│   │   │   ├── get_company_bookings/  # GET /companies/{id}/bookings
│   │   │   ├── get_company_config/    # GET /companies/{id}/config
│   │   │   └── update_company_config/ # PUT /companies/{id}/config
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   └── metrics.go
│   │   └── utils.go
│   ├── usecase/          # Use cases (бизнес-логика высокого уровня)
│   │   ├── create_booking/
│   │   │   ├── usecase.go
│   │   │   ├── contract.go
│   │   │   └── errors.go
│   │   └── get_available_slots/
│   │       ├── usecase.go
│   │       ├── contract.go
│   │       └── errors.go
│   ├── service/          # Сервисный слой (CRUD операции)
│   │   ├── bookings/
│   │   │   ├── service.go         # GetByID, GetUserBookings, Cancel, UpdateStatus
│   │   │   ├── contracts.go       # Интерфейсы зависимостей
│   │   │   ├── errors.go
│   │   │   └── models/
│   │   │       └── models.go
│   │   └── config/
│   │       ├── service.go         # CRUD для конфигурации слотов
│   │       ├── contracts.go
│   │       ├── errors.go
│   │       └── models/
│   ├── infra/            # Инфраструктура
│   │   ├── storage/      # Репозитории БД
│   │   │   ├── booking/
│   │   │   │   ├── contract.go    # Интерфейсы DBExecutor, TxExecutor
│   │   │   │   ├── repository.go  # CRUD методы
│   │   │   │   └── errors.go
│   │   │   └── config/
│   │   │       ├── contract.go
│   │   │       ├── repository.go
│   │   │       └── errors.go
│   │   └── integrations/ # HTTP клиенты для других сервисов
│   │       ├── userservice/
│   │       │   ├── client.go
│   │       │   ├── contract.go
│   │       │   ├── models.go
│   │       │   └── errors.go
│   │       └── sellerservice/
│   │           ├── client.go
│   │           ├── contract.go
│   │           ├── models.go
│   │           └── errors.go
│   ├── domain/           # Доменные модели
│   │   ├── booking.go
│   │   ├── config.go
│   │   └── constants.go
│   └── config/           # Конфигурация приложения
│       └── config.go
├── pkg/                   # Переиспользуемые пакеты
│   ├── metrics/          # Prometheus метрики
│   ├── dbmetrics/        # Обёртка над database/sql с метриками и transaction context
│   ├── txmanager/        # Transaction Manager с поддержкой метрик
│   ├── logger/           # Структурированное логирование
│   ├── psqlbuilder/      # SQL query builder (обёртка над squirrel с $-плейсхолдерами)
│   └── types/            # Переиспользуемые типы (TimeString)
├── migrations/           # Миграции базы данных
├── schemas/              # OpenAPI спецификации
│   ├── schema.yaml       # API сервиса
│   └── clients/          # Схемы других сервисов
│       ├── smc-sellerservice.yaml
│       └── smk-userservice.yaml
└── test_data/           # Тестовые данные
```

## Ключевые паттерны

### 1. Repository слой с поддержкой транзакций через контекст

Repository слой реализован без бизнес-логики - только чистые CRUD операции.

**Работа с транзакциями через context:**
```go
// Создание без транзакции
booking, err := repo.Create(ctx, booking)

// Создание с транзакцией через TransactionManager
err := txManager.DoSerializable(ctx, func(txCtx context.Context) error {
    // Контекст txCtx автоматически содержит транзакцию
    booking, err := repo.Create(txCtx, booking)
    if err != nil {
        return err
    }

    // Все методы автоматически используют транзакцию из контекста
    bookings, err := repo.GetByCompanyAndDateAndAddress(txCtx, companyID, addressID, date)
    // ...
    return nil
})
```

**Transaction context helpers (в pkg/dbmetrics):**
```go
// WithTx - добавляет транзакцию в контекст
ctx = dbmetrics.WithTx(ctx, tx)

// GetExecutor - извлекает executor (транзакцию или db)
executor := dbmetrics.GetExecutor(ctx, r.db)

// IsInTransaction - проверяет наличие транзакции
if dbmetrics.IsInTransaction(ctx) {
    // Добавить FOR UPDATE к запросу
}
```

**Booking Repository методы:**
- `Create(ctx, booking)` - создание
- `GetByID(ctx, id)` - получение по ID
- `GetByUserID(ctx, userID, status)` - история пользователя
- `GetByCompanyWithFilter(ctx, filter)` - бронирования компании с фильтрацией по адресу, дате, статусу
- `GetByCompanyAndDateAndAddress(ctx, companyID, addressID, date)` - бронирования на конкретную дату и адрес
- `UpdateStatus(ctx, id, status)` - обновление статуса
- `Cancel(ctx, id, status, reason)` - отмена с причиной
- `Delete(ctx, id)` - физическое удаление

**Config Repository методы:**
- `Create(ctx, config)` - создание конфигурации
- `GetByID(ctx, id)` - получение по ID
- `GetByCompanyAddressAndService(ctx, companyID, addressID, serviceID)` - получение конфигурации точно по параметрам
  - addressID и serviceID могут быть `nil` (NULL в БД)
  - Поиск ТОЛЬКО точного совпадения (companyID, addressID, serviceID)
- `GetConfigWithHierarchy(ctx, companyID, addressID, serviceID)` - иерархический поиск с приоритетами
  - Уровень 1: если указаны оба параметра → (companyID, addressID, serviceID)
  - Уровень 2: если указан адрес → (companyID, addressID, NULL)
  - Уровень 3: если указана услуга → (companyID, NULL, serviceID)
  - Уровень 4: глобальная → (companyID, NULL, NULL)
  - addressID и serviceID принимаются как `*int64` (nil означает "не указано")
  - Возвращает `ErrConfigNotFound` если не найдено ни на одном уровне
- `GetAllByCompany(ctx, companyID)` - все конфигурации компании
- `Update(ctx, id, config)` - обновление
- `Delete(ctx, id)` - удаление по ID

**Константы статусов (domain/constants.go):**
```go
var InactiveStatuses = []BookingStatus{
    StatusCancelledByUser,
    StatusCancelledByCompany,
    StatusNoShow,
}

var ActiveStatuses = []BookingStatus{
    StatusPending,
    StatusConfirmed,
    StatusInProgress,
    StatusCompleted,
}
```

**Важные изменения в моделях сервисов:**

Config Service использует обновлённую модель `GetConfigRequest`:
```go
// internal/service/config/models/models.go
type GetConfigRequest struct {
    CompanyID int64  `json:"companyId"`
    AddressID *int64 `json:"addressId,omitempty"` // nil = любой адрес
    ServiceID *int64 `json:"serviceId,omitempty"` // nil = любая услуга
}
```

**До изменения:** использовались `int64` со значением `0` для обозначения отсутствия параметра
**После изменения:** используются `*int64` с `nil` для явного указания отсутствия параметра

Это влияет на:
- `ConfigRepository.GetConfigWithHierarchy` - принимает `*int64`
- `ConfigService.GetWithHierarchy` - использует `GetConfigRequest` с `*int64`
- Все handlers для config - конвертируют пустые строки в `nil`

### 2. Transaction Manager для управления транзакциями

```go
type TransactionManager struct {
    db *dbmetrics.DB
}

// Do - обычная транзакция
func (tm *TransactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error

// DoSerializable - транзакция с уровнем изоляции Serializable
func (tm *TransactionManager) DoSerializable(ctx context.Context, fn func(ctx context.Context) error) error

// DoReadOnly - read-only транзакция
func (tm *TransactionManager) DoReadOnly(ctx context.Context, fn func(ctx context.Context) error) error
```

**Особенности:**
- Автоматический rollback при ошибке
- Автоматический commit при успехе
- Поддержка вложенных транзакций (повторное использование существующей)
- Интеграция с метриками через `dbmetrics.DB`

### 3. Обёртывание ошибок между слоями

```go
// Repository Layer
var ErrBookingNotFound = errors.New("booking.repository: booking not found")

// Service Layer
var ErrBookingNotFound = errors.New("booking not found")

func (s *Service) GetByID(ctx context.Context, id int64) (*Booking, error) {
    booking, err := s.repo.GetByID(ctx, id)
    if errors.Is(err, repository.ErrBookingNotFound) {
        return nil, ErrBookingNotFound  // Преобразование в service error
    }
    return booking, nil
}

// Handler Layer
if errors.Is(err, service.ErrBookingNotFound) {
    handlers.RespondNotFound(w)  // 404
    return
}
```

### 4. Contract-based Dependency Injection

```go
// internal/usecase/create_booking/contract.go
type BookingRepository interface {
    Create(ctx context.Context, booking *domain.Booking) (*domain.Booking, error)
    GetByCompanyAndDateAndAddress(ctx, companyID, addressID int64, date time.Time) ([]*domain.Booking, error)
}

type ConfigRepository interface {
    GetByCompanyAddressAndService(ctx, companyID, addressID, serviceID *int64) (*domain.CompanySlotsConfig, error)
}

type UserServiceClient interface {
    GetSelectedCar(ctx context.Context, userID int64) (*Car, error)
}

type SellerServiceClient interface {
    GetCompany(ctx context.Context, companyID int64) (*Company, error)
    GetService(ctx context.Context, companyID, serviceID int64) (*Service, error)
}

type TransactionManager interface {
    DoSerializable(ctx context.Context, fn func(ctx context.Context) error) error
}

type Logger interface {
    Info(format string, v ...interface{})
    Error(format string, v ...interface{})
    Warn(format string, v ...interface{})
}
```

### 5. Use Case: Получение доступных слотов

```go
// internal/usecase/get_available_slots/usecase.go

func (uc *UseCase) Execute(ctx context.Context, req *Request) (*Response, error) {
    // 1. Получить компанию и проверить существование адреса
    company, err := uc.sellerClient.GetCompany(ctx, req.CompanyID)
    if err != nil {
        return nil, err
    }

    // 2. Проверить, что адрес принадлежит компании
    if !addressBelongsToCompany(company, req.AddressID) {
        return nil, ErrAddressNotFound
    }

    // 3. Получить услугу и проверить, что она доступна на этом адресе
    service, err := uc.sellerClient.GetService(ctx, req.CompanyID, req.ServiceID)
    if err != nil {
        return nil, err
    }

    if !serviceAvailableAtAddress(service, req.AddressID) {
        return nil, ErrServiceNotAvailableAtAddress
    }

    // 4. Получить конфигурацию слотов с учётом приоритета
    config, err := uc.configRepo.GetByCompanyAddressAndService(
        ctx, req.CompanyID, &req.AddressID, &req.ServiceID,
    )
    if err != nil || config == nil {
        // Использовать дефолтные значения
        config = getDefaultConfig()
    }

    // 5. Получить рабочие часы для указанной даты
    workingHours := getWorkingHoursForDay(company, req.Date)
    if !workingHours.IsOpen {
        return &Response{Slots: []Slot{}}, nil
    }

    // 6. Получить все бронирования на эту дату и адрес
    bookings, err := uc.bookingRepo.GetByCompanyAndDateAndAddress(
        ctx, req.CompanyID, req.AddressID, req.Date,
    )
    if err != nil {
        return nil, err
    }

    // 7. Сгенерировать все возможные слоты
    allSlots := generateSlots(workingHours, config.SlotDurationMinutes)

    // 8. Для каждого слота посчитать свободные места
    for i := range allSlots {
        occupied := countOverlappingBookings(bookings, allSlots[i])
        allSlots[i].AvailableSpots = config.MaxConcurrentBookings - occupied
        allSlots[i].TotalSpots = config.MaxConcurrentBookings
    }

    return &Response{
        Date:      req.Date,
        CompanyID: req.CompanyID,
        AddressID: req.AddressID,
        ServiceID: req.ServiceID,
        Slots:     allSlots,
    }, nil
}
```

### 6. Use Case: Создание бронирования

```go
// internal/usecase/create_booking/usecase.go

func (uc *UseCase) Execute(ctx context.Context, req *Request) (*Response, error) {
    var result *domain.Booking

    // Выполняем всё в сериализуемой транзакции
    err := uc.txManager.DoSerializable(ctx, func(txCtx context.Context) error {
        // 1. Валидация через внешние сервисы
        company, err := uc.sellerClient.GetCompany(txCtx, req.CompanyID)
        if err != nil {
            return err
        }

        // 2. Проверить, что адрес принадлежит компании
        if !addressBelongsToCompany(company, req.AddressID) {
            return ErrAddressNotFound
        }

        // 3. Получить услугу и проверить доступность на адресе
        service, err := uc.sellerClient.GetService(txCtx, req.CompanyID, req.ServiceID)
        if err != nil {
            return err
        }

        if !serviceAvailableAtAddress(service, req.AddressID) {
            return ErrServiceNotAvailableAtAddress
        }

        // 4. Получить выбранный автомобиль пользователя
        car, err := uc.userClient.GetSelectedCar(txCtx, req.UserID)
        if err != nil {
            return err
        }

        // 5. Получить конфигурацию слотов
        config, err := uc.configRepo.GetByCompanyAddressAndService(
            txCtx, req.CompanyID, &req.AddressID, &req.ServiceID,
        )
        if err != nil || config == nil {
            config = getDefaultConfig()
        }

        // 6. Проверить ограничения бронирования
        if err := validateBookingConstraints(req, config); err != nil {
            return err
        }

        // 7. Проверить доступность слота с блокировкой (FOR UPDATE в транзакции)
        bookings, err := uc.bookingRepo.GetByCompanyAndDateAndAddress(
            txCtx, req.CompanyID, req.AddressID, req.BookingDate,
        )
        if err != nil {
            return err
        }

        occupied := countOverlappingBookings(bookings, req.StartTime, config.SlotDurationMinutes)
        if occupied >= config.MaxConcurrentBookings {
            return ErrSlotNotAvailable
        }

        // 8. Создать бронирование с денормализацией
        booking := &domain.Booking{
            UserID:          req.UserID,
            CompanyID:       req.CompanyID,
            AddressID:       req.AddressID,
            ServiceID:       req.ServiceID,
            CarID:           car.ID,
            BookingDate:     req.BookingDate,
            StartTime:       req.StartTime,
            DurationMinutes: config.SlotDurationMinutes,
            Status:          domain.StatusConfirmed,
            // Денормализация
            ServiceName:     service.Name,
            ServicePrice:    service.Price,
            CarBrand:        &car.Brand,
            CarModel:        &car.Model,
            CarLicensePlate: &car.LicensePlate,
            Notes:           req.Notes,
        }

        created, err := uc.bookingRepo.Create(txCtx, booking)
        if err != nil {
            return err
        }

        result = created
        return nil
    })

    if err != nil {
        return nil, err
    }

    return toResponse(result), nil
}
```

### 7. Handlers Layer - HTTP обработчики запросов

Все handlers следуют единой архитектуре с разделением на файлы:

**Структура каждого handler:**
```
internal/api/handlers/handler_name/
├── handler.go    # Основной HTTP handler с методом Handle
├── contract.go   # Интерфейсы зависимостей (UseCase/Service, Logger)
└── models.go     # HTTP модели request/response и конвертеры (опционально)
```

**Именование:**
- Пакеты с подчёркиванием: `create_booking`, `get_available_slots`
- Импорты используют lowerCamelCase: `createBooking "...internal/usecase/create_booking"`

**Примеры реализации:**

#### Handler с UseCase (create_booking)
```go
// handler.go
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    var req CreateBookingRequest
    if err := handlers.DecodeJSON(r, &req); err != nil {
        handlers.RespondBadRequest(w, "invalid request body")
        return
    }

    // Конвертация HTTP модели в модель UseCase
    useCaseReq, err := req.ToUseCaseRequest()
    if err != nil {
        handlers.RespondBadRequest(w, "invalid date or time format")
        return
    }

    // Вызов UseCase
    result, err := h.useCase.Execute(r.Context(), useCaseReq)
    if err != nil {
        // Обработка специфичных ошибок UseCase
        switch {
        case errors.Is(err, createBooking.ErrSlotNotAvailable):
            handlers.RespondError(w, http.StatusConflict, "slot not available")
        case errors.Is(err, createBooking.ErrCompanyNotFound):
            handlers.RespondNotFound(w, "company not found")
        default:
            handlers.RespondInternalError(w)
        }
        return
    }

    // Конвертация ответа UseCase в HTTP ответ
    response := FromUseCaseResponse(result)
    handlers.RespondJSON(w, http.StatusCreated, response)
}
```

#### Handler с Service (get_booking)
```go
// handler.go
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    // Извлечение параметров из URL
    vars := mux.Vars(r)
    bookingID, err := strconv.ParseInt(vars["bookingId"], 10, 64)
    if err != nil {
        handlers.RespondBadRequest(w, "invalid booking ID")
        return
    }

    // Получение userID из контекста (middleware)
    userID, ok := middleware.GetUserID(r.Context())
    if !ok {
        handlers.RespondUnauthorized(w, "missing user ID")
        return
    }

    // Вызов Service (проверка прав внутри)
    booking, err := h.service.GetByID(r.Context(), bookingID, userID)
    if err != nil {
        switch {
        case errors.Is(err, bookings.ErrBookingNotFound):
            handlers.RespondNotFound(w, "booking not found")
        case errors.Is(err, bookings.ErrAccessDenied):
            handlers.RespondForbidden(w, "access denied")
        default:
            handlers.RespondInternalError(w)
        }
        return
    }

    // Service возвращает готовую HTTP модель
    handlers.RespondJSON(w, http.StatusOK, booking)
}
```

#### Публичный Handler (get_available_slots)
```go
// handler.go - без авторизации
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    // Извлечение параметров из URL и query
    vars := mux.Vars(r)
    companyID, _ := strconv.ParseInt(vars["companyId"], 10, 64)
    addressID, _ := strconv.ParseInt(vars["addressId"], 10, 64)

    serviceIDStr := r.URL.Query().Get("serviceId")
    dateStr := r.URL.Query().Get("date")

    // Валидация и парсинг параметров
    useCaseReq, err := ToUseCaseRequest(companyID, addressID, serviceIDStr, dateStr)
    if err != nil {
        handlers.RespondBadRequest(w, "invalid parameters")
        return
    }

    // Вызов UseCase
    result, err := h.useCase.Execute(r.Context(), useCaseReq)
    // ...
}
```

**Вспомогательные функции (internal/api/handlers/utils.go):**
```go
// RespondJSON - отправка JSON ответа
func RespondJSON(w http.ResponseWriter, status int, payload interface{})

// RespondError - отправка ошибки с кодом и сообщением
func RespondError(w http.ResponseWriter, status int, message string)

// Специализированные функции для типовых ошибок
func RespondBadRequest(w http.ResponseWriter, message string)    // 400
func RespondUnauthorized(w http.ResponseWriter, message string)  // 401
func RespondForbidden(w http.ResponseWriter, message string)     // 403
func RespondNotFound(w http.ResponseWriter, message string)      // 404
func RespondInternalError(w http.ResponseWriter)                 // 500

// DecodeJSON - парсинг JSON из body
func DecodeJSON(r *http.Request, v interface{}) error
```

**Модели конвертации (models.go):**
```go
// HTTP request model с JSON тегами
type CreateBookingRequest struct {
    UserID      int64   `json:"userId"`
    CompanyID   int64   `json:"companyId"`
    BookingDate string  `json:"bookingDate"` // "2025-10-15"
    StartTime   string  `json:"startTime"`   // "10:00"
    // ...
}

// Конвертер в модель UseCase
func (r *CreateBookingRequest) ToUseCaseRequest() (*createBooking.Request, error) {
    // Парсинг даты
    date, err := time.Parse(domain.DateFormat, r.BookingDate)
    if err != nil {
        return nil, err
    }

    // Парсинг времени
    startTime, err := types.NewTimeStringFromString(r.StartTime)
    if err != nil {
        return nil, err
    }

    return &createBooking.Request{
        UserID:    r.UserID,
        CompanyID: r.CompanyID,
        Date:      date,
        StartTime: startTime,
        // ...
    }, nil
}

// HTTP response model
type BookingResponse struct {
    ID          int64  `json:"id"`
    BookingDate string `json:"bookingDate"` // "2025-10-15"
    // ...
}

// Конвертер из UseCase ответа
func FromUseCaseResponse(resp *createBooking.Response) *BookingResponse {
    return &BookingResponse{
        ID:          resp.ID,
        BookingDate: resp.BookingDate.Format(domain.DateFormat),
        StartTime:   resp.StartTime.String(),
        CreatedAt:   resp.CreatedAt.Format(time.RFC3339),
        // ...
    }
}
```

**Особенности реализации:**

1. **Разделение ответственности:**
   - Handler парсит HTTP запрос и формирует HTTP ответ
   - UseCase/Service содержит бизнес-логику
   - Конвертеры изолируют HTTP модели от доменных

2. **Обработка ошибок:**
   - Каждая ошибка из UseCase/Service обрабатывается явно через `errors.Is()`
   - Специфичные HTTP коды для каждого типа ошибки
   - Логирование всех ошибок с контекстом

3. **Авторизация:**
   - GET запросы с X-User-ID header через `middleware.GetUserID()`
   - POST/PATCH/PUT с userId в body
   - Публичные endpoints без проверки прав

4. **Валидация:**
   - Базовая валидация параметров в handler (типы, формат)
   - Бизнес-валидация в UseCase/Service
   - Возврат понятных сообщений об ошибках

**Список всех handlers:**
- `create_booking` - создание бронирования (UseCase)
- `get_available_slots` - получение свободных слотов (UseCase, публичный)
- `get_booking` - получение бронирования (Service)
- `cancel_booking` - отмена бронирования (Service)
- `get_user_bookings` - история пользователя (Service)
- `get_company_bookings` - бронирования компании (Service)
- `get_company_config` - конфигурация компании (Service, публичный)
- `update_company_config` - обновление конфигурации (Service)

## Конфигурация

### .env
```env
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=smk_bookingservice
DB_SSLMODE=disable

# Server
HTTP_PORT=8083

# Logging
LOG_LEVEL=info
LOG_FILE=./logs/app.log

# Metrics
METRICS_ENABLED=true
METRICS_SERVICE_NAME=bookingservice

# External Services
USERSERVICE_URL=http://localhost:8080
SELLERSERVICE_URL=http://localhost:8081
PRICESERVICE_URL=http://localhost:8082
```

### config.toml
```toml
[server]
http_port = 8083
read_timeout = 15
write_timeout = 15
idle_timeout = 120
shutdown_timeout = 10

[database]
host = "localhost"
port = 5432
user = "postgres"
password = "postgres"
dbname = "smk_bookingservice"
sslmode = "disable"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = 300

[metrics]
enabled = true
path = "/metrics"
service_name = "bookingservice"

[logs]
file = "./logs/app.log"
```

## Быстрый старт

### Полная экосистема (рекомендуется)

BookingService зависит от других микросервисов. Для полноценной работы запустите все сервисы:

```bash
# 1. Запустить SellerService, UserService, PriceService (см. документацию каждого сервиса)
cd ../SMK-SellerService && docker-compose up -d
cd ../SMK-UserService && docker-compose up -d
cd ../SMC-PriceService && docker-compose up -d

# 2. Запустить BookingService
cd ../SMC-BookingService
make docker-up

# 3. Загрузить тестовые данные для всех сервисов
chmod +x test_data/load_all_fixtures.sh
./test_data/load_all_fixtures.sh

# 4. Проверить работу API
make test-smoke

# Доступно на:
# - BookingService API: http://localhost:8083/api/v1
# - Metrics: http://localhost:8083/metrics
# - DB: localhost:5438
```

### Только BookingService (для разработки)

Для локальной разработки без интеграций:

```bash
# 1. Установить зависимости
make install

# 2. Настроить окружение
cp .env.example .env
# Отредактируйте .env, установив URLs других сервисов или mock-endpoints

# 3. Запустить БД и применить миграции
make docker-up

# 4. Загрузить тестовые данные
make fixtures

# 5. Запустить приложение (вне Docker, для отладки)
make run
```

**Важно:** Partial unique indexes требуют PostgreSQL 9.5+. При изменении схемы `company_slots_config` используйте `docker-compose down -v` для полного пересоздания БД.

## Makefile команды

### Build & Run
- `make build` - собрать бинарник
- `make run` - запустить локально
- `make test` - запустить тесты
- `make install` - установить зависимости

### Docker
- `make docker-up` - запустить все сервисы
- `make docker-down` - остановить сервисы
- `make docker-logs` - показать логи
- `make docker-clean` - удалить volumes

### Database
- `make migrate-up` - применить миграции
- `make migrate-down` - откатить миграции
- `make db-reset` - сбросить БД
- `make fixtures` - загрузить тестовые данные

### Development
- `make dev` - запустить только БД для локальной разработки
- `make clean` - очистить артефакты
- `make clean-all` - полная очистка

### Testing
- `make test-smoke` - запустить smoke tests (быстрая проверка основных эндпоинтов)
- `make test-api` - запустить интерактивное API тестирование
- `./test_data/load_all_fixtures.sh` - загрузить фикстуры для всех сервисов

## Система метрик Prometheus

### HTTP метрики
- `http_requests_total` - количество HTTP запросов
- `http_request_duration_seconds` - длительность запросов
- `http_errors_total` - количество ошибок

### Database метрики
- `db_queries_total` - количество запросов к БД
- `db_query_duration_seconds` - длительность запросов
- `db_errors_total` - ошибки БД
- `db_connections_*` - статистика пула соединений

**Автоматический сбор:**
- HTTP метрики через middleware
- DB метрики через обёртку `dbmetrics.DB`

## Межсервисное взаимодействие

### Клиенты для других сервисов

```go
// internal/integrations/userservice/client.go
type Client struct {
    baseURL    string
    httpClient *http.Client
    logger     Logger
}

func (c *Client) GetSelectedCar(ctx context.Context, tgUserID int64) (*Car, error) {
    url := fmt.Sprintf("%s/internal/users/%d/cars/selected", c.baseURL, tgUserID)
    // HTTP GET запрос с timeout
    // Парсинг ответа в models.Car
    // Обработка ошибок
}
```

```go
// internal/integrations/sellerservice/client.go
type Client struct {
    baseURL    string
    httpClient *http.Client
    logger     Logger
}

func (c *Client) GetCompany(ctx context.Context, companyID int64) (*Company, error) {
    url := fmt.Sprintf("%s/api/v1/companies/%d", c.baseURL, companyID)
    // HTTP GET запрос
}

func (c *Client) GetService(ctx context.Context, companyID, serviceID int64) (*Service, error) {
    url := fmt.Sprintf("%s/api/v1/companies/%d/services/%d", c.baseURL, companyID, serviceID)
    // HTTP GET запрос
}
```

**Особенности:**
- Timeout для всех запросов (из config)
- Структурированное логирование запросов
- Обработка ошибок с типизацией (ErrCompanyNotFound, ErrServiceNotFound)
- Модели данных в отдельном файле `models.go`
- JSON теги используют snake_case для соответствия API других сервисов

**Docker конфигурация:**
В `.env` для работы из Docker контейнера используется `host.docker.internal`:
```env
USERSERVICE_URL=http://host.docker.internal:8080
SELLERSERVICE_URL=http://host.docker.internal:8081
```

## Важные замечания

### Partial Unique Indexes и NULL значения

В PostgreSQL обычные UNIQUE constraints не работают корректно с NULL значениями (NULL != NULL). Для таблицы `company_slots_config` используются **partial unique indexes** с WHERE условиями:

```sql
-- Глобальная конфигурация: (company_id, NULL, NULL)
CREATE UNIQUE INDEX uq_company_global
    ON company_slots_config (company_id)
    WHERE address_id IS NULL AND service_id IS NULL;

-- Конфигурация для адреса: (company_id, address_id, NULL)
CREATE UNIQUE INDEX uq_company_address
    ON company_slots_config (company_id, address_id)
    WHERE address_id IS NOT NULL AND service_id IS NULL;
```

**Важно:** Partial indexes НЕ работают с `ON CONFLICT`. В фикстурах используется простая вставка без ON CONFLICT - уникальность гарантируется индексами.

### Интеграция с SellerService

SellerService возвращает JSON в snake_case:
```json
{
  "working_hours": {...},
  "manager_ids": [123],
  "address_ids": [100, 101],
  "company_id": 1
}
```

Модели в `internal/integrations/sellerservice/models.go` должны использовать соответствующие JSON теги.

### Загрузка фикстур

Фикстуры нужно загружать в правильном порядке из-за зависимостей:
1. **SellerService** - компании, адреса, услуги
2. **UserService** - пользователи, автомобили
3. **PriceService** - правила ценообразования
4. **BookingService** - конфигурация слотов, бронирования

Используйте скрипт `./test_data/load_all_fixtures.sh` для автоматической загрузки всех фикстур.

### Известные проблемы и решения

**Проблема:** "there is no unique or exclusion constraint matching the ON CONFLICT specification"
**Решение:** Partial indexes не поддерживают ON CONFLICT. Удалите ON CONFLICT блоки из фикстур.

**Проблема:** "company is closed on [date]" при запросе слотов
**Решение:** Проверьте, что SellerService возвращает `working_hours` в JSON. Обновите JSON теги в models.go на snake_case.

**Проблема:** "услуга недоступна на выбранном адресе"
**Решение:** Убедитесь, что JSON тег для AddressIDs использует `address_ids` (не `addressIds`).

## TODO

### Готово ✅

**Инфраструктура:**
- [x] Domain модели (booking.go, config.go, constants.go) с поддержкой адресов
- [x] pkg/dbmetrics с transaction context helpers
- [x] pkg/txmanager - Transaction Manager с поддержкой метрик
- [x] pkg/types - TimeString для работы с временем (с методами IsBefore, IsAfter, AddMinutes)
- [x] pkg/psqlbuilder - обёртка над squirrel
- [x] pkg/metrics - Prometheus метрики
- [x] pkg/logger - структурированное логирование
- [x] Константы статусов бронирований

**База данных:**
- [x] OpenAPI schema - обновлён с поддержкой addressId
- [x] Миграции - добавлен address_id в таблицу bookings
- [x] Миграции - добавлен address_id в таблицу company_slots_config с иерархией
- [x] Индексы для эффективных запросов по адресам

**Repository слой:**
- [x] Booking Repository - полная поддержка адресов, транзакций через контекст, FOR UPDATE
- [x] Config Repository - иерархическая конфигурация (service@address > address > service > global)
- [x] Метод GetConfigWithHierarchy для получения конфигурации с приоритетами

**Service слой:**
- [x] Bookings Service - CRUD операции (GetByID, GetUserBookings, GetCompanyBookings, Cancel, UpdateStatus)
- [x] Config Service - полный CRUD для конфигурации слотов с валидацией и проверкой прав

**Интеграции:**
- [x] UserService Client - получение выбранного автомобиля
- [x] SellerService Client - получение компании и услуг

**Use Cases:**
- [x] GetAvailableSlots - получение доступных слотов с учётом:
  - Рабочих часов компании
  - Текущего времени и minBookingNoticeMinutes
  - Параллельных бронирований (maxConcurrentBookings)
  - Пересечений временных интервалов (граничные случаи не считаются)
  - Генерация слотов с фиксированным шагом от начала работы
- [x] CreateBooking - создание бронирования с:
  - Сериализуемой транзакцией для предотвращения гонки
  - Валидацией всех ограничений (дата, время, слоты)
  - Денормализацией данных (услуга, автомобиль)
  - Блокировкой записей (FOR UPDATE)
  - Проверкой доступности: overlappingCount >= MaxConcurrentBookings

**Handlers слой:**
- [x] create_booking - POST /bookings (использует CreateBooking UseCase)
- [x] get_available_slots - GET /companies/{id}/addresses/{id}/available-slots (публичный, использует GetAvailableSlots UseCase)
- [x] get_booking - GET /bookings/{id} (использует Bookings Service с проверкой прав)
- [x] cancel_booking - PATCH /bookings/{id}/cancel (использует Bookings Service)
- [x] get_user_bookings - GET /users/{userId}/bookings (использует Bookings Service)
- [x] get_company_bookings - GET /companies/{id}/bookings (использует Bookings Service)
- [x] get_company_config - GET /companies/{id}/config (публичный, использует Config Service)
- [x] update_company_config - PUT /companies/{id}/config (использует Config Service)
- [x] Все сообщения для пользователей переведены на русский язык

**Важные технические изменения:**
- [x] Обновление `GetConfigRequest` - использует `*int64` вместо `int64` для опциональных параметров
- [x] Обновление `GetConfigWithHierarchy` в repository - принимает `*int64`, проверяет nil перед попытками поиска
- [x] Обновление contracts для Config Service - сигнатуры методов с `*int64`
- [x] Обновление integration clients - добавлен параметр timeout в конструкторы
- [x] Обновление UseCases - TimeProvider создаётся внутри (RealTimeProvider из contract.go)
- [x] Создание pkg/simpletxmanager - простой менеджер транзакций без метрик для сценариев без Prometheus
- [x] Обновление bookings.Service - удалены неиспользуемые зависимости (configRepo, userClient, txManager)

**Инфраструктура и конфигурация:**
- [x] pkg/simpletxmanager - простой менеджер транзакций без зависимости от метрик
- [x] Обновление config.toml - добавлены настройки для UserService и SellerService (URL, timeout)
- [x] Обновление internal/config/config.go - добавлен IntegrationConfig, валидация, переменные окружения
- [x] Создание .env и обновление .env.example - настройки для BookingService с интеграциями
- [x] Настройка Docker для межсервисного взаимодействия через host.docker.internal
- [x] Подключение всех handlers в cmd/main.go:
  - Инициализация integration clients с раздельными timeout
  - Выбор транзакционного менеджера в зависимости от включённых метрик (txmanager vs simpletxmanager)
  - Инициализация всех services и usecases с правильными зависимостями
  - Настройка 8 HTTP endpoints (2 публичных, 6 защищённых)
  - Использование lowerCamelCase для именования импортов
- [x] Успешная компиляция проекта (бинарник ~14MB)

**Миграции и БД:**
- [x] Partial unique indexes для поддержки NULL в уникальных ограничениях:
  - `uq_company_global` - для (company_id, NULL, NULL)
  - `uq_company_address` - для (company_id, address_id, NULL)
  - `uq_company_service` - для (company_id, NULL, service_id)
  - `uq_company_address_service` - для (company_id, address_id, service_id)
- [x] Обновление миграции 000002 с корректными индексами
- [x] Обновление down-миграции для корректного отката

**Интеграция с SellerService:**
- [x] Исправление JSON тегов в models.go для соответствия snake_case API:
  - `working_hours` вместо `workingHours`
  - `manager_ids` вместо `managerIds`
  - `address_ids` вместо `addressIds`
  - `company_id`, `created_at`, `updated_at` и т.д.
- [x] Обновление OpenAPI схемы smc-sellerservice.yaml
- [x] Логирование декодированных данных для отладки

**Тестовые данные:**
- [x] Исправление fixtures для company_slots_config (удаление ON CONFLICT, добавление точек с запятой)
- [x] Создание fixtures для SellerService (3 компании, 4 адреса, 5 услуг)
- [x] Создание fixtures для UserService (11 пользователей, 7 автомобилей)
- [x] Создание fixtures для PriceService (5 правил ценообразования)
- [x] Скрипт load_all_fixtures.sh для автоматической загрузки всех фикстур
- [x] Документация FIXTURES_SETUP.md с описанием структуры данных

**Тестирование:**
- [x] Создание TEST_PLAN.md с 80+ тест-кейсами
- [x] Скрипт api_requests.sh для интерактивного тестирования API
- [x] Smoke tests в Makefile
- [x] Успешная проверка endpoint /available-slots с корректными данными (4/4 бокса)

### Проверено и работает ✅
- [x] GET /companies/{id}/addresses/{id}/available-slots - возвращает корректные слоты с учётом конфигурации
- [x] Иерархическая конфигурация слотов работает корректно (приоритет: услуга@адрес > адрес > глобальная)
- [x] Интеграция с SellerService (получение компаний, услуг, рабочих часов)
- [x] Partial unique indexes предотвращают дубликаты с NULL значениями
- [x] Prometheus метрики собираются корректно

### В процессе 🚧
- [ ] **Полное тестирование API** - проверка всех 8 endpoints с различными сценариями
- [ ] **Unit и integration тесты** - покрытие тестами handlers, usecases, repositories

### Планируется 📋
- [ ] Уведомления через Telegram Bot при создании/отмене бронирования
- [ ] Напоминания о записи (за N часов до начала)
- [ ] Система отзывов о выполненных услугах
- [ ] Аналитика и статистика для компаний
- [ ] WebSocket для real-time обновлений слотов
- [ ] Интеграция с платёжными системами

## Документация

- [OpenAPI Schema](schemas/schema.yaml) - полное описание API
- [Database Migrations](migrations/README.md) - документация по миграциям
- [Test Fixtures](migrations/fixtures/README.md) - тестовые данные

## Зависимости

- **Gorilla Mux** - HTTP роутинг
- **lib/pq** - PostgreSQL драйвер
- **Prometheus client** - метрики
- **Squirrel** - SQL query builder (через pkg/psqlbuilder)
- **golang-migrate** - миграции БД

## Лицензия

MIT

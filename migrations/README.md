# Database Migrations

Миграции базы данных для BookingService с использованием [golang-migrate/migrate](https://github.com/golang-migrate/migrate).

## Структура миграций

```
migrations/
├── 000001_create_bookings_table.up.sql          # Создание таблицы бронирований
├── 000001_create_bookings_table.down.sql        # Откат таблицы бронирований
├── 000002_create_company_slots_config_table.up.sql   # Создание таблицы конфигурации
├── 000002_create_company_slots_config_table.down.sql # Откат таблицы конфигурации
├── 000003_create_triggers.up.sql                # Создание триггеров
├── 000003_create_triggers.down.sql              # Откат триггеров
└── fixtures/                                     # Тестовые данные (опционально)
    ├── 001_company_configs.sql
    ├── 002_bookings.sql
    └── README.md
```

## Таблицы базы данных

### bookings

Основная таблица бронирований с денормализованными данными для истории.

**Особенности:**
- Денормализация данных услуг и автомобилей для сохранения истории
- Индексы для быстрого поиска по пользователю, компании и дате
- Частичный индекс для проверки доступности слотов (исключает отменённые)
- Триггер автоматического обновления `updated_at`

### company_slots_config

Конфигурация слотов бронирования для компаний и услуг.

**Особенности:**
- Поддержка глобальных настроек компании (`service_id IS NULL`)
- Поддержка специфичных настроек для отдельных услуг
- Приоритет: настройка услуги > настройка компании > дефолтные значения
- Уникальное ограничение на пару `(company_id, service_id)`

**Примеры конфигурации:**

```sql
-- Глобальная настройка для компании (4 бокса)
INSERT INTO company_slots_config (company_id, service_id, max_concurrent_bookings)
VALUES (123, NULL, 4);

-- Специфичная настройка для услуги (только 2 бокса)
INSERT INTO company_slots_config (company_id, service_id, max_concurrent_bookings)
VALUES (123, 456, 2);
```

## Применение миграций

### Через Docker Compose

```bash
# Применить все миграции (автоматически при запуске docker-compose up)
docker-compose up migrate

# Применить миграции вручную
docker-compose run --rm migrate -path /migrations -database "postgres://user:password@postgres:5432/bookingservice?sslmode=disable" up
```

### Через Makefile

```bash
# Применить миграции
make migrate-up

# Откатить последнюю миграцию
make migrate-down

# Откатить все миграции
make migrate-drop

# Применить миграции заново
make migrate-fresh
```

### Через CLI (локально)

```bash
# Установить migrate CLI
brew install golang-migrate

# Применить миграции
migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5438/bookingservice?sslmode=disable" up

# Откатить последнюю миграцию
migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5438/bookingservice?sslmode=disable" down 1

# Проверить версию
migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5438/bookingservice?sslmode=disable" version
```

## Создание новой миграции

```bash
# Формат имени: {version}_{description}.up.sql и .down.sql
# Версия должна быть на 1 больше предыдущей

# Пример: создание файлов для новой миграции
touch migrations/000004_add_index_to_bookings.up.sql
touch migrations/000004_add_index_to_bookings.down.sql
```

**Важно:**
- Всегда создавайте как `.up.sql`, так и `.down.sql` файлы
- Номер версии должен быть строго последовательным
- В `.down.sql` реализуйте полный откат изменений из `.up.sql`

## Загрузка фикстур (тестовые данные)

См. [fixtures/README.md](fixtures/README.md)

```bash
# Через Makefile (если добавить команду)
make fixtures

# Вручную
docker exec -i bookingservice-db psql -U postgres -d bookingservice < migrations/fixtures/001_company_configs.sql
docker exec -i bookingservice-db psql -U postgres -d bookingservice < migrations/fixtures/002_bookings.sql
```

## Индексы и производительность

### Критичные индексы

1. **idx_bookings_availability** - частичный индекс для проверки свободных слотов
   - Используется при создании бронирования
   - Исключает отменённые записи из индекса

2. **idx_bookings_company_date_time** - составной индекс для быстрого поиска
   - Используется при получении бронирований компании
   - Используется при проверке доступности слотов

3. **idx_bookings_user_created** - для истории пользователя с сортировкой

### Мониторинг индексов

```sql
-- Проверка использования индексов
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- Неиспользуемые индексы
SELECT schemaname, tablename, indexname
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND schemaname = 'public';
```

## Триггеры

### update_updated_at_column()

Автоматически обновляет поле `updated_at` при изменении записи.

**Применяется к:**
- `bookings`
- `company_slots_config`

## Troubleshooting

### Миграция не применяется

```bash
# Проверить текущую версию
migrate -path ./migrations -database "..." version

# Форсировать установку версии (осторожно!)
migrate -path ./migrations -database "..." force VERSION_NUMBER
```

### Ошибка "Dirty database version"

```bash
# Проверить состояние
docker exec -it bookingservice-db psql -U postgres -d bookingservice -c "SELECT * FROM schema_migrations;"

# Очистить dirty state (если уверены, что миграция откачена вручную)
migrate -path ./migrations -database "..." force VERSION_NUMBER
```

### Полный сброс БД

```bash
# ВНИМАНИЕ: Удалит все данные!
docker-compose down -v
docker-compose up -d postgres
docker-compose up migrate
```

## Соглашения

1. **Именование файлов**: `{version}_{snake_case_description}.{up|down}.sql`
2. **Идемпотентность**: используйте `IF EXISTS` / `IF NOT EXISTS`
3. **Комментарии**: добавляйте `COMMENT ON` для документации схемы
4. **Транзакции**: каждая миграция выполняется в отдельной транзакции
5. **Откат**: всегда тестируйте `.down.sql` файлы

## Полезные команды PostgreSQL

```sql
-- Проверить структуру таблицы
\d bookings
\d company_slots_config

-- Список всех индексов
\di

-- Список триггеров
\dS bookings

-- Размер таблиц
SELECT pg_size_pretty(pg_total_relation_size('bookings'));
```

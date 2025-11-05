-- Создание таблицы конфигурации слотов для компаний, адресов и услуг
CREATE TABLE IF NOT EXISTS company_slots_config (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    address_id BIGINT,  -- NULL = настройка для всех адресов компании
    service_id BIGINT,  -- NULL = настройка для всех услуг

    -- Конфигурация слотов
    slot_duration_minutes INT NOT NULL DEFAULT 30,
    max_concurrent_bookings INT NOT NULL DEFAULT 1,

    -- Ограничения бронирования
    advance_booking_days INT NOT NULL DEFAULT 0,
    min_booking_notice_minutes INT NOT NULL DEFAULT 60,

    -- Аудит
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Ограничения
    CONSTRAINT chk_slot_duration CHECK (slot_duration_minutes >= 5),
    CONSTRAINT chk_max_concurrent CHECK (max_concurrent_bookings >= 1),
    CONSTRAINT chk_advance_booking CHECK (advance_booking_days >= 0),
    CONSTRAINT chk_min_notice CHECK (min_booking_notice_minutes >= 0)
);

-- Индексы для быстрого поиска с учётом иерархии
CREATE INDEX idx_company_slots_config_company ON company_slots_config(company_id);
CREATE INDEX idx_company_slots_config_address ON company_slots_config(company_id, address_id);
CREATE INDEX idx_company_slots_config_service ON company_slots_config(company_id, service_id);
CREATE INDEX idx_company_slots_config_addr_serv ON company_slots_config(company_id, address_id, service_id);

-- Уникальность с поддержкой NULL через partial unique indexes
-- В PostgreSQL NULL != NULL в обычных UNIQUE constraints, поэтому используем WHERE условия

-- 1. Глобальная конфигурация: (company_id, NULL, NULL)
CREATE UNIQUE INDEX uq_company_global
    ON company_slots_config (company_id)
    WHERE address_id IS NULL AND service_id IS NULL;

-- 2. Конфигурация для адреса: (company_id, address_id, NULL)
CREATE UNIQUE INDEX uq_company_address
    ON company_slots_config (company_id, address_id)
    WHERE address_id IS NOT NULL AND service_id IS NULL;

-- 3. Конфигурация для услуги (глобально): (company_id, NULL, service_id)
CREATE UNIQUE INDEX uq_company_service
    ON company_slots_config (company_id, service_id)
    WHERE address_id IS NULL AND service_id IS NOT NULL;

-- 4. Конфигурация для услуги на адресе: (company_id, address_id, service_id)
CREATE UNIQUE INDEX uq_company_address_service
    ON company_slots_config (company_id, address_id, service_id)
    WHERE address_id IS NOT NULL AND service_id IS NOT NULL;

-- Комментарии к таблице и столбцам
COMMENT ON TABLE company_slots_config IS 'Конфигурация слотов бронирования для компаний, адресов и услуг с поддержкой иерархии';
COMMENT ON COLUMN company_slots_config.company_id IS 'ID компании из SellerService';
COMMENT ON COLUMN company_slots_config.address_id IS 'ID адреса компании из SellerService (NULL = настройка для всех адресов компании)';
COMMENT ON COLUMN company_slots_config.service_id IS 'ID услуги из SellerService (NULL = настройка для всех услуг)';
COMMENT ON COLUMN company_slots_config.slot_duration_minutes IS 'Длительность услуги и шаг временных слотов в минутах (30, 60, 90, 120 и т.д.). Это значение используется как для генерации доступных слотов, так и для определения длительности бронирования';
COMMENT ON COLUMN company_slots_config.max_concurrent_bookings IS 'Максимальное количество одновременных бронирований (количество боксов на адресе)';
COMMENT ON COLUMN company_slots_config.advance_booking_days IS 'Ограничение на бронирование в будущем в днях (0 = без ограничений)';
COMMENT ON COLUMN company_slots_config.min_booking_notice_minutes IS 'Минимальное время до записи в минутах';

-- Примеры использования с иерархией конфигураций:
--
-- 1. Глобальная настройка для всей компании 123:
--    INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
--    VALUES (123, NULL, NULL, 3);
--
-- 2. Настройка для адреса 100 компании 123:
--    INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
--    VALUES (123, 100, NULL, 4);
--
-- 3. Настройка для услуги 456 на адресе 100 компании 123:
--    INSERT INTO company_slots_config (company_id, address_id, service_id, max_concurrent_bookings)
--    VALUES (123, 100, 456, 2);
--
-- Приоритет применения (от высшего к низшему):
-- 1. Настройка для конкретной услуги на конкретном адресе (company_id, address_id, service_id)
-- 2. Настройка для адреса (company_id, address_id, NULL)
-- 3. Глобальная настройка компании (company_id, NULL, NULL)
-- 4. Дефолтные значения из приложения

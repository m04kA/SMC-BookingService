-- Создание таблицы бронирований
CREATE TABLE IF NOT EXISTS bookings (
    id BIGSERIAL PRIMARY KEY,

    -- Внешние ссылки (не FK, т.к. данные в других сервисах)
    user_id BIGINT NOT NULL,
    company_id BIGINT NOT NULL,
    address_id BIGINT NOT NULL,
    service_id BIGINT NOT NULL,
    car_id BIGINT NOT NULL,

    -- Временные параметры
    booking_date DATE NOT NULL,
    start_time TIME NOT NULL,
    duration_minutes INT NOT NULL,  -- Длительность услуги (может быть разной: 30, 60, 90, 120 минут и т.д.)

    -- Статус бронирования
    status VARCHAR(30) NOT NULL DEFAULT 'confirmed',

    -- Денормализованные данные для истории
    -- (на случай изменения/удаления данных в других сервисах)
    service_name VARCHAR(200) NOT NULL,
    service_price DECIMAL(10,2) NOT NULL,
    car_brand VARCHAR(100),
    car_model VARCHAR(100),
    car_license_plate VARCHAR(20),

    -- Дополнительная информация
    notes TEXT,
    cancellation_reason TEXT,
    cancelled_at TIMESTAMP,

    -- Аудит
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Ограничения
    CONSTRAINT chk_status CHECK (
        status IN (
            'pending',
            'confirmed',
            'in_progress',
            'completed',
            'cancelled_by_user',
            'cancelled_by_company',
            'no_show'
        )
    ),
    CONSTRAINT chk_duration CHECK (duration_minutes > 0)
);

-- Индексы для оптимизации запросов
CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_company_date ON bookings(company_id, booking_date);
CREATE INDEX idx_bookings_company_address_date ON bookings(company_id, address_id, booking_date);
CREATE INDEX idx_bookings_company_address_date_time ON bookings(company_id, address_id, booking_date, start_time);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_user_created ON bookings(user_id, created_at DESC);

-- Композитный индекс для быстрой проверки доступности слотов на конкретном адресе
CREATE INDEX idx_bookings_availability ON bookings(company_id, address_id, booking_date, start_time, status)
WHERE status NOT IN ('cancelled_by_user', 'cancelled_by_company', 'no_show');

-- Комментарии к таблице и столбцам
COMMENT ON TABLE bookings IS 'Таблица бронирований услуг автомойки';
COMMENT ON COLUMN bookings.user_id IS 'Telegram ID пользователя из UserService';
COMMENT ON COLUMN bookings.company_id IS 'ID компании из SellerService';
COMMENT ON COLUMN bookings.address_id IS 'ID адреса компании из SellerService (компания может иметь несколько точек обслуживания)';
COMMENT ON COLUMN bookings.service_id IS 'ID услуги из SellerService';
COMMENT ON COLUMN bookings.car_id IS 'ID автомобиля из UserService';
COMMENT ON COLUMN bookings.duration_minutes IS 'Длительность услуги в минутах (может быть разной: 30, 60, 90, 120 минут). Берётся из SellerService при создании бронирования';
COMMENT ON COLUMN bookings.status IS 'Статус: pending, confirmed, in_progress, completed, cancelled_by_user, cancelled_by_company, no_show';
COMMENT ON COLUMN bookings.service_name IS 'Денормализованное название услуги для истории';
COMMENT ON COLUMN bookings.service_price IS 'Денормализованная цена услуги для истории';

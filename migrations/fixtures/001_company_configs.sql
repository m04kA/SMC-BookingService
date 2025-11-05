-- ==========================================
-- Фикстуры: Конфигурация слотов для компаний
-- ==========================================
-- Демонстрирует иерархическую систему конфигурации:
-- 1. Глобальная для компании (company_id, NULL, NULL)
-- 2. Для конкретного адреса (company_id, address_id, NULL)
-- 3. Для конкретной услуги (company_id, NULL, service_id)
-- 4. Для услуги на адресе (company_id, address_id, service_id) - наивысший приоритет

-- ==========================================
-- КОМПАНИЯ 1: Автомойка Премиум
-- ==========================================
-- Имеет 2 адреса: 100 (Тверская) и 101 (Ленина)
-- Услуги: 1 (Комплексная мойка), 2 (Экспресс-мойка), 3 (Детейлинг)

-- Глобальная конфигурация для компании 1
-- (используется по умолчанию для всех адресов и услуг)
INSERT INTO company_slots_config (
    company_id, address_id, service_id,
    slot_duration_minutes, max_concurrent_bookings,
    advance_booking_days, min_booking_notice_minutes
)
VALUES (
    1, NULL, NULL,
    30, 3, 30, 60
);

-- Конфигурация для адреса 100 (Тверская)
-- Переопределяет глобальную: 4 бокса вместо 3
INSERT INTO company_slots_config (
    company_id, address_id, service_id,
    slot_duration_minutes, max_concurrent_bookings,
    advance_booking_days, min_booking_notice_minutes
)
VALUES (
    1, 100, NULL,
    30, 4, 30, 60
);

-- Конфигурация для адреса 101 (Ленина)
-- Меньше боксов, но можно бронировать заранее без ограничений
INSERT INTO company_slots_config (
    company_id, address_id, service_id,
    slot_duration_minutes, max_concurrent_bookings,
    advance_booking_days, min_booking_notice_minutes
)
VALUES (
    1, 101, NULL,
    30, 2, 0, 60
);

-- Специфичная конфигурация для услуги 2 (Экспресс-мойка) на адресе 100
-- Экспресс-мойка занимает только 1 бокс из 4
INSERT INTO company_slots_config (
    company_id, address_id, service_id,
    slot_duration_minutes, max_concurrent_bookings,
    advance_booking_days, min_booking_notice_minutes
)
VALUES (
    1, 100, 2,
    30, 4, 30, 30
);

-- Специфичная конфигурация для услуги 3 (Детейлинг) на адресе 101
-- Детейлинг занимает оба бокса на адресе 101
INSERT INTO company_slots_config (
    company_id, address_id, service_id,
    slot_duration_minutes, max_concurrent_bookings,
    advance_booking_days, min_booking_notice_minutes
)
VALUES (
    1, 101, 3,
    60, 1, 0, 120
);

-- ==========================================
-- КОМПАНИЯ 2: СТО Профи
-- ==========================================
-- Имеет 1 адрес: 200 (Новая)
-- Работает круглосуточно, только 1 бокс

-- Глобальная конфигурация для компании 2
-- Только 1 бокс, можно бронировать без ограничений по дате, за 2 часа
INSERT INTO company_slots_config (
    company_id, address_id, service_id,
    slot_duration_minutes, max_concurrent_bookings,
    advance_booking_days, min_booking_notice_minutes
)
VALUES (
    2, NULL, NULL,
    45, 1, 0, 120
);

-- ==========================================
-- КОМПАНИЯ 3: Детейлинг Центр
-- ==========================================
-- Имеет 1 адрес: 300 (Невский пр.)
-- 6 боксов, но бронировать можно только на 14 дней вперёд

-- Глобальная конфигурация для компании 3
INSERT INTO company_slots_config (
    company_id, address_id, service_id,
    slot_duration_minutes, max_concurrent_bookings,
    advance_booking_days, min_booking_notice_minutes
)
VALUES (
    3, NULL, NULL,
    60, 6, 14, 30
);

-- ==========================================
-- ИТОГО конфигураций: 7
-- ==========================================
-- Компания 1: 5 конфигураций (глобальная + 2 адреса + 2 услуги@адрес)
-- Компания 2: 1 конфигурация (глобальная)
-- Компания 3: 1 конфигурация (глобальная)

-- Примеры приоритетов для компании 1:
-- 1. Услуга 2 на адресе 100 → (1, 100, 2) = 4 бокса, 30 мин уведомления
-- 2. Услуга 1 на адресе 100 → (1, 100, NULL) = 4 бокса, 60 мин уведомления
-- 3. Услуга 3 на адресе 101 → (1, 101, 3) = 1 бокс, 120 мин уведомления
-- 4. Услуга 1 на адресе 101 → (1, 101, NULL) = 2 бокса, 60 мин уведомления
-- 5. Несуществующий адрес → (1, NULL, NULL) = 3 бокса, 60 мин уведомления

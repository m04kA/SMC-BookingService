-- ==========================================
-- Фикстуры: Тестовые бронирования
-- ==========================================
-- Создаёт разнообразные бронирования для тестирования:
-- - Разные статусы (confirmed, completed, cancelled, in_progress, pending, no_show)
-- - Разные компании и адреса
-- - Денормализованные данные (услуги, автомобили)
-- - Параллельные бронирования в один слот
-- - Бронирования в прошлом, настоящем и будущем

-- ==========================================
-- КОМПАНИЯ 1, АДРЕС 100 (Тверская)
-- ==========================================

-- Бронирование 1: Подтверждённое на завтра, 10:00-11:00
-- Для тестирования параллельных бронирований (1/4 боксов занято)
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate,
    notes
)
VALUES (
    123456789, 1, 100, 1, 1001,
    CURRENT_DATE + INTERVAL '1 day', '10:00', 60,
    'confirmed', 'Комплексная мойка', 1500.00,
    'BMW', 'X5', 'А123БВ799',
    'Пожалуйста, уделите внимание дискам'
);

-- Бронирование 2: Ещё одно подтверждённое на завтра, 10:00-11:00 (параллельное)
-- Для тестирования параллельных бронирований (2/4 боксов занято)
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    987654321, 1, 100, 1, 2001,
    CURRENT_DATE + INTERVAL '1 day', '10:00', 60,
    'confirmed', 'Комплексная мойка', 1500.00,
    'Mercedes', 'E-Class', 'В999КС777'
);

-- Бронирование 3: Экспресс-мойка на завтра, 14:00-14:30
-- Для тестирования разных услуг на одном адресе
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    111222333, 1, 100, 2, 3001,
    CURRENT_DATE + INTERVAL '1 day', '14:00', 30,
    'confirmed', 'Экспресс-мойка', 800.00,
    'Audi', 'A4', 'С555АА199'
);

-- Бронирование 4: Выполненное (история пользователя 123456789)
-- Неделю назад
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    123456789, 1, 100, 2, 1001,
    CURRENT_DATE - INTERVAL '7 days', '11:00', 30,
    'completed', 'Экспресс-мойка', 800.00,
    'BMW', 'X5', 'А123БВ799'
);

-- Бронирование 5: Полностью занятый слот (3/4 боксов)
-- Для тестирования TC-2.4 (проверка maxConcurrentBookings)
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    444555666, 1, 100, 1, 4001,
    CURRENT_DATE + INTERVAL '1 day', '16:00', 60,
    'confirmed', 'Комплексная мойка', 1500.00,
    'Tesla', 'Model 3', 'Т123КХ777'
);

-- Бронирование 6: Полностью занятый слот (4/4 боксов)
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    555666777, 1, 100, 1, 5001,
    CURRENT_DATE + INTERVAL '1 day', '16:00', 60,
    'confirmed', 'Комплексная мойка', 1500.00,
    'Volkswagen', 'Polo', 'О777ОО799'
);

-- ==========================================
-- КОМПАНИЯ 1, АДРЕС 101 (Ленина)
-- ==========================================

-- Бронирование 7: Детейлинг на адресе 101
-- Для тестирования разных адресов одной компании
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate,
    notes
)
VALUES (
    123456789, 1, 101, 3, 1001,
    CURRENT_DATE + INTERVAL '2 days', '10:00', 120,
    'confirmed', 'Детейлинг', 5000.00,
    'BMW', 'X5', 'А123БВ799',
    'Требуется полировка кузова'
);

-- Бронирование 8: Отменённое компанией на адресе 101
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate,
    cancellation_reason, cancelled_at
)
VALUES (
    666777888, 1, 101, 3, 6001,
    CURRENT_DATE + INTERVAL '3 days', '15:00', 120,
    'cancelled_by_company', 'Детейлинг', 5000.00,
    'Porsche', 'Cayenne', 'Н123МР777',
    'Технические работы на мойке', NOW()
);

-- ==========================================
-- КОМПАНИЯ 2, АДРЕС 200 (СТО Профи)
-- ==========================================

-- Бронирование 9: Отменённое пользователем
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate,
    cancellation_reason, cancelled_at
)
VALUES (
    987654321, 2, 200, 10, 2001,
    CURRENT_DATE - INTERVAL '2 days', '16:00', 45,
    'cancelled_by_user', 'Замена масла', 2000.00,
    'Mercedes', 'E-Class', 'В999КС777',
    'Изменились планы', CURRENT_DATE - INTERVAL '3 days'
);

-- Бронирование 10: Клиент не явился
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    555666777, 2, 200, 10, 5001,
    CURRENT_DATE - INTERVAL '1 day', '09:00', 45,
    'no_show', 'Замена масла', 2000.00,
    'Volkswagen', 'Polo', 'О777ОО799'
);

-- Бронирование 11: В процессе выполнения (сегодня)
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    444555666, 2, 200, 10, 4001,
    CURRENT_DATE, '12:00', 45,
    'in_progress', 'Замена масла', 2000.00,
    'Tesla', 'Model 3', 'Т123КХ777'
);

-- ==========================================
-- КОМПАНИЯ 3, АДРЕС 300 (Детейлинг Центр)
-- ==========================================

-- Бронирование 12: Ожидает подтверждения
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    777888999, 3, 300, 20, 7001,
    CURRENT_DATE + INTERVAL '5 days', '10:00', 180,
    'pending', 'Полировка кузова', 8000.00,
    'Lexus', 'RX350', 'К888КК199'
);

-- Бронирование 13: Подтверждённое на 10 дней вперёд
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    123456789, 3, 300, 20, 1001,
    CURRENT_DATE + INTERVAL '10 days', '14:00', 180,
    'confirmed', 'Полировка кузова', 8000.00,
    'BMW', 'X5', 'А123БВ799'
);

-- ==========================================
-- Дополнительные бронирования для пользователя 123456789
-- (для тестирования истории пользователя)
-- ==========================================

-- Бронирование 14: Выполненное 2 недели назад
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    123456789, 1, 100, 1, 1001,
    CURRENT_DATE - INTERVAL '14 days', '10:00', 60,
    'completed', 'Комплексная мойка', 1500.00,
    'BMW', 'X5', 'А123БВ799'
);

-- Бронирование 15: Выполненное месяц назад
INSERT INTO bookings (
    user_id, company_id, address_id, service_id, car_id,
    booking_date, start_time, duration_minutes,
    status, service_name, service_price,
    car_brand, car_model, car_license_plate
)
VALUES (
    123456789, 2, 200, 10, 1001,
    CURRENT_DATE - INTERVAL '30 days', '15:00', 45,
    'completed', 'Замена масла', 2000.00,
    'BMW', 'X5', 'А123БВ799'
);

-- ==========================================
-- ИТОГО бронирований: 15
-- ==========================================
-- По компаниям:
--   Компания 1, адрес 100: 6 бронирований
--   Компания 1, адрес 101: 2 бронирования
--   Компания 2, адрес 200: 3 бронирования
--   Компания 3, адрес 300: 2 бронирования
--   Пользователь 123456789: 2 дополнительных в истории

-- По статусам:
--   confirmed: 7
--   completed: 3
--   cancelled_by_user: 1
--   cancelled_by_company: 1
--   no_show: 1
--   in_progress: 1
--   pending: 1

-- Особенности:
--   - Параллельные бронирования: 2 на 10:00 завтра (адрес 100)
--   - Полностью занятый слот: 4/4 боксов на 16:00 завтра (адрес 100)
--   - Разные адреса одной компании (100 и 101)
--   - История пользователя 123456789: 6 бронирований (4 completed, 2 confirmed)

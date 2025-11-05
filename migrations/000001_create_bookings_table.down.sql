-- Откат миграции: удаление таблицы бронирований
DROP INDEX IF EXISTS idx_bookings_availability;
DROP INDEX IF EXISTS idx_bookings_user_created;
DROP INDEX IF EXISTS idx_bookings_status;
DROP INDEX IF EXISTS idx_bookings_company_date_time;
DROP INDEX IF EXISTS idx_bookings_company_date;
DROP INDEX IF EXISTS idx_bookings_user_id;

DROP TABLE IF EXISTS bookings;

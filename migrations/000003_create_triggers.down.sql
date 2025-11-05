-- Откат миграции: удаление триггеров и функции

DROP TRIGGER IF EXISTS tr_company_slots_config_updated_at ON company_slots_config;
DROP TRIGGER IF EXISTS tr_bookings_updated_at ON bookings;

DROP FUNCTION IF EXISTS update_updated_at_column();

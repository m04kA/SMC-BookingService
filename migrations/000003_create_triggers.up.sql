-- Создание триггера для автоматического обновления updated_at

-- Функция для обновления поля updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер для таблицы bookings
CREATE TRIGGER tr_bookings_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Триггер для таблицы company_slots_config
CREATE TRIGGER tr_company_slots_config_updated_at
    BEFORE UPDATE ON company_slots_config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON FUNCTION update_updated_at_column() IS 'Функция для автоматического обновления поля updated_at';

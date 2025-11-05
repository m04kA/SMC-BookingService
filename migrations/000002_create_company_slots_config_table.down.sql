-- Откат миграции: удаление таблицы конфигурации слотов

-- Удаление unique indexes
DROP INDEX IF EXISTS uq_company_address_service;
DROP INDEX IF EXISTS uq_company_service;
DROP INDEX IF EXISTS uq_company_address;
DROP INDEX IF EXISTS uq_company_global;

-- Удаление обычных indexes
DROP INDEX IF EXISTS idx_company_slots_config_addr_serv;
DROP INDEX IF EXISTS idx_company_slots_config_service;
DROP INDEX IF EXISTS idx_company_slots_config_address;
DROP INDEX IF EXISTS idx_company_slots_config_company;

-- Удаление таблицы
DROP TABLE IF EXISTS company_slots_config;

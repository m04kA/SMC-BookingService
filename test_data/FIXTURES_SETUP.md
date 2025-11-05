# SMC Ecosystem - Test Fixtures Setup Guide

Руководство по загрузке тестовых данных для всех микросервисов экосистемы SMC.

## Обзор

Для полноценного тестирования BookingService требуются данные в 4 сервисах:

```
┌─────────────────┐      ┌──────────────────┐      ┌─────────────────┐
│  UserService    │      │  SellerService   │      │  PriceService   │
│  (port 8080)    │      │  (port 8081)     │      │  (port 8082)    │
│                 │      │                  │      │                 │
│ 11 пользователей│      │ 3 компании       │      │ 5 правил        │
│ 7 автомобилей   │      │ 4 адреса         │      │ ценообразования │
│                 │      │ 5 услуг          │      │                 │
└────────┬────────┘      └────────┬─────────┘      └────────┬────────┘
         │                        │                         │
         └────────────────────────┼─────────────────────────┘
                                  │
                         ┌────────▼────────┐
                         │ BookingService  │
                         │  (port 8083)    │
                         │                 │
                         │ 7 конфигураций  │
                         │ 15 бронирований │
                         └─────────────────┘
```

---

## Быстрый старт

### Автоматическая загрузка (рекомендуется)

```bash
# Из корня BookingService
chmod +x test_data/load_all_fixtures.sh
./test_data/load_all_fixtures.sh
```

Скрипт автоматически:
1. Проверит доступность всех сервисов
2. Загрузит фикстуры в правильном порядке
3. Выведет отчёт об успешности

### Ручная загрузка

Загрузите фикстуры в следующем порядке:

```bash
# 1. SellerService (компании, адреса, услуги)
docker exec -i sellerservice-db psql -U postgres -d smk_sellerservice \
  < ~/GolandProjects/SMK-SellerService/migrations/fixtures/001_test_companies.sql

# 2. UserService (пользователи, автомобили)
docker exec -i userservice-db psql -U postgres -d smk_userservice \
  < ~/GolandProjects/SMK-UserService/migrations/fixtures/001_test_users.sql

# 3. PriceService (правила ценообразования)
docker exec -i priceservice-db psql -U postgres -d smc_priceservice \
  < ~/GolandProjects/SMC-PriceService/migrations/fixtures/001_test_pricing_rules.sql

# 4. BookingService (конфигурация слотов, бронирования)
docker exec -i bookingservice-db psql -U postgres -d smk_bookingservice \
  < migrations/fixtures/001_company_configs.sql
docker exec -i bookingservice-db psql -U postgres -d smk_bookingservice \
  < migrations/fixtures/002_bookings.sql
```

---

## Детальное описание данных

### 1. SellerService

**Файл**: `SMK-SellerService/migrations/fixtures/001_test_companies.sql`

**Содержимое**:
- 3 компании (Автомойка Премиум, СТО Профи, Детейлинг Центр)
- 4 адреса (100, 101, 200, 300)
- 5 услуг (1, 2, 3, 10, 20)
- Рабочие часы для каждой компании

**Ключевые особенности**:
- Услуга 1 (Комплексная мойка) доступна на адресах 100 и 101
- Услуга 2 (Экспресс-мойка) доступна ТОЛЬКО на адресе 100
- Услуга 3 (Детейлинг) доступна ТОЛЬКО на адресе 101
- Компания 1: выходной в воскресенье
- Компания 2: работает круглосуточно

**Проверка**:
```bash
docker exec -it sellerservice-db psql -U postgres -d smk_sellerservice \
  -c "SELECT id, name FROM companies;"
```

### 2. UserService

**Файл**: `SMK-UserService/migrations/fixtures/001_test_users.sql`

**Содержимое**:
- 11 пользователей:
  - 7 обычных с автомобилями
  - 3 менеджера компаний
  - 1 без автомобиля (для негативных тестов)
- 7 автомобилей с `is_selected = true`

**Ключевые пользователи**:
- `123456789`: Иван Петров (BMW X5, класс L) - основной тестовый пользователь
- `987654321`: Мария Сидорова (Mercedes E-Class, класс E)
- `777777777`: Менеджер компании 1
- `999999999`: Без автомобиля (для теста TC-2.7)

**Проверка**:
```bash
docker exec -it userservice-db psql -U postgres -d smk_userservice \
  -c "SELECT u.tg_user_id, u.name, c.brand, c.model FROM users u LEFT JOIN cars c ON u.tg_user_id = c.user_id WHERE c.is_selected = true;"
```

### 3. PriceService

**Файл**: `SMC-PriceService/migrations/fixtures/001_test_pricing_rules.sql`

**Содержимое**:
- 5 правил ценообразования для 5 услуг

**Типы ценообразования**:
1. **Static** (Услуга 2): фиксированная цена 800₽
2. **Multiplier** (Услуги 1, 20): базовая цена × множитель класса
3. **Fixed** (Услуги 3, 10): фиксированная цена для каждого класса

**Примеры цен**:
- Комплексная мойка (услуга 1), BMW X5 (L): 1000₽
- Комплексная мойка (услуга 1), Audi A4 (D): 1500₽
- Экспресс-мойка (услуга 2), любой класс: 800₽

**Проверка**:
```bash
docker exec -it priceservice-db psql -U postgres -d smc_priceservice \
  -c "SELECT company_id, service_id, pricing_type, base_price FROM pricing_rules;"
```

### 4. BookingService

**Файлы**:
- `migrations/fixtures/001_company_configs.sql` - конфигурация слотов
- `migrations/fixtures/002_bookings.sql` - тестовые бронирования

**Содержимое**:
- 7 конфигураций слотов (иерархическая система)
- 15 бронирований с разными статусами

**Иерархия конфигураций**:
1. Глобальная: (company_id, NULL, NULL)
2. Для адреса: (company_id, address_id, NULL)
3. Для услуги на адресе: (company_id, address_id, service_id) - **наивысший приоритет**

**Проверка**:
```bash
docker exec -it bookingservice-db psql -U postgres -d smk_bookingservice \
  -c "SELECT company_id, address_id, service_id, max_concurrent_bookings FROM company_slots_config ORDER BY company_id, address_id NULLS FIRST, service_id NULLS FIRST;"
```

---

## Связи между сервисами

### Компания 1 (ID: 1)
| Сервис | Данные |
|--------|--------|
| SellerService | Компания "Автомойка Премиум", адреса 100 и 101, услуги 1, 2, 3 |
| BookingService | 8 бронирований (6 на адресе 100, 2 на адресе 101) |
| PriceService | Правила для услуг 1, 2, 3 |
| UserService | Менеджер 777777777 |

### Компания 2 (ID: 2)
| Сервис | Данные |
|--------|--------|
| SellerService | Компания "СТО Профи", адрес 200, услуга 10 |
| BookingService | 3 бронирования на адресе 200 |
| PriceService | Правила для услуги 10 |
| UserService | Менеджер 888888888 |

### Компания 3 (ID: 3)
| Сервис | Данные |
|--------|--------|
| SellerService | Компания "Детейлинг Центр", адрес 300, услуга 20 |
| BookingService | 2 бронирования на адресе 300 |
| PriceService | Правила для услуги 20 |
| UserService | Менеджер 999999000 |

---

## Проверка целостности данных

### Скрипт для полной проверки

```bash
#!/bin/bash

echo "=== Проверка SellerService ==="
docker exec -it sellerservice-db psql -U postgres -d smk_sellerservice \
  -c "SELECT COUNT(*) as companies FROM companies;" \
  -c "SELECT COUNT(*) as addresses FROM addresses;" \
  -c "SELECT COUNT(*) as services FROM services;"

echo -e "\n=== Проверка UserService ==="
docker exec -it userservice-db psql -U postgres -d smk_userservice \
  -c "SELECT COUNT(*) as users FROM users;" \
  -c "SELECT COUNT(*) as cars FROM cars WHERE is_selected = true;"

echo -e "\n=== Проверка PriceService ==="
docker exec -it priceservice-db psql -U postgres -d smc_priceservice \
  -c "SELECT COUNT(*) as pricing_rules FROM pricing_rules;"

echo -e "\n=== Проверка BookingService ==="
docker exec -it bookingservice-db psql -U postgres -d smk_bookingservice \
  -c "SELECT COUNT(*) as configs FROM company_slots_config;" \
  -c "SELECT COUNT(*) as bookings FROM bookings;"
```

**Ожидаемый вывод**:
```
=== Проверка SellerService ===
 companies
-----------
         3

 addresses
-----------
         4

 services
----------
        5

=== Проверка UserService ===
 users
-------
    11

 cars
------
     7

=== Проверка PriceService ===
 pricing_rules
---------------
             5

=== Проверка BookingService ===
 configs
---------
       7

 bookings
----------
       15
```

---

## Сброс всех данных

```bash
# SellerService
docker exec -it sellerservice-db psql -U postgres -d smk_sellerservice \
  -c "TRUNCATE companies CASCADE;"

# UserService
docker exec -it userservice-db psql -U postgres -d smk_userservice \
  -c "TRUNCATE users CASCADE;"

# PriceService
docker exec -it priceservice-db psql -U postgres -d smc_priceservice \
  -c "TRUNCATE pricing_rules CASCADE;"

# BookingService
docker exec -it bookingservice-db psql -U postgres -d smk_bookingservice \
  -c "TRUNCATE bookings CASCADE; TRUNCATE company_slots_config CASCADE;"
```

После сброса загрузите фикстуры заново.

---

## Troubleshooting

### Ошибка: "relation does not exist"

**Причина**: Миграции не применены.

**Решение**:
```bash
# В каждом сервисе
make migrate-up
# или
make db-reset
```

### Ошибка: "duplicate key value"

**Причина**: Фикстуры уже загружены.

**Решение**: Фикстуры используют `ON CONFLICT DO UPDATE`, поэтому ошибки быть не должно. Если она есть, проверьте, что миграции применены корректно.

### Ошибка: "foreign key violation"

**Причина**: Фикстуры загружены в неправильном порядке.

**Решение**: Загрузите в порядке: SellerService → UserService → PriceService → BookingService.

### Сервис недоступен

**Причина**: Docker контейнер не запущен.

**Решение**:
```bash
# Проверить статус
docker-compose ps

# Запустить сервис
docker-compose up -d <service-name>-db
```

---

## Дополнительные ресурсы

- [SellerService Fixtures](../../../SMK-SellerService/migrations/fixtures/README.md)
- [UserService Fixtures](../../../SMK-UserService/migrations/fixtures/README.md)
- [PriceService Fixtures](../../../SMC-PriceService/migrations/fixtures/README.md)
- [BookingService Fixtures](../../migrations/fixtures/README.md)
- [BookingService Test Plan](TEST_PLAN.md)

---

## Поддержка

Если возникли проблемы с загрузкой фикстур:
1. Проверьте, что все сервисы запущены: `docker-compose ps`
2. Проверьте логи: `docker-compose logs <service-name>`
3. Проверьте миграции: `make migrate-up` в каждом сервисе
4. Сбросьте данные и загрузите заново

**Happy Testing! 🚀**

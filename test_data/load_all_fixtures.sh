#!/bin/bash

# ==========================================
# SMC Ecosystem - Load All Fixtures
# ==========================================
# Автоматически загружает тестовые данные во все сервисы
#
# Использование:
#   chmod +x test_data/load_all_fixtures.sh
#   ./test_data/load_all_fixtures.sh

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Пути к проектам
SELLER_SERVICE_PATH="../SMK-SellerService"
USER_SERVICE_PATH="../SMK-UserService"
PRICE_SERVICE_PATH="../SMC-PriceService"
BOOKING_SERVICE_PATH="."

# Имена контейнеров БД
SELLER_DB="smk-sellerservice-db"
USER_DB="smk-userservice-db"
PRICE_DB="smc-priceservice-db"
BOOKING_DB="bookingservice-db"

# Функции для вывода
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Проверка доступности контейнера БД
check_db() {
    local container_name=$1
    if ! docker ps | grep -q "$container_name"; then
        print_error "Контейнер $container_name не запущен"
        return 1
    fi
    print_success "Контейнер $container_name доступен"
    return 0
}

# Загрузка SQL фикстур
load_fixture() {
    local container_name=$1
    local db_name=$2
    local fixture_path=$3
    local description=$4

    echo -n "Загрузка $description... "

    if [ ! -f "$fixture_path" ]; then
        print_error "Файл не найден: $fixture_path"
        return 1
    fi

    if docker exec -i "$container_name" psql -U postgres -d "$db_name" < "$fixture_path" > /dev/null 2>&1; then
        print_success "OK"
        return 0
    else
        print_error "Ошибка загрузки"
        return 1
    fi
}

# Проверка количества записей
check_count() {
    local container_name=$1
    local db_name=$2
    local table=$3
    local expected=$4

    local count=$(docker exec "$container_name" psql -U postgres -d "$db_name" -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null | tr -d ' ')

    if [ "$count" == "$expected" ]; then
        print_success "$table: $count/$expected записей"
        return 0
    else
        print_warning "$table: $count/$expected записей (ожидалось $expected)"
        return 1
    fi
}

# Главная функция
main() {
    print_header "SMC Ecosystem - Loading Test Fixtures"

    local all_ok=true

    # ==========================================
    # Проверка доступности БД
    # ==========================================
    print_header "1. Проверка доступности сервисов"

    check_db "$SELLER_DB" || all_ok=false
    check_db "$USER_DB" || all_ok=false
    check_db "$PRICE_DB" || all_ok=false
    check_db "$BOOKING_DB" || all_ok=false

    if [ "$all_ok" != true ]; then
        print_error "Некоторые сервисы недоступны. Запустите: docker-compose up -d"
        exit 1
    fi

    # ==========================================
    # Загрузка фикстур
    # ==========================================
    print_header "2. Загрузка фикстур"

    echo "Порядок загрузки: SellerService → UserService → PriceService → BookingService"
    echo ""

    # SellerService
    echo "[1/4] SellerService"
    load_fixture "$SELLER_DB" "smk_sellerservice" \
        "$SELLER_SERVICE_PATH/migrations/fixtures/001_test_companies.sql" \
        "Компании, адреса, услуги" || all_ok=false

    sleep 1

    # UserService
    echo "[2/4] UserService"
    load_fixture "$USER_DB" "smk_userservice" \
        "$USER_SERVICE_PATH/migrations/fixtures/001_test_users.sql" \
        "Пользователи, автомобили" || all_ok=false

    sleep 1

    # PriceService
    echo "[3/4] PriceService"
    load_fixture "$PRICE_DB" "smc_priceservice" \
        "$PRICE_SERVICE_PATH/migrations/fixtures/001_test_pricing_rules.sql" \
        "Правила ценообразования" || all_ok=false

    sleep 1

    # BookingService
    echo "[4/4] BookingService"
    load_fixture "$BOOKING_DB" "smk_bookingservice" \
        "$BOOKING_SERVICE_PATH/migrations/fixtures/001_company_configs.sql" \
        "Конфигурация слотов" || all_ok=false

    load_fixture "$BOOKING_DB" "smk_bookingservice" \
        "$BOOKING_SERVICE_PATH/migrations/fixtures/002_bookings.sql" \
        "Тестовые бронирования" || all_ok=false

    # ==========================================
    # Проверка загруженных данных
    # ==========================================
    print_header "3. Проверка загруженных данных"

    echo "SellerService:"
    check_count "$SELLER_DB" "smk_sellerservice" "companies" "3"
    check_count "$SELLER_DB" "smk_sellerservice" "addresses" "4"
    check_count "$SELLER_DB" "smk_sellerservice" "services" "5"

    echo ""
    echo "UserService:"
    check_count "$USER_DB" "smk_userservice" "users" "11"
    docker exec "$USER_DB" psql -U postgres -d smk_userservice -t -c "SELECT COUNT(*) FROM cars WHERE is_selected = true;" > /tmp/cars_count 2>/dev/null
    local cars_selected=$(cat /tmp/cars_count | tr -d ' ')
    if [ "$cars_selected" == "7" ]; then
        print_success "cars (selected): 7/7 записей"
    else
        print_warning "cars (selected): $cars_selected/7 записей"
        all_ok=false
    fi

    echo ""
    echo "PriceService:"
    check_count "$PRICE_DB" "smc_priceservice" "pricing_rules" "5"

    echo ""
    echo "BookingService:"
    check_count "$BOOKING_DB" "smk_bookingservice" "company_slots_config" "7"
    check_count "$BOOKING_DB" "smk_bookingservice" "bookings" "15"

    # ==========================================
    # Итоги
    # ==========================================
    print_header "4. Итоги"

    if [ "$all_ok" == true ]; then
        print_success "Все фикстуры успешно загружены!"
        echo ""
        echo "Тестовые данные доступны:"
        echo "  - SellerService: 3 компании, 4 адреса, 5 услуг"
        echo "  - UserService: 11 пользователей, 7 автомобилей"
        echo "  - PriceService: 5 правил ценообразования"
        echo "  - BookingService: 7 конфигураций, 15 бронирований"
        echo ""
        echo "Можно начинать тестирование:"
        echo "  make test-api"
        echo "  make test-smoke"
        echo ""
        exit 0
    else
        print_error "Некоторые фикстуры не загружены или данные неполные"
        echo ""
        echo "Рекомендации:"
        echo "  1. Проверьте логи: docker-compose logs <service-name>"
        echo "  2. Проверьте миграции: make migrate-up"
        echo "  3. Сбросьте БД: make db-reset && make fixtures"
        echo ""
        exit 1
    fi
}

# Проверка зависимостей
if ! command -v docker > /dev/null 2>&1; then
    print_error "Docker не установлен"
    exit 1
fi

# Запуск
main

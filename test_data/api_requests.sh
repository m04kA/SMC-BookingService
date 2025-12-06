#!/bin/bash

# ==========================================
# BookingService API Test Commands
# ==========================================
# Набор curl команд для тестирования всех endpoints согласно TEST_PLAN.md
#
# Использование:
#   chmod +x test_data/api_requests.sh
#   ./test_data/api_requests.sh
#
# Или вызывать отдельные функции:
#   source test_data/api_requests.sh
#   tc_1_1  # Запустить конкретный тест
#
# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
BASE_URL="http://localhost:8083"
TOMORROW=$(date -v+1d +%Y-%m-%d 2>/dev/null || date -d tomorrow +%Y-%m-%d)
TODAY=$(date +%Y-%m-%d)
YESTERDAY=$(date -v-1d +%Y-%m-%d 2>/dev/null || date -d yesterday +%Y-%m-%d)
DAY_AFTER_TOMORROW=$(date -v+2d +%Y-%m-%d 2>/dev/null || date -d "+2 days" +%Y-%m-%d)
# Следующий понедельник (рабочий день для компании 1, которая не работает в воскресенье)
NEXT_MONDAY=$(date -v+mon +%Y-%m-%d 2>/dev/null || date -d "next monday" +%Y-%m-%d)

# Функции для вывода
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_test() {
    echo -e "${YELLOW}[TEST] $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# ==========================================
# 1. Available Slots (Получение свободных слотов)
# ==========================================

# TC-1.1: Получение слотов на ближайший рабочий день (понедельник)
tc_1_1() {
    print_test "TC-1.1: Получение слотов на ближайший понедельник (компания 1, адрес 100, услуга 1)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/100/available-slots?serviceId=1&date=${NEXT_MONDAY}" \
      -H "Content-Type: application/json" | jq .
}

# TC-1.2: Получение слотов на сегодня (учёт minBookingNoticeMinutes)
tc_1_2() {
    print_test "TC-1.2: Получение слотов на сегодня (проверка minBookingNoticeMinutes)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/100/available-slots?serviceId=1&date=${TODAY}" \
      -H "Content-Type: application/json" | jq .
}

# TC-1.3: Получение слотов в выходной день (воскресенье)
tc_1_3() {
    print_test "TC-1.3: Получение слотов в выходной день"
    # Найти ближайшее воскресенье
    SUNDAY=$(date -v+sun +%Y-%m-%d 2>/dev/null || date -d "next sunday" +%Y-%m-%d)
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/100/available-slots?serviceId=1&date=${SUNDAY}" \
      -H "Content-Type: application/json" | jq .
}

# TC-1.4: Получение слотов для услуги с индивидуальной конфигурацией
tc_1_4() {
    print_test "TC-1.4: Слоты для услуги 2 на адресе 100 (индивидуальная конфигурация)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/100/available-slots?serviceId=2&date=${TOMORROW}" \
      -H "Content-Type: application/json" | jq .
}

# TC-1.6: Несуществующая компания
tc_1_6() {
    print_test "TC-1.6: Несуществующая компания (ожидается 404)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/99999/addresses/100/available-slots?serviceId=1&date=${TOMORROW}" \
      -H "Content-Type: application/json" -w "\nHTTP Code: %{http_code}\n"
}

# TC-1.7: Адрес не принадлежит компании
tc_1_7() {
    print_test "TC-1.7: Адрес не принадлежит компании (ожидается 404)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/999/available-slots?serviceId=1&date=${TOMORROW}" \
      -H "Content-Type: application/json" -w "\nHTTP Code: %{http_code}\n"
}

# TC-1.8: Услуга не доступна на этом адресе
tc_1_8() {
    print_test "TC-1.8: Услуга 3 недоступна на адресе 100 (ожидается 400)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/100/available-slots?serviceId=3&date=${TOMORROW}" \
      -H "Content-Type: application/json" -w "\nHTTP Code: %{http_code}\n"
}

# TC-1.9: Невалидная дата
tc_1_9() {
    print_test "TC-1.9: Невалидная дата (ожидается 400)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/100/available-slots?serviceId=1&date=invalid-date" \
      -H "Content-Type: application/json" -w "\nHTTP Code: %{http_code}\n"
}

# TC-1.10: Дата в прошлом
tc_1_10() {
    print_test "TC-1.10: Дата в прошлом (пустой массив слотов)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/1/addresses/100/available-slots?serviceId=1&date=${YESTERDAY}" \
      -H "Content-Type: application/json" | jq .
}

# ==========================================
# 2. Create Booking (Создание бронирования)
# ==========================================

# TC-2.1: Успешное создание бронирования
tc_2_1() {
    print_test "TC-2.1: Успешное создание бронирования (ожидается 201)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 123456789" \
      -d "{
        \"userId\": 123456789,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"${NEXT_MONDAY}\",
        \"startTime\": \"11:00\"
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# TC-2.2: Создание с notes
tc_2_2() {
    print_test "TC-2.2: Создание бронирования с notes"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 123456789" \
      -d "{
        \"userId\": 123456789,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"${DAY_AFTER_TOMORROW}\",
        \"startTime\": \"14:00\",
        \"notes\": \"Пожалуйста, уделите внимание дискам\"
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# TC-2.3: Параллельное бронирование
tc_2_3() {
    print_test "TC-2.3: Параллельное бронирование (2 бокса свободно)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 987654321" \
      -d "{
        \"userId\": 987654321,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"${TOMORROW}\",
        \"startTime\": \"11:00\"
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# TC-2.4: Слот полностью занят
tc_2_4() {
    print_test "TC-2.4: Попытка забронировать полностью занятый слот (ожидается 409)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 111222333" \
      -d "{
        \"userId\": 111222333,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"${TOMORROW}\",
        \"startTime\": \"16:00\"
      }" -w "\nHTTP Code: %{http_code}\n"
}

# TC-2.7: Пользователь без выбранного автомобиля
tc_2_7() {
    print_test "TC-2.7: Пользователь без выбранного автомобиля (ожидается 400)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 999999999" \
      -d "{
        \"userId\": 999999999,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"${TOMORROW}\",
        \"startTime\": \"15:00\"
      }" -w "\nHTTP Code: %{http_code}\n"
}

# TC-2.8: Несуществующая компания
tc_2_8() {
    print_test "TC-2.8: Несуществующая компания (ожидается 404)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 123456789" \
      -d "{
        \"userId\": 123456789,
        \"companyId\": 99999,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"${TOMORROW}\",
        \"startTime\": \"15:00\"
      }" -w "\nHTTP Code: %{http_code}\n"
}

# TC-2.10: Услуга не доступна на этом адресе
tc_2_10() {
    print_test "TC-2.10: Услуга 3 недоступна на адресе 100 (ожидается 400)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 123456789" \
      -d "{
        \"userId\": 123456789,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 3,
        \"bookingDate\": \"${TOMORROW}\",
        \"startTime\": \"15:00\"
      }" -w "\nHTTP Code: %{http_code}\n"
}

# TC-2.14: Невалидный формат даты
tc_2_14() {
    print_test "TC-2.14: Невалидный формат даты (ожидается 400)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 123456789" \
      -d "{
        \"userId\": 123456789,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"2025-13-45\",
        \"startTime\": \"10:00\"
      }" -w "\nHTTP Code: %{http_code}\n"
}

# TC-2.15: Невалидный формат времени
tc_2_15() {
    print_test "TC-2.15: Невалидный формат времени (ожидается 400)"
    curl -s -X POST "${BASE_URL}/api/v1/bookings" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 123456789" \
      -d "{
        \"userId\": 123456789,
        \"companyId\": 1,
        \"addressId\": 100,
        \"serviceId\": 1,
        \"bookingDate\": \"${TOMORROW}\",
        \"startTime\": \"25:99\"
      }" -w "\nHTTP Code: %{http_code}\n"
}

# ==========================================
# 3. Get Booking (Получение бронирования)
# ==========================================

# TC-3.1: Получение своего бронирования (владелец)
tc_3_1() {
    BOOKING_ID=${1:-1}
    print_test "TC-3.1: Получение своего бронирования (ID: $BOOKING_ID)"
    curl -s -X GET "${BASE_URL}/api/v1/bookings/${BOOKING_ID}" \
      -H "X-User-ID: 123456789" | jq .
}

# TC-3.2: Получение чужого бронирования (не владелец)
tc_3_2() {
    BOOKING_ID=${1:-1}
    print_test "TC-3.2: Попытка получить чужое бронирование (ожидается 403)"
    curl -s -X GET "${BASE_URL}/api/v1/bookings/${BOOKING_ID}" \
      -H "X-User-ID: 999999999" -w "\nHTTP Code: %{http_code}\n"
}

# TC-3.3: Получение бронирования менеджером компании
tc_3_3() {
    BOOKING_ID=${1:-1}
    print_test "TC-3.3: Получение бронирования менеджером компании"
    curl -s -X GET "${BASE_URL}/api/v1/bookings/${BOOKING_ID}" \
      -H "X-User-ID: 777777777" | jq .
}

# TC-3.4: Несуществующее бронирование
tc_3_4() {
    print_test "TC-3.4: Несуществующее бронирование (ожидается 404)"
    curl -s -X GET "${BASE_URL}/api/v1/bookings/999999" \
      -H "X-User-ID: 123456789" -w "\nHTTP Code: %{http_code}\n"
}

# TC-3.5: Без заголовка X-User-ID
tc_3_5() {
    print_test "TC-3.5: Запрос без X-User-ID (ожидается 401)"
    curl -s -X GET "${BASE_URL}/api/v1/bookings/1" -w "\nHTTP Code: %{http_code}\n"
}

# ==========================================
# 4. Cancel Booking (Отмена бронирования)
# ==========================================

# TC-4.1: Успешная отмена своего бронирования
tc_4_1() {
    BOOKING_ID=${1:-1}
    print_test "TC-4.1: Отмена своего бронирования (ID: $BOOKING_ID)"
    curl -s -X PATCH "${BASE_URL}/api/v1/bookings/${BOOKING_ID}/cancel" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 123456789" \
      -d "{
        \"userId\": 123456789,
        \"cancellationReason\": \"Изменились планы\"
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# TC-4.2: Отмена без указания причины
tc_4_2() {
    BOOKING_ID=${1:-2}
    print_test "TC-4.2: Отмена без причины (ID: $BOOKING_ID)"
    curl -s -X PATCH "${BASE_URL}/api/v1/bookings/${BOOKING_ID}/cancel" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 987654321" \
      -d "{
        \"userId\": 987654321
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# TC-4.3: Попытка отменить чужое бронирование
tc_4_3() {
    BOOKING_ID=${1:-1}
    print_test "TC-4.3: Попытка отменить чужое бронирование (ожидается 403)"
    curl -s -X PATCH "${BASE_URL}/api/v1/bookings/${BOOKING_ID}/cancel" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 999999999" \
      -d "{
        \"userId\": 999999999,
        \"cancellationReason\": \"Хочу отменить\"
      }" -w "\nHTTP Code: %{http_code}\n"
}

# TC-4.4: Отмена менеджером компании
tc_4_4() {
    BOOKING_ID=${1:-3}
    print_test "TC-4.4: Отмена бронирования менеджером компании"
    curl -s -X PATCH "${BASE_URL}/api/v1/bookings/${BOOKING_ID}/cancel" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 777777777" \
      -d "{
        \"userId\": 777777777,
        \"cancellationReason\": \"Технические работы на мойке\"
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# ==========================================
# 5. User Bookings (История бронирований пользователя)
# ==========================================

# TC-5.1: Получение всех бронирований пользователя
tc_5_1() {
    USER_ID=${1:-123456789}
    print_test "TC-5.1: Все бронирования пользователя $USER_ID"
    curl -s -X GET "${BASE_URL}/api/v1/users/${USER_ID}/bookings" \
      -H "X-User-ID: ${USER_ID}" | jq .
}

# TC-5.2: Фильтрация по статусу "confirmed"
tc_5_2() {
    USER_ID=${1:-123456789}
    print_test "TC-5.2: Только подтверждённые бронирования"
    curl -s -X GET "${BASE_URL}/api/v1/users/${USER_ID}/bookings?status=confirmed" \
      -H "X-User-ID: ${USER_ID}" | jq .
}

# TC-5.3: Фильтрация по статусу "completed"
tc_5_3() {
    USER_ID=${1:-123456789}
    print_test "TC-5.3: Только выполненные бронирования"
    curl -s -X GET "${BASE_URL}/api/v1/users/${USER_ID}/bookings?status=completed" \
      -H "X-User-ID: ${USER_ID}" | jq .
}

# TC-5.4: Попытка получить чужую историю
tc_5_4() {
    print_test "TC-5.4: Попытка получить чужую историю (ожидается 403)"
    curl -s -X GET "${BASE_URL}/api/v1/users/123456789/bookings" \
      -H "X-User-ID: 999999999" -w "\nHTTP Code: %{http_code}\n"
}

# ==========================================
# 6. Company Bookings (Бронирования компании)
# ==========================================

# TC-6.1: Получение всех бронирований компании (менеджер)
tc_6_1() {
    COMPANY_ID=${1:-1}
    print_test "TC-6.1: Все бронирования компании $COMPANY_ID"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/bookings" \
      -H "X-User-ID: 777777777" | jq .
}

# TC-6.2: Фильтрация по адресу
tc_6_2() {
    COMPANY_ID=${1:-1}
    ADDRESS_ID=${2:-100}
    print_test "TC-6.2: Бронирования на адресе $ADDRESS_ID"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/bookings?addressId=${ADDRESS_ID}" \
      -H "X-User-ID: 777777777" | jq .
}

# TC-6.3: Фильтрация по дате
tc_6_3() {
    COMPANY_ID=${1:-1}
    print_test "TC-6.3: Бронирования на завтра"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/bookings?date=${TOMORROW}" \
      -H "X-User-ID: 777777777" | jq .
}

# TC-6.4: Фильтрация по периоду
tc_6_4() {
    COMPANY_ID=${1:-1}
    START_DATE=$(date -v-7d +%Y-%m-%d 2>/dev/null || date -d "7 days ago" +%Y-%m-%d)
    END_DATE=$(date -v+7d +%Y-%m-%d 2>/dev/null || date -d "+7 days" +%Y-%m-%d)
    print_test "TC-6.4: Бронирования за период ${START_DATE} - ${END_DATE}"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/bookings?startDate=${START_DATE}&endDate=${END_DATE}" \
      -H "X-User-ID: 777777777" | jq .
}

# TC-6.5: Фильтрация по статусу
tc_6_5() {
    COMPANY_ID=${1:-1}
    print_test "TC-6.5: Только подтверждённые бронирования компании"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/bookings?status=confirmed" \
      -H "X-User-ID: 777777777" | jq .
}

# TC-6.6: Включение отменённых бронирований
tc_6_6() {
    COMPANY_ID=${1:-1}
    print_test "TC-6.6: Все бронирования, включая отменённые"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/bookings?includeInactive=true" \
      -H "X-User-ID: 777777777" | jq .
}

# TC-6.7: Попытка доступа не-менеджером
tc_6_7() {
    COMPANY_ID=${1:-1}
    print_test "TC-6.7: Попытка доступа не-менеджером (ожидается 403)"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/bookings" \
      -H "X-User-ID: 999999999" -w "\nHTTP Code: %{http_code}\n"
}

# ==========================================
# 7. Company Config (Получение конфигурации)
# ==========================================

# TC-7.1: Глобальная конфигурация компании
tc_7_1() {
    COMPANY_ID=${1:-1}
    print_test "TC-7.1: Глобальная конфигурация компании $COMPANY_ID"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/config" | jq .
}

# TC-7.2: Конфигурация для конкретного адреса
tc_7_2() {
    COMPANY_ID=${1:-1}
    ADDRESS_ID=${2:-100}
    print_test "TC-7.2: Конфигурация для адреса $ADDRESS_ID"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/config?addressId=${ADDRESS_ID}" | jq .
}

# TC-7.3: Конфигурация для услуги на адресе
tc_7_3() {
    COMPANY_ID=${1:-1}
    ADDRESS_ID=${2:-100}
    SERVICE_ID=${3:-2}
    print_test "TC-7.3: Конфигурация для услуги $SERVICE_ID на адресе $ADDRESS_ID"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/config?addressId=${ADDRESS_ID}&serviceId=${SERVICE_ID}" | jq .
}

# TC-7.4: Конфигурация для услуги (без адреса)
tc_7_4() {
    COMPANY_ID=${1:-1}
    SERVICE_ID=${2:-2}
    print_test "TC-7.4: Конфигурация для услуги $SERVICE_ID"
    curl -s -X GET "${BASE_URL}/api/v1/companies/${COMPANY_ID}/config?serviceId=${SERVICE_ID}" | jq .
}

# ==========================================
# 8. Update Config (Обновление конфигурации)
# ==========================================

# TC-8.1: Обновление глобальной конфигурации
tc_8_1() {
    COMPANY_ID=${1:-1}
    print_test "TC-8.1: Обновление глобальной конфигурации компании $COMPANY_ID"
    curl -s -X PUT "${BASE_URL}/api/v1/companies/${COMPANY_ID}/config" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 777777777" \
      -d "{
        \"userId\": 777777777,
        \"slotDurationMinutes\": 60,
        \"maxConcurrentBookings\": 5,
        \"advanceBookingDays\": 0,
        \"minBookingNoticeMinutes\": 120
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# TC-8.2: Создание конфигурации для адреса
tc_8_2() {
    COMPANY_ID=${1:-1}
    ADDRESS_ID=${2:-100}
    print_test "TC-8.2: Создание конфигурации для адреса $ADDRESS_ID"
    curl -s -X PUT "${BASE_URL}/api/v1/companies/${COMPANY_ID}/config" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 777777777" \
      -d "{
        \"userId\": 777777777,
        \"addressId\": ${ADDRESS_ID},
        \"slotDurationMinutes\": 30,
        \"maxConcurrentBookings\": 4,
        \"advanceBookingDays\": 30,
        \"minBookingNoticeMinutes\": 60
      }" -w "\nHTTP Code: %{http_code}\n" | jq .
}

# TC-8.4: Попытка обновления не-менеджером
tc_8_4() {
    COMPANY_ID=${1:-1}
    print_test "TC-8.4: Попытка обновления не-менеджером (ожидается 403)"
    curl -s -X PUT "${BASE_URL}/api/v1/companies/${COMPANY_ID}/config" \
      -H "Content-Type: application/json" \
      -H "X-User-ID: 999999999" \
      -d "{
        \"userId\": 999999999,
        \"slotDurationMinutes\": 30,
        \"maxConcurrentBookings\": 10
      }" -w "\nHTTP Code: %{http_code}\n"
}

# ==========================================
# Prometheus Metrics
# ==========================================

tc_metrics() {
    print_test "Проверка Prometheus метрик"
    curl -s -X GET "${BASE_URL}/metrics" | grep -E "^(http_|db_)" | head -20
}

# ==========================================
# Главное меню
# ==========================================

show_menu() {
    echo ""
    print_header "BookingService API Test Suite"
    echo "Выберите группу тестов:"
    echo "  1) Available Slots (TC-1.*)"
    echo "  2) Create Booking (TC-2.*)"
    echo "  3) Get Booking (TC-3.*)"
    echo "  4) Cancel Booking (TC-4.*)"
    echo "  5) User Bookings (TC-5.*)"
    echo "  6) Company Bookings (TC-6.*)"
    echo "  7) Company Config (TC-7.*)"
    echo "  8) Update Config (TC-8.*)"
    echo "  9) Prometheus Metrics"
    echo "  s) Smoke Tests (основные сценарии)"
    echo "  a) All Tests (все тесты)"
    echo "  q) Quit"
    echo ""
}

# Smoke тесты
run_smoke_tests() {
    print_header "SMOKE TESTS"
    tc_1_1
    sleep 1
    tc_2_1
    sleep 1
    tc_3_1
    sleep 1
    tc_metrics
    print_success "Smoke tests completed"
}

# Запуск всех тестов группы 1
run_group_1() {
    print_header "GROUP 1: Available Slots Tests"
    tc_1_1; sleep 0.5
    tc_1_2; sleep 0.5
    tc_1_3; sleep 0.5
    tc_1_4; sleep 0.5
    tc_1_6; sleep 0.5
    tc_1_7; sleep 0.5
    tc_1_8; sleep 0.5
    tc_1_9; sleep 0.5
    tc_1_10
    print_success "Group 1 completed"
}

# Запуск всех тестов группы 2
run_group_2() {
    print_header "GROUP 2: Create Booking Tests"
    tc_2_1; sleep 0.5
    tc_2_2; sleep 0.5
    tc_2_3; sleep 0.5
    tc_2_4; sleep 0.5
    tc_2_7; sleep 0.5
    tc_2_8; sleep 0.5
    tc_2_10; sleep 0.5
    tc_2_14; sleep 0.5
    tc_2_15
    print_success "Group 2 completed"
}

# Интерактивный режим
interactive_mode() {
    while true; do
        show_menu
        read -p "Ваш выбор: " choice
        case $choice in
            1) run_group_1 ;;
            2) run_group_2 ;;
            3) print_header "GROUP 3: Get Booking Tests"
               tc_3_1; sleep 0.5
               tc_3_2; sleep 0.5
               tc_3_3; sleep 0.5
               tc_3_4; sleep 0.5
               tc_3_5 ;;
            4) print_header "GROUP 4: Cancel Booking Tests"
               tc_4_1; sleep 0.5
               tc_4_2; sleep 0.5
               tc_4_3; sleep 0.5
               tc_4_4 ;;
            5) print_header "GROUP 5: User Bookings Tests"
               tc_5_1; sleep 0.5
               tc_5_2; sleep 0.5
               tc_5_3; sleep 0.5
               tc_5_4 ;;
            6) print_header "GROUP 6: Company Bookings Tests"
               tc_6_1; sleep 0.5
               tc_6_2; sleep 0.5
               tc_6_3; sleep 0.5
               tc_6_4; sleep 0.5
               tc_6_5; sleep 0.5
               tc_6_6; sleep 0.5
               tc_6_7 ;;
            7) print_header "GROUP 7: Company Config Tests"
               tc_7_1; sleep 0.5
               tc_7_2; sleep 0.5
               tc_7_3; sleep 0.5
               tc_7_4 ;;
            8) print_header "GROUP 8: Update Config Tests"
               tc_8_1; sleep 0.5
               tc_8_2; sleep 0.5
               tc_8_4 ;;
            9) tc_metrics ;;
            s|S) run_smoke_tests ;;
            a|A) echo "Запуск всех тестов..."
                 run_smoke_tests; sleep 2
                 run_group_1; sleep 2
                 run_group_2; sleep 2
                 print_success "All tests completed" ;;
            q|Q) echo "Выход..."; exit 0 ;;
            *) print_error "Неверный выбор" ;;
        esac
    done
}

# Если скрипт запущен напрямую (не через source)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Проверка зависимостей
    if ! command -v jq > /dev/null 2>&1; then
        print_error "jq не установлен. Установите: brew install jq"
        exit 1
    fi

    # Проверка доступности сервиса
    if ! curl -s -f "${BASE_URL}/metrics" > /dev/null 2>&1; then
        print_error "BookingService недоступен на ${BASE_URL}"
        print_error "Запустите сервис: make docker-up"
        exit 1
    fi

    # Запуск интерактивного режима
    interactive_mode
fi

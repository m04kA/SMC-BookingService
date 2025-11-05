package create_booking

import (
	"fmt"
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	"github.com/m04kA/SMC-BookingService/pkg/types"
)

// validateRequest валидирует входные данные запроса
func validateRequest(req *Request) error {
	if req.UserID <= 0 {
		return fmt.Errorf("%w: userID must be positive", ErrInvalidInput)
	}

	if req.CompanyID <= 0 {
		return fmt.Errorf("%w: companyID must be positive", ErrInvalidInput)
	}

	if req.AddressID <= 0 {
		return fmt.Errorf("%w: addressID must be positive", ErrInvalidInput)
	}

	if req.ServiceID <= 0 {
		return fmt.Errorf("%w: serviceID must be positive", ErrInvalidInput)
	}

	// Проверяем, что дата не является нулевой
	if req.Date.IsZero() {
		return fmt.Errorf("%w: date is required", ErrInvalidInput)
	}

	// Проверяем, что время начала указано
	if req.StartTime.IsZero() {
		return fmt.Errorf("%w: startTime is required", ErrInvalidInput)
	}

	// Валидируем формат времени
	if err := req.StartTime.Validate(); err != nil {
		return fmt.Errorf("%w: invalid startTime format: %v", ErrInvalidInput, err)
	}

	return nil
}

// validateDate проверяет, что дата подходит для бронирования
func validateDate(bookingDate time.Time, now time.Time, advanceBookingDays int) error {
	// Проверяем, что дата не в прошлом
	if isDateInPast(bookingDate, now) {
		return ErrInvalidDate
	}

	// Если advanceBookingDays = 0, нет ограничений на дату
	if advanceBookingDays == 0 {
		return nil
	}

	// Проверяем, что дата не превышает ограничение advanceBookingDays
	maxDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, advanceBookingDays)

	bookingDateOnly := time.Date(bookingDate.Year(), bookingDate.Month(), bookingDate.Day(), 0, 0, 0, 0, bookingDate.Location())

	if bookingDateOnly.After(maxDate) {
		return fmt.Errorf("%w: can only book %d days in advance", ErrDateTooFarInFuture, advanceBookingDays)
	}

	return nil
}

// validateBookingTime проверяет, что бронирование не нарушает minBookingNoticeMinutes
func validateBookingTime(
	bookingDate time.Time,
	startTime types.TimeString,
	now time.Time,
	minBookingNoticeMinutes int,
) error {
	// Если дата бронирования не сегодня, проверка не нужна
	if !isSameDay(bookingDate, now) {
		return nil
	}

	// Вычисляем минимальное допустимое время
	currentTime := types.NewTimeString(now)
	minAllowedTime, err := currentTime.AddMinutes(minBookingNoticeMinutes)
	if err != nil {
		return fmt.Errorf("%w: failed to calculate min allowed time: %v", ErrInternal, err)
	}

	// Проверяем, что время начала не раньше минимального
	if startTime.IsBefore(minAllowedTime) {
		return fmt.Errorf("%w: must book at least %d minutes in advance", ErrTooLateToBook, minBookingNoticeMinutes)
	}

	return nil
}

// validateAddressExists проверяет, что адрес существует в компании
func validateAddressExists(company *sellerservice.Company, addressID int64) error {
	for _, addr := range company.Addresses {
		if addr.ID == addressID {
			return nil
		}
	}
	return ErrAddressNotFound
}

// validateServiceAtAddress проверяет, что услуга доступна на указанном адресе
func validateServiceAtAddress(service *sellerservice.Service, addressID int64) error {
	for _, addrID := range service.AddressIDs {
		if addrID == addressID {
			return nil
		}
	}
	return ErrServiceNotAvailableAtAddress
}

// countOverlappingBookings подсчитывает количество активных бронирований на указанный слот
func countOverlappingBookings(
	startTime types.TimeString,
	slotDuration int,
	bookings []*domain.Booking,
) (int, error) {
	slotEnd, err := startTime.AddMinutes(slotDuration)
	if err != nil {
		return 0, err
	}

	count := 0

	for _, booking := range bookings {
		// Пропускаем неактивные бронирования
		if !booking.IsActive() {
			continue
		}

		bookingStart := booking.StartTime
		bookingEnd, err := booking.StartTime.AddMinutes(booking.DurationMinutes)
		if err != nil {
			// Если не можем вычислить конец бронирования, пропускаем
			continue
		}

		// Проверяем пересечение (строгие неравенства, граничные случаи не считаются)
		if bookingStart.IsBefore(slotEnd) && bookingEnd.IsAfter(startTime) {
			count++
		}
	}

	return count, nil
}

// getWorkingHoursForDay возвращает расписание работы компании на указанный день недели
func getWorkingHoursForDay(company *sellerservice.Company, date time.Time) sellerservice.DaySchedule {
	weekday := date.Weekday()

	switch weekday {
	case time.Monday:
		return company.WorkingHours.Monday
	case time.Tuesday:
		return company.WorkingHours.Tuesday
	case time.Wednesday:
		return company.WorkingHours.Wednesday
	case time.Thursday:
		return company.WorkingHours.Thursday
	case time.Friday:
		return company.WorkingHours.Friday
	case time.Saturday:
		return company.WorkingHours.Saturday
	case time.Sunday:
		return company.WorkingHours.Sunday
	default:
		return sellerservice.DaySchedule{IsOpen: false}
	}
}

// isSameDay проверяет, что две даты относятся к одному и тому же дню
func isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// isDateInPast проверяет, что дата в прошлом (раньше сегодняшнего дня)
func isDateInPast(date, now time.Time) bool {
	// Обнуляем время, чтобы сравнивать только даты
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	nowOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return dateOnly.Before(nowOnly)
}

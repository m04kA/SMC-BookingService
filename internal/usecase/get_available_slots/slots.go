package get_available_slots

import (
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	"github.com/m04kA/SMC-BookingService/pkg/types"
)

// generateTimeSlots генерирует список всех возможных временных слотов на день
// Слоты генерируются с начала работы компании с фиксированным шагом slotDuration
// Затем фильтруются с учетом текущего времени и минимального времени до бронирования
func generateTimeSlots(
	workingHours sellerservice.DaySchedule,
	slotDuration int,
	requestDate time.Time,
	now time.Time,
	minBookingNoticeMinutes int,
) ([]types.TimeString, error) {
	// Проверяем, что дата не в прошлом
	if isDateInPast(requestDate, now) {
		return []types.TimeString{}, nil
	}

	// Если компания закрыта в этот день
	if !workingHours.IsOpen || workingHours.OpenTime == nil || workingHours.CloseTime == nil {
		return []types.TimeString{}, nil
	}

	// Парсим время открытия и закрытия
	openTime, err := types.NewTimeStringFromString(*workingHours.OpenTime)
	if err != nil {
		return nil, err
	}

	closeTime, err := types.NewTimeStringFromString(*workingHours.CloseTime)
	if err != nil {
		return nil, err
	}

	// Шаг 1: Генерируем ВСЕ слоты от начала работы до конца с фиксированным шагом
	allSlots := make([]types.TimeString, 0)
	currentSlot := openTime

	for currentSlot.IsBefore(closeTime) {
		// Проверяем, что слот не выходит за время закрытия
		slotEnd, err := currentSlot.AddMinutes(slotDuration)
		if err != nil {
			return nil, err
		}
		if slotEnd.IsAfter(closeTime) {
			break
		}

		allSlots = append(allSlots, currentSlot)
		currentSlot, err = currentSlot.AddMinutes(slotDuration)
		if err != nil {
			return nil, err
		}
	}

	// Шаг 2: Если дата бронирования НЕ сегодня - возвращаем все слоты
	if !isSameDay(requestDate, now) {
		return allSlots, nil
	}

	// Шаг 3: Если дата бронирования - сегодня, фильтруем слоты по времени
	// Вычисляем минимальное допустимое время начала слота
	currentTime := types.NewTimeString(now)
	minAllowedTime, err := currentTime.AddMinutes(minBookingNoticeMinutes)
	if err != nil {
		return nil, err
	}

	// Фильтруем слоты - оставляем только те, которые начинаются не раньше minAllowedTime
	availableSlots := make([]types.TimeString, 0)
	for _, slot := range allSlots {
		if !slot.IsBefore(minAllowedTime) {
			availableSlots = append(availableSlots, slot)
		}
	}

	return availableSlots, nil
}

// calculateAvailableSpots вычисляет количество свободных мест для каждого слота
func calculateAvailableSpots(
	slots []types.TimeString,
	slotDuration int,
	bookings []*domain.Booking,
	maxConcurrentBookings int,
) []Slot {
	result := make([]Slot, len(slots))

	for i, slotStart := range slots {
		// Подсчитываем количество бронирований, пересекающихся с этим слотом
		overlappingCount := countOverlappingBookings(slotStart, slotDuration, bookings)

		availableSpots := maxConcurrentBookings - overlappingCount
		if availableSpots < 0 {
			availableSpots = 0
		}

		result[i] = Slot{
			StartTime:       slotStart,
			DurationMinutes: slotDuration,
			AvailableSpots:  availableSpots,
			TotalSpots:      maxConcurrentBookings,
		}
	}

	return result
}

// countOverlappingBookings подсчитывает количество бронирований, пересекающихся с указанным слотом
// Пересечение есть только если интервалы действительно накладываются друг на друга
// Если одно бронирование заканчивается ровно там, где начинается слот (или наоборот) - это НЕ пересечение
//
// Примеры:
// - Слот 11:30-12:00, бронирование 11:20-11:40 → ЕСТЬ пересечение (11:30-11:40)
// - Слот 11:30-12:00, бронирование 11:00-11:30 → НЕТ пересечения (граничат)
// - Слот 11:30-12:00, бронирование 12:00-12:30 → НЕТ пересечения (граничат)
func countOverlappingBookings(slotStart types.TimeString, slotDuration int, bookings []*domain.Booking) int {
	slotEnd, err := slotStart.AddMinutes(slotDuration)
	if err != nil {
		// Если не можем вычислить конец слота, считаем что пересечений нет
		return 0
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

		// Проверяем РЕАЛЬНОЕ пересечение временных интервалов
		// Интервалы пересекаются, только если:
		// - начало бронирования СТРОГО раньше конца слота И
		// - конец бронирования СТРОГО позже начала слота
		//
		// Используем строгие неравенства (IsBefore, IsAfter), чтобы граничные случаи не считались пересечением
		if bookingStart.IsBefore(slotEnd) && bookingEnd.IsAfter(slotStart) {
			count++
		}
	}

	return count
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

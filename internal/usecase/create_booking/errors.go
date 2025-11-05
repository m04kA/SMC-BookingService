package create_booking

import "errors"

var (
	// ErrCompanyNotFound возвращается, когда компания не найдена
	ErrCompanyNotFound = errors.New("create_booking: company not found")

	// ErrAddressNotFound возвращается, когда адрес не найден в компании
	ErrAddressNotFound = errors.New("create_booking: address not found")

	// ErrServiceNotFound возвращается, когда услуга не найдена
	ErrServiceNotFound = errors.New("create_booking: service not found")

	// ErrServiceNotAvailableAtAddress возвращается, когда услуга недоступна на указанном адресе
	ErrServiceNotAvailableAtAddress = errors.New("create_booking: service is not available at this address")

	// ErrCarNotFound возвращается, когда у пользователя нет выбранного автомобиля
	ErrCarNotFound = errors.New("create_booking: user has no selected car")

	// ErrInvalidDate возвращается при некорректной дате бронирования
	ErrInvalidDate = errors.New("create_booking: invalid booking date")

	// ErrDateTooFarInFuture возвращается, когда дата превышает ограничение advanceBookingDays
	ErrDateTooFarInFuture = errors.New("create_booking: date is too far in the future")

	// ErrCompanyClosed возвращается, когда компания закрыта в указанную дату
	ErrCompanyClosed = errors.New("create_booking: company is closed on this date")

	// ErrSlotNotAvailable возвращается, когда выбранный слот недоступен (все места заняты)
	ErrSlotNotAvailable = errors.New("create_booking: slot is not available")

	// ErrInvalidTimeSlot возвращается, когда время слота некорректно (не кратно slotDuration или вне рабочих часов)
	ErrInvalidTimeSlot = errors.New("create_booking: invalid time slot")

	// ErrTooLateToBook возвращается, когда попытка забронировать слот нарушает minBookingNoticeMinutes
	ErrTooLateToBook = errors.New("create_booking: too late to book this slot")

	// ErrInvalidInput возвращается при некорректных входных данных
	ErrInvalidInput = errors.New("create_booking: invalid input data")

	// ErrInternal возвращается при внутренних ошибках usecase
	ErrInternal = errors.New("create_booking: internal error")
)

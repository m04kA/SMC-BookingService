package domain

// Default configuration values
const (
	DefaultSlotDurationMinutes     = 30
	DefaultMaxConcurrentBookings   = 1
	DefaultAdvanceBookingDays      = 0  // 0 = unlimited
	DefaultMinBookingNoticeMinutes = 60 // 1 hour
)

// Business validation constants
const (
	MinSlotDurationMinutes     = 5
	MaxSlotDurationMinutes     = 480 // 8 hours
	MinConcurrentBookings      = 1
	MaxConcurrentBookings      = 100
	MinAdvanceBookingDays      = 0
	MaxAdvanceBookingDays      = 365 // 1 year
	MinBookingNoticeMinutes    = 0
	MaxBookingNoticeMinutes    = 10080 // 1 week
	MaxNotesLength             = 500
	MaxCancellationReasonLength = 500
)

// Time format constants
const (
	TimeFormat = "15:04"      // HH:MM
	DateFormat = "2006-01-02" // YYYY-MM-DD
)

// InactiveStatuses список статусов неактивных бронирований
// Используется для фильтрации при подсчёте доступных слотов
var InactiveStatuses = []BookingStatus{
	StatusCancelledByUser,
	StatusCancelledByCompany,
	StatusNoShow,
}

// ActiveStatuses список статусов активных бронирований
// Используется для фильтрации активных бронирований
var ActiveStatuses = []BookingStatus{
	StatusPending,
	StatusConfirmed,
	StatusInProgress,
	StatusCompleted,
}

package sellerservice

import "time"

// Company модель компании из SellerService
type Company struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Addresses    []Address    `json:"addresses"`
	WorkingHours WorkingHours `json:"working_hours"`
	ManagerIDs   []int64      `json:"manager_ids"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Address адрес компании (без координат)
type Address struct {
	ID       int64  `json:"id"`
	City     string `json:"city"`
	Street   string `json:"street"`
	Building string `json:"building"`
}

// WorkingHours рабочие часы компании
type WorkingHours struct {
	Monday    DaySchedule `json:"monday"`
	Tuesday   DaySchedule `json:"tuesday"`
	Wednesday DaySchedule `json:"wednesday"`
	Thursday  DaySchedule `json:"thursday"`
	Friday    DaySchedule `json:"friday"`
	Saturday  DaySchedule `json:"saturday"`
	Sunday    DaySchedule `json:"sunday"`
}

// DaySchedule расписание на день
type DaySchedule struct {
	IsOpen    bool    `json:"isOpen"`
	OpenTime  *string `json:"openTime,omitempty"`
	CloseTime *string `json:"closeTime,omitempty"`
}

// Service модель услуги из SellerService
type Service struct {
	ID              int64     `json:"id"`
	CompanyID       int64     `json:"company_id"`
	Name            string    `json:"name"`
	AverageDuration *int      `json:"average_duration,omitempty"`
	AddressIDs      []int64   `json:"address_ids"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Опциональные поля цены (заполняются при наличии X-User-ID)
	Price             *float64 `json:"price,omitempty"`
	Currency          *string  `json:"currency,omitempty"`
	PricingType       *string  `json:"pricing_type,omitempty"`
	VehicleClass      *string  `json:"vehicle_class,omitempty"`
	AppliedMultiplier *float64 `json:"applied_multiplier,omitempty"`
}

// ErrorResponse модель ошибки от SellerService
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

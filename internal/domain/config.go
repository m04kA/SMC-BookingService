package domain

import "time"

// CompanySlotsConfig represents the booking configuration for a company
// Supports hierarchical configuration:
// 1. Service at specific address (company_id, address_id, service_id)
// 2. Address-wide (company_id, address_id, NULL)
// 3. Company-wide (company_id, NULL, NULL)
type CompanySlotsConfig struct {
	ID                      int64
	CompanyID               int64
	AddressID               *int64 // NULL = config for all addresses
	ServiceID               *int64 // NULL = config for all services
	SlotDurationMinutes     int
	MaxConcurrentBookings   int
	AdvanceBookingDays      int // 0 = unlimited
	MinBookingNoticeMinutes int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// IsGlobalConfig returns true if this is a global company configuration (not address or service-specific)
func (c *CompanySlotsConfig) IsGlobalConfig() bool {
	return c.AddressID == nil && c.ServiceID == nil
}

// IsAddressSpecific returns true if this configuration is for a specific address
func (c *CompanySlotsConfig) IsAddressSpecific() bool {
	return c.AddressID != nil && c.ServiceID == nil
}

// IsServiceSpecific returns true if this configuration is for a specific service (address-wide)
func (c *CompanySlotsConfig) IsServiceSpecific() bool {
	return c.AddressID == nil && c.ServiceID != nil
}

// IsServiceAtAddress returns true if this configuration is for a specific service at a specific address
func (c *CompanySlotsConfig) IsServiceAtAddress() bool {
	return c.AddressID != nil && c.ServiceID != nil
}

// HasAdvanceBookingLimit returns true if there's a limit on how far in advance bookings can be made
func (c *CompanySlotsConfig) HasAdvanceBookingLimit() bool {
	return c.AdvanceBookingDays > 0
}

// SupportsParallelBookings returns true if multiple concurrent bookings are supported
func (c *CompanySlotsConfig) SupportsParallelBookings() bool {
	return c.MaxConcurrentBookings > 1
}

package domain

import "github.com/m04kA/SMC-BookingService/pkg/types"

// AvailableSlot represents a time slot available for booking
type AvailableSlot struct {
	StartTime       types.TimeString
	DurationMinutes int
	AvailableSpots  int // Available spots (boxes)
	TotalSpots      int // Total spots (boxes)
}

// IsFull returns true if the slot has no available spots
func (s *AvailableSlot) IsFull() bool {
	return s.AvailableSpots <= 0
}

// IsPartiallyAvailable returns true if the slot has some but not all spots available
func (s *AvailableSlot) IsPartiallyAvailable() bool {
	return s.AvailableSpots > 0 && s.AvailableSpots < s.TotalSpots
}

// IsFullyAvailable returns true if all spots are available
func (s *AvailableSlot) IsFullyAvailable() bool {
	return s.AvailableSpots == s.TotalSpots
}

// OccupancyRate returns the occupancy rate as a percentage (0-100)
func (s *AvailableSlot) OccupancyRate() float64 {
	if s.TotalSpots == 0 {
		return 0
	}
	occupied := s.TotalSpots - s.AvailableSpots
	return float64(occupied) / float64(s.TotalSpots) * 100
}

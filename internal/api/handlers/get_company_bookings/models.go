package get_company_bookings

import (
	"fmt"
	"strconv"
	"time"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

// ToServiceRequest формирует запрос к сервису из query параметров
func ToServiceRequest(
	companyID int64,
	userID int64,
	addressIDStr string,
	statusStr string,
	dateStr string,
	includeInactiveStr string,
) (*models.GetCompanyBookingsRequest, error) {
	req := &models.GetCompanyBookingsRequest{
		UserID:          userID,
		CompanyID:       companyID,
		IncludeInactive: false, // По умолчанию только активные
	}

	// Парсим addressId если указан
	if addressIDStr != "" {
		addressID, err := strconv.ParseInt(addressIDStr, 10, 64)
		if err != nil {
			return nil, err
		}
		req.AddressID = &addressID
	}

	// Парсим status если указан
	if statusStr != "" {
		req.Status = &statusStr
	}

	// Парсим date если указана
	if dateStr != "" {
		date, err := time.Parse(domain.DateFormat, dateStr)
		if err != nil {
			return nil, err
		}
		req.StartDate = &date
		req.EndDate = &date
	}

	// Парсим includeInactive если указан
	if includeInactiveStr != "" {
		includeInactive, err := strconv.ParseBool(includeInactiveStr)
		if err != nil {
			return nil, fmt.Errorf("invalid includeInactive value: %w", err)
		}
		req.IncludeInactive = includeInactive
	}

	return req, nil
}

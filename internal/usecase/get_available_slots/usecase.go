package get_available_slots

import (
	"context"
	"errors"
	"fmt"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	configRepo "github.com/m04kA/SMC-BookingService/internal/infra/storage/config"
	sellerClient "github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	"github.com/m04kA/SMC-BookingService/pkg/ptr"
)

// UseCase use case для получения доступных слотов для бронирования
type UseCase struct {
	bookingRepo  BookingRepository
	configRepo   ConfigRepository
	sellerClient SellerServiceClient
	timeProvider TimeProvider
	logger       Logger
}

// NewUseCase создает новый экземпляр use case
func NewUseCase(
	bookingRepo BookingRepository,
	configRepo ConfigRepository,
	sellerClient SellerServiceClient,
	logger Logger,
) *UseCase {
	return &UseCase{
		bookingRepo:  bookingRepo,
		configRepo:   configRepo,
		sellerClient: sellerClient,
		timeProvider: &RealTimeProvider{},
		logger:       logger,
	}
}

// Execute выполняет use case получения доступных слотов
func (uc *UseCase) Execute(ctx context.Context, req *Request) (*Response, error) {
	uc.logger.Info("GetAvailableSlots: user=%d, company=%d, address=%d, service=%d, date=%s",
		req.UserID, req.CompanyID, req.AddressID, req.ServiceID, req.Date.Format(domain.DateFormat))

	// 1. Валидация входных данных
	if err := validateRequest(req); err != nil {
		uc.logger.Warn("GetAvailableSlots: validation failed: %v", err)
		return nil, err
	}

	// 2. Получаем текущее время
	now := uc.timeProvider.Now()

	// 3. Получаем компанию
	company, err := uc.sellerClient.GetCompany(ctx, req.CompanyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			uc.logger.Warn("GetAvailableSlots: company id=%d not found", req.CompanyID)
			return nil, ErrCompanyNotFound
		}
		uc.logger.Error("GetAvailableSlots: failed to get company id=%d: %v", req.CompanyID, err)
		return nil, fmt.Errorf("%w: failed to get company: %v", ErrInternal, err)
	}

	// 4. Проверяем существование адреса
	if err := validateAddressExists(company, req.AddressID); err != nil {
		uc.logger.Warn("GetAvailableSlots: address id=%d not found in company id=%d", req.AddressID, req.CompanyID)
		return nil, err
	}

	// 5. Получаем услугу
	service, err := uc.sellerClient.GetService(ctx, req.CompanyID, req.ServiceID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrServiceNotFound) {
			uc.logger.Warn("GetAvailableSlots: service id=%d not found", req.ServiceID)
			return nil, ErrServiceNotFound
		}
		uc.logger.Error("GetAvailableSlots: failed to get service id=%d: %v", req.ServiceID, err)
		return nil, fmt.Errorf("%w: failed to get service: %v", ErrInternal, err)
	}

	// 6. Проверяем, что услуга доступна на этом адресе
	if err := validateServiceAtAddress(service, req.AddressID); err != nil {
		uc.logger.Warn("GetAvailableSlots: service id=%d not available at address id=%d",
			req.ServiceID, req.AddressID)
		return nil, err
	}

	// 7. Получаем конфигурацию слотов с учетом иерархии
	config, err := uc.configRepo.GetConfigWithHierarchy(ctx, req.CompanyID, ptr.Ptr(req.AddressID), ptr.Ptr(req.ServiceID))
	if err != nil && !errors.Is(err, configRepo.ErrConfigNotFound) {
		uc.logger.Error("GetAvailableSlots: failed to get config: %v", err)
		return nil, fmt.Errorf("%w: failed to get config: %v", ErrInternal, err)
	}

	// Если конфигурация не найдена, используем дефолтные значения
	if config == nil {
		config = &domain.CompanySlotsConfig{
			SlotDurationMinutes:     domain.DefaultSlotDurationMinutes,
			MaxConcurrentBookings:   domain.DefaultMaxConcurrentBookings,
			AdvanceBookingDays:      domain.DefaultAdvanceBookingDays,
			MinBookingNoticeMinutes: domain.DefaultMinBookingNoticeMinutes,
		}
		uc.logger.Info("GetAvailableSlots: using default config for company=%d, address=%d, service=%d",
			req.CompanyID, req.AddressID, req.ServiceID)
	} else {
		uc.logger.Info("GetAvailableSlots: using config id=%d", config.ID)
	}

	// 8. Валидация даты с учетом конфигурации
	if err := validateDate(req.Date, now, config.AdvanceBookingDays); err != nil {
		uc.logger.Warn("GetAvailableSlots: date validation failed: %v", err)
		return nil, err
	}

	// 9. Получаем рабочие часы на указанную дату
	workingHours := getWorkingHoursForDay(company, req.Date)
	if !workingHours.IsOpen {
		uc.logger.Info("GetAvailableSlots: company is closed on %s", req.Date.Format(domain.DateFormat))
		return &Response{
			Date:      req.Date,
			CompanyID: req.CompanyID,
			AddressID: req.AddressID,
			ServiceID: req.ServiceID,
			Slots:     []Slot{},
		}, nil
	}

	// 10. Генерируем временные слоты
	timeSlots, err := generateTimeSlots(
		workingHours,
		config.SlotDurationMinutes,
		req.Date,
		now,
		config.MinBookingNoticeMinutes,
	)
	if err != nil {
		uc.logger.Error("GetAvailableSlots: failed to generate time slots: %v", err)
		return nil, fmt.Errorf("%w: failed to generate time slots: %v", ErrInternal, err)
	}

	// 11. Получаем все бронирования на эту дату и адрес
	filter := domain.CompanyBookingsFilter{
		CompanyID:       req.CompanyID,
		AddressID:       &req.AddressID,
		StartDate:       &req.Date,
		EndDate:         &req.Date,
		IncludeInactive: false, // Только активные бронирования
	}

	bookings, err := uc.bookingRepo.GetByCompanyWithFilter(ctx, filter)
	if err != nil {
		uc.logger.Error("GetAvailableSlots: failed to get bookings: %v", err)
		return nil, fmt.Errorf("%w: failed to get bookings: %v", ErrInternal, err)
	}

	// 12. Вычисляем доступность для каждого слота
	slots := calculateAvailableSpots(
		timeSlots,
		config.SlotDurationMinutes,
		bookings,
		config.MaxConcurrentBookings,
	)

	uc.logger.Info("GetAvailableSlots: generated %d slots for company=%d, address=%d, service=%d, date=%s",
		len(slots), req.CompanyID, req.AddressID, req.ServiceID, req.Date.Format(domain.DateFormat))

	return &Response{
		Date:      req.Date,
		CompanyID: req.CompanyID,
		AddressID: req.AddressID,
		ServiceID: req.ServiceID,
		Slots:     slots,
	}, nil
}

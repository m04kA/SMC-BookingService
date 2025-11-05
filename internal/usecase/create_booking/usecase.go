package create_booking

import (
	"context"
	"errors"
	"fmt"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	configRepo "github.com/m04kA/SMC-BookingService/internal/infra/storage/config"
	sellerClient "github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	userClient "github.com/m04kA/SMC-BookingService/internal/integrations/userservice"
	"github.com/m04kA/SMC-BookingService/pkg/ptr"
)

// UseCase use case для создания бронирования
type UseCase struct {
	bookingRepo  BookingRepository
	configRepo   ConfigRepository
	sellerClient SellerServiceClient
	userClient   UserServiceClient
	txManager    TransactionManager
	timeProvider TimeProvider
	logger       Logger
}

// NewUseCase создает новый экземпляр use case
func NewUseCase(
	bookingRepo BookingRepository,
	configRepo ConfigRepository,
	sellerClient SellerServiceClient,
	userClient UserServiceClient,
	txManager TransactionManager,
	logger Logger,
) *UseCase {
	return &UseCase{
		bookingRepo:  bookingRepo,
		configRepo:   configRepo,
		sellerClient: sellerClient,
		userClient:   userClient,
		txManager:    txManager,
		timeProvider: &RealTimeProvider{},
		logger:       logger,
	}
}

// Execute выполняет use case создания бронирования
// Использует сериализуемую транзакцию для предотвращения гонки данных
func (uc *UseCase) Execute(ctx context.Context, req *Request) (*Response, error) {
	uc.logger.Info("CreateBooking: user=%d, company=%d, address=%d, service=%d, date=%s, time=%s",
		req.UserID, req.CompanyID, req.AddressID, req.ServiceID, req.Date.Format(domain.DateFormat), req.StartTime)

	// 1. Валидация входных данных
	if err := validateRequest(req); err != nil {
		uc.logger.Warn("CreateBooking: validation failed: %v", err)
		return nil, err
	}

	// 2. Получаем текущее время
	now := uc.timeProvider.Now()

	// 3. Получаем компанию
	company, err := uc.sellerClient.GetCompany(ctx, req.CompanyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			uc.logger.Warn("CreateBooking: company id=%d not found", req.CompanyID)
			return nil, ErrCompanyNotFound
		}
		uc.logger.Error("CreateBooking: failed to get company id=%d: %v", req.CompanyID, err)
		return nil, fmt.Errorf("%w: failed to get company: %v", ErrInternal, err)
	}

	// 4. Проверяем существование адреса
	if err := validateAddressExists(company, req.AddressID); err != nil {
		uc.logger.Warn("CreateBooking: address id=%d not found in company id=%d", req.AddressID, req.CompanyID)
		return nil, err
	}

	// 5. Получаем услугу
	service, err := uc.sellerClient.GetService(ctx, req.CompanyID, req.ServiceID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrServiceNotFound) {
			uc.logger.Warn("CreateBooking: service id=%d not found", req.ServiceID)
			return nil, ErrServiceNotFound
		}
		uc.logger.Error("CreateBooking: failed to get service id=%d: %v", req.ServiceID, err)
		return nil, fmt.Errorf("%w: failed to get service: %v", ErrInternal, err)
	}

	// 6. Проверяем, что услуга доступна на этом адресе
	if err := validateServiceAtAddress(service, req.AddressID); err != nil {
		uc.logger.Warn("CreateBooking: service id=%d not available at address id=%d",
			req.ServiceID, req.AddressID)
		return nil, err
	}

	// 7. Получаем выбранный автомобиль пользователя
	car, err := uc.userClient.GetSelectedCar(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, userClient.ErrCarNotFound) {
			uc.logger.Warn("CreateBooking: user id=%d has no selected car", req.UserID)
			return nil, ErrCarNotFound
		}
		uc.logger.Error("CreateBooking: failed to get selected car for user id=%d: %v", req.UserID, err)
		return nil, fmt.Errorf("%w: failed to get selected car: %v", ErrInternal, err)
	}

	// Переменная для хранения результата
	var result *domain.Booking

	// 8. Выполняем операции с БД в сериализуемой транзакции
	err = uc.txManager.DoSerializable(ctx, func(txCtx context.Context) error {
		// 8.1. Получаем конфигурацию слотов с учетом иерархии
		config, err := uc.configRepo.GetConfigWithHierarchy(txCtx, req.CompanyID, ptr.Ptr(req.AddressID), ptr.Ptr(req.ServiceID))
		if err != nil && !errors.Is(err, configRepo.ErrConfigNotFound) {
			uc.logger.Error("CreateBooking: failed to get config: %v", err)
			return fmt.Errorf("%w: failed to get config: %v", ErrInternal, err)
		}

		// Если конфигурация не найдена, используем дефолтные значения
		if config == nil {
			config = &domain.CompanySlotsConfig{
				SlotDurationMinutes:     domain.DefaultSlotDurationMinutes,
				MaxConcurrentBookings:   domain.DefaultMaxConcurrentBookings,
				AdvanceBookingDays:      domain.DefaultAdvanceBookingDays,
				MinBookingNoticeMinutes: domain.DefaultMinBookingNoticeMinutes,
			}
			uc.logger.Info("CreateBooking: using default config for company=%d, address=%d, service=%d",
				req.CompanyID, req.AddressID, req.ServiceID)
		} else {
			uc.logger.Info("CreateBooking: using config id=%d", config.ID)
		}

		// 8.2. Валидация даты с учетом конфигурации
		if err := validateDate(req.Date, now, config.AdvanceBookingDays); err != nil {
			uc.logger.Warn("CreateBooking: date validation failed: %v", err)
			return err
		}

		// 8.3. Получаем рабочие часы на указанную дату
		workingHours := getWorkingHoursForDay(company, req.Date)
		if !workingHours.IsOpen {
			uc.logger.Warn("CreateBooking: company is closed on %s", req.Date.Format(domain.DateFormat))
			return ErrCompanyClosed
		}

		// 8.4. Валидация времени бронирования (minBookingNoticeMinutes)
		if err := validateBookingTime(req.Date, req.StartTime, now, config.MinBookingNoticeMinutes); err != nil {
			uc.logger.Warn("CreateBooking: booking time validation failed: %v", err)
			return err
		}

		// 8.5. Получаем все активные бронирования на эту дату и адрес с блокировкой (FOR UPDATE)
		filter := domain.CompanyBookingsFilter{
			CompanyID:       req.CompanyID,
			AddressID:       &req.AddressID,
			StartDate:       &req.Date,
			EndDate:         &req.Date,
			IncludeInactive: false, // Только активные бронирования
		}

		bookings, err := uc.bookingRepo.GetByCompanyWithFilter(txCtx, filter)
		if err != nil {
			uc.logger.Error("CreateBooking: failed to get bookings: %v", err)
			return fmt.Errorf("%w: failed to get bookings: %v", ErrInternal, err)
		}

		// 8.6. Проверяем доступность слота
		overlappingCount, err := countOverlappingBookings(req.StartTime, config.SlotDurationMinutes, bookings)
		if err != nil {
			uc.logger.Error("CreateBooking: failed to count overlapping bookings: %v", err)
			return fmt.Errorf("%w: failed to count overlapping bookings: %v", ErrInternal, err)
		}

		// Если MaxConcurrentBookings = 4, то допустимо overlappingCount = 0, 1, 2, 3
		// При overlappingCount >= 4 слот недоступен
		if overlappingCount >= config.MaxConcurrentBookings {
			uc.logger.Warn("CreateBooking: slot not available, %d/%d spots taken",
				overlappingCount, config.MaxConcurrentBookings)
			return ErrSlotNotAvailable
		}

		uc.logger.Info("CreateBooking: slot available, %d/%d spots taken",
			overlappingCount, config.MaxConcurrentBookings)

		// 8.7. Создаем бронирование с денормализацией данных
		booking := &domain.Booking{
			UserID:          req.UserID,
			CompanyID:       req.CompanyID,
			AddressID:       req.AddressID,
			ServiceID:       req.ServiceID,
			CarID:           car.ID,
			BookingDate:     req.Date,
			StartTime:       req.StartTime,
			DurationMinutes: config.SlotDurationMinutes,
			Status:          domain.StatusConfirmed,
			// Денормализация данных услуги
			ServiceName:  service.Name,
			ServicePrice: getServicePrice(service),
			// Денормализация данных автомобиля
			CarBrand:        &car.Brand,
			CarModel:        &car.Model,
			CarLicensePlate: &car.LicensePlate,
			// Заметки
			Notes: req.Notes,
		}

		// 8.8. Сохраняем бронирование
		created, err := uc.bookingRepo.Create(txCtx, booking)
		if err != nil {
			uc.logger.Error("CreateBooking: failed to create booking: %v", err)
			return fmt.Errorf("%w: failed to create booking: %v", ErrInternal, err)
		}

		result = created
		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info("CreateBooking: successfully created booking id=%d", result.ID)

	// Конвертируем в response
	return &Response{
		ID:              result.ID,
		UserID:          result.UserID,
		CompanyID:       result.CompanyID,
		AddressID:       result.AddressID,
		ServiceID:       result.ServiceID,
		CarID:           result.CarID,
		BookingDate:     result.BookingDate,
		StartTime:       result.StartTime,
		DurationMinutes: result.DurationMinutes,
		Status:          string(result.Status),
		ServiceName:     result.ServiceName,
		ServicePrice:    result.ServicePrice,
		CarBrand:        result.CarBrand,
		CarModel:        result.CarModel,
		CarLicensePlate: result.CarLicensePlate,
		Notes:           result.Notes,
		CreatedAt:       result.CreatedAt,
		UpdatedAt:       result.UpdatedAt,
	}, nil
}

// getServicePrice извлекает цену из услуги
// Если цена не указана (nil), возвращает 0.0
func getServicePrice(service *sellerClient.Service) float64 {
	if service.Price == nil {
		return 0.0
	}
	return *service.Price
}

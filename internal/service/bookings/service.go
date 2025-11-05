package bookings

import (
	"context"
	"errors"
	"fmt"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	bookingRepo "github.com/m04kA/SMC-BookingService/internal/infra/storage/booking"
	sellerClient "github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	"github.com/m04kA/SMC-BookingService/internal/service/bookings/models"
)

// Service сервис для работы с бронированиями
type Service struct {
	bookingRepo  BookingRepository
	sellerClient SellerServiceClient
	logger       Logger
}

// NewService создает новый экземпляр сервиса бронирований
func NewService(
	bookingRepo BookingRepository,
	sellerClient SellerServiceClient,
	logger Logger,
) *Service {
	return &Service{
		bookingRepo:  bookingRepo,
		sellerClient: sellerClient,
		logger:       logger,
	}
}

// GetByID получает бронирование по ID
// Проверяет права доступа - пользователь может видеть только своё бронирование
// или если он является менеджером компании
func (s *Service) GetByID(ctx context.Context, id int64, userID int64) (*models.BookingResponse, error) {
	s.logger.Info("GetByID: fetching booking id=%d for user=%d", id, userID)

	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, bookingRepo.ErrBookingNotFound) {
			s.logger.Warn("GetByID: booking id=%d not found", id)
			return nil, ErrBookingNotFound
		}
		s.logger.Error("GetByID: repository error for booking id=%d: %v", id, err)
		return nil, fmt.Errorf("%w: GetByID - repository error: %v", ErrInternal, err)
	}

	// Проверяем права доступа
	if err := s.checkUserAccess(ctx, booking, userID); err != nil {
		s.logger.Warn("GetByID: access denied for user=%d to booking id=%d", userID, id)
		return nil, err
	}

	s.logger.Info("GetByID: successfully fetched booking id=%d", id)
	return models.FromDomainBooking(booking), nil
}

// GetUserBookings получает историю бронирований пользователя
// Опционально фильтрует по статусу
func (s *Service) GetUserBookings(ctx context.Context, req *models.GetUserBookingsRequest) (*models.BookingListResponse, error) {
	s.logger.Info("GetUserBookings: fetching bookings for user=%d, status=%v", req.UserID, req.Status)

	// Конвертируем статус из строки в domain.BookingStatus
	var domainStatus *domain.BookingStatus
	if req.Status != nil {
		status, err := models.ToDomainBookingStatus(*req.Status)
		if err != nil {
			s.logger.Warn("GetUserBookings: invalid status=%s for user=%d", *req.Status, req.UserID)
			return nil, fmt.Errorf("%w: invalid status", ErrInvalidInput)
		}
		domainStatus = &status
	}

	bookings, err := s.bookingRepo.GetByUserID(ctx, req.UserID, domainStatus)
	if err != nil {
		s.logger.Error("GetUserBookings: repository error for user=%d: %v", req.UserID, err)
		return nil, fmt.Errorf("%w: GetUserBookings - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("GetUserBookings: successfully fetched %d bookings for user=%d", len(bookings), req.UserID)
	return models.FromDomainBookingList(bookings), nil
}

// GetCompanyBookings получает бронирования компании с гибкой фильтрацией
// Поддерживает фильтрацию по адресу, периоду, статусу и включению неактивных бронирований
// Доступно только менеджерам компании
//
// Примеры использования:
// - Все активные бронирования: GetCompanyBookings(ctx, &GetCompanyBookingsRequest{CompanyID: 123, UserID: 456})
// - Бронирования на конкретном адресе: указать AddressID
// - Бронирования на дату: StartDate и EndDate указывают на одну дату
// - Бронирования за период: StartDate и EndDate указывают на разные даты
// - Только подтвержденные: указать Status = "confirmed"
// - Включая отменённые: IncludeInactive = true
func (s *Service) GetCompanyBookings(ctx context.Context, req *models.GetCompanyBookingsRequest) (*models.BookingListResponse, error) {
	// Логируем запрос с деталями фильтрации
	logMsg := fmt.Sprintf("GetCompanyBookings: fetching bookings for company=%d, user=%d", req.CompanyID, req.UserID)
	if req.AddressID != nil {
		logMsg += fmt.Sprintf(", address=%d", *req.AddressID)
	}
	if req.StartDate != nil && req.EndDate != nil {
		logMsg += fmt.Sprintf(", period=%s to %s", req.StartDate.Format("2006-01-02"), req.EndDate.Format("2006-01-02"))
	}
	if req.Status != nil {
		logMsg += fmt.Sprintf(", status=%s", *req.Status)
	}
	if req.IncludeInactive {
		logMsg += ", includeInactive=true"
	}
	s.logger.Info(logMsg)

	// Проверяем права доступа менеджера
	if err := s.checkManagerAccess(ctx, req.CompanyID, req.UserID); err != nil {
		return nil, err
	}

	// Конвертируем request в domain фильтр
	filter, err := req.ToDomainFilter()
	if err != nil {
		s.logger.Warn("GetCompanyBookings: invalid filter for company=%d: %v", req.CompanyID, err)
		return nil, fmt.Errorf("%w: invalid filter", ErrInvalidInput)
	}

	// Получаем бронирования с фильтрацией
	bookings, err := s.bookingRepo.GetByCompanyWithFilter(ctx, filter)
	if err != nil {
		s.logger.Error("GetCompanyBookings: repository error for company=%d: %v", req.CompanyID, err)
		return nil, fmt.Errorf("%w: GetCompanyBookings - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("GetCompanyBookings: successfully fetched %d bookings for company=%d", len(bookings), req.CompanyID)
	return models.FromDomainBookingList(bookings), nil
}

// Cancel отменяет бронирование
// Пользователь может отменить только своё бронирование (cancelled_by_user)
// Менеджер может отменить любое бронирование компании (cancelled_by_company)
func (s *Service) Cancel(ctx context.Context, bookingID int64, req *models.CancelBookingRequest) error {
	s.logger.Info("Cancel: cancelling booking id=%d by user=%d", bookingID, req.UserID)

	// Получаем бронирование
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, bookingRepo.ErrBookingNotFound) {
			s.logger.Warn("Cancel: booking id=%d not found", bookingID)
			return ErrBookingNotFound
		}
		s.logger.Error("Cancel: repository error for booking id=%d: %v", bookingID, err)
		return fmt.Errorf("%w: Cancel - repository error: %v", ErrInternal, err)
	}

	// Проверяем, можно ли отменить бронирование
	if !booking.CanBeCancelled() {
		s.logger.Warn("Cancel: booking id=%d cannot be cancelled, status=%s", bookingID, booking.Status)
		return ErrCannotCancel
	}

	// Определяем статус отмены в зависимости от прав доступа
	var cancelStatus domain.BookingStatus

	// Проверяем, является ли пользователь владельцем бронирования
	if booking.UserID == req.UserID {
		cancelStatus = domain.StatusCancelledByUser
	} else {
		// Проверяем, является ли пользователь менеджером компании
		if err := s.checkManagerAccess(ctx, booking.CompanyID, req.UserID); err != nil {
			s.logger.Warn("Cancel: access denied for user=%d to cancel booking id=%d", req.UserID, bookingID)
			return ErrAccessDenied
		}
		cancelStatus = domain.StatusCancelledByCompany
	}

	// Отменяем бронирование
	if err := s.bookingRepo.Cancel(ctx, bookingID, cancelStatus, req.CancellationReason); err != nil {
		if errors.Is(err, bookingRepo.ErrBookingNotFound) {
			s.logger.Warn("Cancel: booking id=%d not found during cancellation", bookingID)
			return ErrBookingNotFound
		}
		s.logger.Error("Cancel: repository error for booking id=%d: %v", bookingID, err)
		return fmt.Errorf("%w: Cancel - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("Cancel: successfully cancelled booking id=%d with status=%s", bookingID, cancelStatus)
	return nil
}

// UpdateStatus обновляет статус бронирования
// Доступно только менеджерам компании
func (s *Service) UpdateStatus(ctx context.Context, bookingID int64, req *models.UpdateStatusRequest) error {
	s.logger.Info("UpdateStatus: updating booking id=%d to status=%s by user=%d",
		bookingID, req.Status, req.UserID)

	// Получаем бронирование
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, bookingRepo.ErrBookingNotFound) {
			s.logger.Warn("UpdateStatus: booking id=%d not found", bookingID)
			return ErrBookingNotFound
		}
		s.logger.Error("UpdateStatus: repository error for booking id=%d: %v", bookingID, err)
		return fmt.Errorf("%w: UpdateStatus - repository error: %v", ErrInternal, err)
	}

	// Проверяем права доступа (только менеджер компании)
	if err := s.checkManagerAccess(ctx, booking.CompanyID, req.UserID); err != nil {
		return err
	}

	// Валидируем и конвертируем статус
	newStatus, err := models.ToDomainBookingStatus(req.Status)
	if err != nil {
		s.logger.Warn("UpdateStatus: invalid status=%s for booking id=%d", req.Status, bookingID)
		return fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	// Обновляем статус
	if err := s.bookingRepo.UpdateStatus(ctx, bookingID, newStatus); err != nil {
		if errors.Is(err, bookingRepo.ErrBookingNotFound) {
			s.logger.Warn("UpdateStatus: booking id=%d not found during update", bookingID)
			return ErrBookingNotFound
		}
		s.logger.Error("UpdateStatus: repository error for booking id=%d: %v", bookingID, err)
		return fmt.Errorf("%w: UpdateStatus - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("UpdateStatus: successfully updated booking id=%d to status=%s", bookingID, newStatus)
	return nil
}

// Вспомогательные методы

// checkUserAccess проверяет, что пользователь имеет доступ к бронированию
// Пользователь может видеть своё бронирование или если он менеджер компании
func (s *Service) checkUserAccess(ctx context.Context, booking *domain.Booking, userID int64) error {
	// Если пользователь владелец бронирования - доступ разрешён
	if booking.UserID == userID {
		return nil
	}

	// Проверяем, является ли пользователь менеджером компании
	if err := s.checkManagerAccess(ctx, booking.CompanyID, userID); err != nil {
		// Ошибка уже залогирована в checkManagerAccess
		return ErrAccessDenied
	}

	return nil
}

// checkManagerAccess проверяет, что пользователь является менеджером компании
func (s *Service) checkManagerAccess(ctx context.Context, companyID int64, userID int64) error {
	// Получаем компанию через SellerService
	company, err := s.sellerClient.GetCompany(ctx, companyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			s.logger.Warn("checkManagerAccess: company id=%d not found", companyID)
			return ErrCompanyNotFound
		}
		s.logger.Error("checkManagerAccess: failed to get company id=%d: %v", companyID, err)
		return fmt.Errorf("%w: checkManagerAccess - failed to get company: %v", ErrInternal, err)
	}

	// Проверяем, что userID в списке менеджеров
	for _, managerID := range company.ManagerIDs {
		if managerID == userID {
			s.logger.Info("checkManagerAccess: user=%d is manager of company=%d", userID, companyID)
			return nil
		}
	}

	s.logger.Warn("checkManagerAccess: user=%d is not a manager of company=%d", userID, companyID)
	return ErrAccessDenied
}

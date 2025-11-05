package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/m04kA/SMC-BookingService/internal/domain"
	configRepo "github.com/m04kA/SMC-BookingService/internal/infra/storage/config"
	sellerClient "github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	"github.com/m04kA/SMC-BookingService/internal/service/config/models"
)

// Service сервис для работы с конфигурацией слотов
type Service struct {
	configRepo   ConfigRepository
	sellerClient SellerServiceClient
	logger       Logger
}

// NewService создает новый экземпляр сервиса конфигурации
func NewService(
	configRepo ConfigRepository,
	sellerClient SellerServiceClient,
	logger Logger,
) *Service {
	return &Service{
		configRepo:   configRepo,
		sellerClient: sellerClient,
		logger:       logger,
	}
}

// Create создает новую конфигурацию слотов
// Доступно только менеджерам компании
// Проверяет существование компании, адреса (если указан) и услуги (если указана)
func (s *Service) Create(ctx context.Context, req *models.CreateConfigRequest) (*models.ConfigResponse, error) {
	s.logger.Info("Create: creating config for company=%d, address=%v, service=%v by user=%d",
		req.CompanyID, req.AddressID, req.ServiceID, req.UserID)

	// 1. Валидируем входные данные
	if err := s.validateConfigData(req.SlotDurationMinutes, req.MaxConcurrentBookings,
		req.AdvanceBookingDays, req.MinBookingNoticeMinutes); err != nil {
		s.logger.Warn("Create: validation failed: %v", err)
		return nil, err
	}

	// 2. Получаем компанию для проверки прав доступа и адресов
	company, err := s.sellerClient.GetCompany(ctx, req.CompanyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			s.logger.Warn("Create: company id=%d not found", req.CompanyID)
			return nil, ErrCompanyNotFound
		}
		s.logger.Error("Create: failed to get company id=%d: %v", req.CompanyID, err)
		return nil, fmt.Errorf("%w: failed to get company: %v", ErrInternal, err)
	}

	// 3. Проверяем права доступа (только менеджер компании)
	if !s.isManager(company, req.UserID) {
		s.logger.Warn("Create: user=%d is not a manager of company=%d", req.UserID, req.CompanyID)
		return nil, ErrAccessDenied
	}

	// 4. Если указан addressID, проверяем его существование
	if req.AddressID != nil {
		if !s.addressExists(company, *req.AddressID) {
			s.logger.Warn("Create: address id=%d not found in company=%d", *req.AddressID, req.CompanyID)
			return nil, ErrAddressNotFound
		}
	}

	// 5. Если указан serviceID, проверяем его существование и привязку к адресу
	if req.ServiceID != nil {
		service, err := s.sellerClient.GetService(ctx, req.CompanyID, *req.ServiceID)
		if err != nil {
			if errors.Is(err, sellerClient.ErrServiceNotFound) {
				s.logger.Warn("Create: service id=%d not found in company=%d", *req.ServiceID, req.CompanyID)
				return nil, ErrServiceNotFound
			}
			s.logger.Error("Create: failed to get service id=%d: %v", *req.ServiceID, err)
			return nil, fmt.Errorf("%w: failed to get service: %v", ErrInternal, err)
		}

		// Если указан и адрес, и услуга - проверяем, что услуга доступна на этом адресе
		if req.AddressID != nil {
			if !s.serviceAtAddress(service, *req.AddressID) {
				s.logger.Warn("Create: service id=%d is not available at address id=%d",
					*req.ServiceID, *req.AddressID)
				return nil, fmt.Errorf("%w: service is not available at this address", ErrInvalidInput)
			}
		}
	}

	// 6. Проверяем, не существует ли уже конфигурация с такими параметрами
	existingConfig, err := s.configRepo.GetByCompanyAddressAndService(ctx, req.CompanyID, req.AddressID, req.ServiceID)
	if err != nil && !errors.Is(err, configRepo.ErrConfigNotFound) {
		s.logger.Error("Create: failed to check existing config: %v", err)
		return nil, fmt.Errorf("%w: failed to check existing config: %v", ErrInternal, err)
	}
	if existingConfig != nil {
		s.logger.Warn("Create: config already exists for company=%d, address=%v, service=%v",
			req.CompanyID, req.AddressID, req.ServiceID)
		return nil, ErrConfigAlreadyExists
	}

	// 7. Создаем конфигурацию
	domainConfig := req.ToDomainConfig()
	createdConfig, err := s.configRepo.Create(ctx, domainConfig)
	if err != nil {
		s.logger.Error("Create: repository error: %v", err)
		return nil, fmt.Errorf("%w: Create - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("Create: successfully created config id=%d", createdConfig.ID)
	return models.FromDomainConfig(createdConfig), nil
}

// GetByID получает конфигурацию по ID
// Публичный метод - доступен всем
func (s *Service) GetByID(ctx context.Context, id int64) (*models.ConfigResponse, error) {
	s.logger.Info("GetByID: fetching config id=%d", id)

	config, err := s.configRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, configRepo.ErrConfigNotFound) {
			s.logger.Warn("GetByID: config id=%d not found", id)
			return nil, ErrConfigNotFound
		}
		s.logger.Error("GetByID: repository error for config id=%d: %v", id, err)
		return nil, fmt.Errorf("%w: GetByID - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("GetByID: successfully fetched config id=%d", id)
	return models.FromDomainConfig(config), nil
}

// GetWithHierarchy получает конфигурацию с учетом иерархии приоритетов
// Публичный метод - используется для получения актуальной конфигурации при бронировании
// Приоритет: service@address > address > service > global
func (s *Service) GetWithHierarchy(ctx context.Context, req *models.GetConfigRequest) (*models.ConfigResponse, error) {
	s.logger.Info("GetWithHierarchy: fetching config for company=%d, address=%d, service=%d",
		req.CompanyID, req.AddressID, req.ServiceID)

	config, err := s.configRepo.GetConfigWithHierarchy(ctx, req.CompanyID, req.AddressID, req.ServiceID)
	if err != nil {
		if errors.Is(err, configRepo.ErrConfigNotFound) {
			s.logger.Warn("GetWithHierarchy: no config found for company=%d, address=%d, service=%d",
				req.CompanyID, req.AddressID, req.ServiceID)
			return nil, ErrConfigNotFound
		}
		s.logger.Error("GetWithHierarchy: repository error: %v", err)
		return nil, fmt.Errorf("%w: GetWithHierarchy - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("GetWithHierarchy: successfully fetched config id=%d (level: %s)",
		config.ID, s.getConfigLevel(config))
	return models.FromDomainConfig(config), nil
}

// GetAllByCompany получает все конфигурации компании
// Доступно только менеджерам компании
func (s *Service) GetAllByCompany(ctx context.Context, companyID int64, userID int64) (*models.ConfigListResponse, error) {
	s.logger.Info("GetAllByCompany: fetching configs for company=%d by user=%d", companyID, userID)

	// Получаем компанию для проверки прав доступа
	company, err := s.sellerClient.GetCompany(ctx, companyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			s.logger.Warn("GetAllByCompany: company id=%d not found", companyID)
			return nil, ErrCompanyNotFound
		}
		s.logger.Error("GetAllByCompany: failed to get company id=%d: %v", companyID, err)
		return nil, fmt.Errorf("%w: failed to get company: %v", ErrInternal, err)
	}

	// Проверяем права доступа (только менеджер компании)
	if !s.isManager(company, userID) {
		s.logger.Warn("GetAllByCompany: user=%d is not a manager of company=%d", userID, companyID)
		return nil, ErrAccessDenied
	}

	configs, err := s.configRepo.GetAllByCompany(ctx, companyID)
	if err != nil {
		s.logger.Error("GetAllByCompany: repository error for company=%d: %v", companyID, err)
		return nil, fmt.Errorf("%w: GetAllByCompany - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("GetAllByCompany: successfully fetched %d configs for company=%d", len(configs), companyID)
	return models.FromDomainConfigList(configs), nil
}

// Update обновляет существующую конфигурацию
// Доступно только менеджерам компании
// Поддерживает частичное обновление - обновляются только указанные поля
func (s *Service) Update(ctx context.Context, id int64, req *models.UpdateConfigRequest) (*models.ConfigResponse, error) {
	s.logger.Info("Update: updating config id=%d by user=%d", id, req.UserID)

	// 1. Получаем существующую конфигурацию
	config, err := s.configRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, configRepo.ErrConfigNotFound) {
			s.logger.Warn("Update: config id=%d not found", id)
			return nil, ErrConfigNotFound
		}
		s.logger.Error("Update: repository error for config id=%d: %v", id, err)
		return nil, fmt.Errorf("%w: Update - repository error: %v", ErrInternal, err)
	}

	// 2. Применяем обновления к конфигурации (создаём копию для валидации)
	tempConfig := *config
	req.ApplyToConfig(&tempConfig)

	// 3. Валидируем обновленные данные
	if err := s.validateConfigData(tempConfig.SlotDurationMinutes, tempConfig.MaxConcurrentBookings,
		tempConfig.AdvanceBookingDays, tempConfig.MinBookingNoticeMinutes); err != nil {
		s.logger.Warn("Update: validation failed for config id=%d: %v", id, err)
		return nil, err
	}

	// 4. Получаем компанию для проверки прав доступа
	company, err := s.sellerClient.GetCompany(ctx, config.CompanyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			s.logger.Warn("Update: company id=%d not found", config.CompanyID)
			return nil, ErrCompanyNotFound
		}
		s.logger.Error("Update: failed to get company id=%d: %v", config.CompanyID, err)
		return nil, fmt.Errorf("%w: failed to get company: %v", ErrInternal, err)
	}

	// 5. Проверяем права доступа (только менеджер компании)
	if !s.isManager(company, req.UserID) {
		s.logger.Warn("Update: user=%d is not a manager of company=%d", req.UserID, config.CompanyID)
		return nil, ErrAccessDenied
	}

	// 6. Применяем обновления к оригинальной конфигурации
	req.ApplyToConfig(config)

	// 7. Обновляем конфигурацию в БД
	updatedConfig, err := s.configRepo.Update(ctx, id, config)
	if err != nil {
		if errors.Is(err, configRepo.ErrConfigNotFound) {
			s.logger.Warn("Update: config id=%d not found during update", id)
			return nil, ErrConfigNotFound
		}
		s.logger.Error("Update: repository error for config id=%d: %v", id, err)
		return nil, fmt.Errorf("%w: Update - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("Update: successfully updated config id=%d", id)
	return models.FromDomainConfig(updatedConfig), nil
}

// Delete удаляет конфигурацию по ID
// Доступно только менеджерам компании
func (s *Service) Delete(ctx context.Context, id int64, userID int64) error {
	s.logger.Info("Delete: deleting config id=%d by user=%d", id, userID)

	// 1. Получаем конфигурацию для проверки прав доступа
	config, err := s.configRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, configRepo.ErrConfigNotFound) {
			s.logger.Warn("Delete: config id=%d not found", id)
			return ErrConfigNotFound
		}
		s.logger.Error("Delete: repository error for config id=%d: %v", id, err)
		return fmt.Errorf("%w: Delete - repository error: %v", ErrInternal, err)
	}

	// 2. Получаем компанию для проверки прав доступа
	company, err := s.sellerClient.GetCompany(ctx, config.CompanyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			s.logger.Warn("Delete: company id=%d not found", config.CompanyID)
			return ErrCompanyNotFound
		}
		s.logger.Error("Delete: failed to get company id=%d: %v", config.CompanyID, err)
		return fmt.Errorf("%w: failed to get company: %v", ErrInternal, err)
	}

	// 3. Проверяем права доступа (только менеджер компании)
	if !s.isManager(company, userID) {
		s.logger.Warn("Delete: user=%d is not a manager of company=%d", userID, config.CompanyID)
		return ErrAccessDenied
	}

	// 4. Удаляем конфигурацию
	if err := s.configRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, configRepo.ErrConfigNotFound) {
			s.logger.Warn("Delete: config id=%d not found during deletion", id)
			return ErrConfigNotFound
		}
		s.logger.Error("Delete: repository error for config id=%d: %v", id, err)
		return fmt.Errorf("%w: Delete - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("Delete: successfully deleted config id=%d", id)
	return nil
}

// DeleteByKey удаляет конфигурацию по ключу (company_id, address_id, service_id)
// Доступно только менеджерам компании
func (s *Service) DeleteByKey(ctx context.Context, req *models.DeleteConfigRequest) error {
	s.logger.Info("DeleteByKey: deleting config for company=%d, address=%v, service=%v by user=%d",
		req.CompanyID, req.AddressID, req.ServiceID, req.UserID)

	// 1. Получаем компанию для проверки прав доступа
	company, err := s.sellerClient.GetCompany(ctx, req.CompanyID)
	if err != nil {
		if errors.Is(err, sellerClient.ErrCompanyNotFound) {
			s.logger.Warn("DeleteByKey: company id=%d not found", req.CompanyID)
			return ErrCompanyNotFound
		}
		s.logger.Error("DeleteByKey: failed to get company id=%d: %v", req.CompanyID, err)
		return fmt.Errorf("%w: failed to get company: %v", ErrInternal, err)
	}

	// 2. Проверяем права доступа (только менеджер компании)
	if !s.isManager(company, req.UserID) {
		s.logger.Warn("DeleteByKey: user=%d is not a manager of company=%d", req.UserID, req.CompanyID)
		return ErrAccessDenied
	}

	// 3. Удаляем конфигурацию по ключу
	if err := s.configRepo.DeleteByCompanyAddressAndService(ctx, req.CompanyID, req.AddressID, req.ServiceID); err != nil {
		if errors.Is(err, configRepo.ErrConfigNotFound) {
			s.logger.Warn("DeleteByKey: config not found for company=%d, address=%v, service=%v",
				req.CompanyID, req.AddressID, req.ServiceID)
			return ErrConfigNotFound
		}
		s.logger.Error("DeleteByKey: repository error: %v", err)
		return fmt.Errorf("%w: DeleteByKey - repository error: %v", ErrInternal, err)
	}

	s.logger.Info("DeleteByKey: successfully deleted config for company=%d, address=%v, service=%v",
		req.CompanyID, req.AddressID, req.ServiceID)
	return nil
}

// Вспомогательные методы

// isManager проверяет, что пользователь является менеджером компании
func (s *Service) isManager(company *sellerClient.Company, userID int64) bool {
	for _, managerID := range company.ManagerIDs {
		if managerID == userID {
			return true
		}
	}
	return false
}

// validateConfigData валидирует параметры конфигурации
func (s *Service) validateConfigData(slotDuration, maxConcurrent, advanceDays, minNotice int) error {
	// Проверяем slotDurationMinutes
	if slotDuration <= 0 || slotDuration > 480 { // максимум 8 часов
		return fmt.Errorf("%w: slotDurationMinutes must be between 1 and 480", ErrInvalidInput)
	}

	// Проверяем maxConcurrentBookings
	if maxConcurrent <= 0 || maxConcurrent > 100 {
		return fmt.Errorf("%w: maxConcurrentBookings must be between 1 and 100", ErrInvalidInput)
	}

	// Проверяем advanceBookingDays
	if advanceDays < 0 || advanceDays > 365 {
		return fmt.Errorf("%w: advanceBookingDays must be between 0 and 365", ErrInvalidInput)
	}

	// Проверяем minBookingNoticeMinutes
	if minNotice < 0 || minNotice > 10080 { // максимум 7 дней в минутах
		return fmt.Errorf("%w: minBookingNoticeMinutes must be between 0 and 10080", ErrInvalidInput)
	}

	return nil
}

// addressExists проверяет, что адрес существует в компании
func (s *Service) addressExists(company *sellerClient.Company, addressID int64) bool {
	for _, addr := range company.Addresses {
		if addr.ID == addressID {
			return true
		}
	}
	return false
}

// serviceAtAddress проверяет, что услуга доступна на указанном адресе
func (s *Service) serviceAtAddress(service *sellerClient.Service, addressID int64) bool {
	for _, addrID := range service.AddressIDs {
		if addrID == addressID {
			return true
		}
	}
	return false
}

// getConfigLevel возвращает строковое представление уровня конфигурации для логирования
func (s *Service) getConfigLevel(config *domain.CompanySlotsConfig) string {
	if config.IsServiceAtAddress() {
		return "service@address"
	}
	if config.IsAddressSpecific() {
		return "address"
	}
	if config.IsServiceSpecific() {
		return "service"
	}
	return "global"
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	cancelBookingHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/cancel_booking"
	createBookingHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/create_booking"
	getAvailableSlotsHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/get_available_slots"
	getBookingHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/get_booking"
	getCompanyBookingsHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/get_company_bookings"
	getCompanyConfigHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/get_company_config"
	getUserBookingsHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/get_user_bookings"
	updateCompanyConfigHandler "github.com/m04kA/SMC-BookingService/internal/api/handlers/update_company_config"
	"github.com/m04kA/SMC-BookingService/internal/api/middleware"
	"github.com/m04kA/SMC-BookingService/internal/config"
	bookingRepo "github.com/m04kA/SMC-BookingService/internal/infra/storage/booking"
	configRepo "github.com/m04kA/SMC-BookingService/internal/infra/storage/config"
	sellerServiceClient "github.com/m04kA/SMC-BookingService/internal/integrations/sellerservice"
	userServiceClient "github.com/m04kA/SMC-BookingService/internal/integrations/userservice"
	bookingsService "github.com/m04kA/SMC-BookingService/internal/service/bookings"
	configService "github.com/m04kA/SMC-BookingService/internal/service/config"
	createBookingUC "github.com/m04kA/SMC-BookingService/internal/usecase/create_booking"
	getAvailableSlotsUC "github.com/m04kA/SMC-BookingService/internal/usecase/get_available_slots"
	"github.com/m04kA/SMC-BookingService/pkg/dbmetrics"
	"github.com/m04kA/SMC-BookingService/pkg/logger"
	"github.com/m04kA/SMC-BookingService/pkg/metrics"
	"github.com/m04kA/SMC-BookingService/pkg/simpletxmanager"
	"github.com/m04kA/SMC-BookingService/pkg/txmanager"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load("config.toml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Инициализируем логгер
	log, err := logger.New(cfg.Logs.File, cfg.Logs.Level)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("Starting SMC-BookingService...")
	log.Info("Configuration loaded from config.toml")

	// Инициализируем метрики (если включены)
	var metricsCollector *metrics.Metrics
	var wrappedDB *dbmetrics.DB
	stopMetricsCh := make(chan struct{})

	if cfg.Metrics.Enabled {
		metricsCollector = metrics.New(cfg.Metrics.ServiceName)
		log.Info("Metrics enabled at %s", cfg.Metrics.Path)
	}

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatal("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Настраиваем connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database: %v", err)
	}
	log.Info("Successfully connected to database (host=%s, port=%d, db=%s)",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// Инициализируем интеграционных клиентов
	userClient := userServiceClient.NewClient(
		cfg.UserService.URL,
		time.Duration(cfg.UserService.Timeout)*time.Second,
		log,
	)
	sellerClient := sellerServiceClient.NewClient(
		cfg.SellerService.URL,
		time.Duration(cfg.SellerService.Timeout)*time.Second,
		log,
	)
	log.Info("Integration clients initialized (UserService=%s timeout=%ds, SellerService=%s timeout=%ds)",
		cfg.UserService.URL, cfg.UserService.Timeout, cfg.SellerService.URL, cfg.SellerService.Timeout)

	// Инициализируем репозитории и сервисы (с метриками или без)
	var (
		bookingRepository *bookingRepo.Repository
		configRepository  *configRepo.Repository
	)

	// Интерфейс для transaction manager (используется в usecases)
	// TODO: Точно нужно переделать эту шл
	type TxManager interface {
		DoSerializable(ctx context.Context, fn func(ctx context.Context) error) error
	}
	var txMgr TxManager

	if cfg.Metrics.Enabled {
		wrappedDB = dbmetrics.WrapWithDefault(db, metricsCollector, cfg.Metrics.ServiceName, stopMetricsCh)
		log.Info("Database metrics collection started")

		// Инициализируем репозитории с обёрткой метрик
		bookingRepository = bookingRepo.NewRepository(wrappedDB)
		configRepository = configRepo.NewRepository(wrappedDB)
		txMgr = txmanager.NewTransactionManager(wrappedDB)
	} else {
		// Инициализируем репозитории без метрик
		bookingRepository = bookingRepo.NewRepository(db)
		configRepository = configRepo.NewRepository(db)
		txMgr = simpletxmanager.NewTransactionManager(db)
	}

	// Инициализируем сервисы
	bookingSvc := bookingsService.NewService(
		bookingRepository,
		sellerClient,
		log,
	)
	configSvc := configService.NewService(
		configRepository,
		sellerClient,
		log,
	)

	// Инициализируем use cases
	createBookingUseCase := createBookingUC.NewUseCase(
		bookingRepository,
		configRepository,
		sellerClient,
		userClient,
		txMgr,
		log,
	)

	getAvailableSlotsUseCase := getAvailableSlotsUC.NewUseCase(
		bookingRepository,
		configRepository,
		sellerClient,
		log,
	)

	// Инициализируем handlers
	createBooking := createBookingHandler.NewHandler(createBookingUseCase, log)
	getAvailableSlots := getAvailableSlotsHandler.NewHandler(getAvailableSlotsUseCase, log)
	getBooking := getBookingHandler.NewHandler(bookingSvc, log)
	cancelBooking := cancelBookingHandler.NewHandler(bookingSvc, log)
	getUserBookings := getUserBookingsHandler.NewHandler(bookingSvc, log)
	getCompanyBookings := getCompanyBookingsHandler.NewHandler(bookingSvc, log)
	getCompanyConfig := getCompanyConfigHandler.NewHandler(configSvc, log)
	updateCompanyConfig := updateCompanyConfigHandler.NewHandler(configSvc, log)

	// Настраиваем роутер
	r := mux.NewRouter()

	// Добавляем metrics middleware (если метрики включены)
	if cfg.Metrics.Enabled {
		r.Use(middleware.MetricsMiddleware(metricsCollector, cfg.Metrics.ServiceName))
		log.Info("HTTP metrics middleware enabled")
	}

	// Metrics endpoint (публичный, без аутентификации)
	if cfg.Metrics.Enabled {
		r.Handle(cfg.Metrics.Path, promhttp.Handler()).Methods(http.MethodGet)
		log.Info("Prometheus metrics endpoint exposed at %s", cfg.Metrics.Path)
	}

	// API prefix
	api := r.PathPrefix("/api/v1").Subrouter()

	// ============================================================
	// PUBLIC ROUTES (без аутентификации)
	// ============================================================

	// Получение доступных слотов для бронирования
	api.HandleFunc("/companies/{companyId}/addresses/{addressId}/available-slots",
		getAvailableSlots.Handle).Methods(http.MethodGet)

	// Получение конфигурации слотов компании
	api.HandleFunc("/companies/{companyId}/config",
		getCompanyConfig.Handle).Methods(http.MethodGet)

	// ============================================================
	// PROTECTED ROUTES (требуют X-User-ID header)
	// ============================================================

	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.Auth)

	// --- Бронирования ---
	// Создание бронирования
	protected.HandleFunc("/bookings", createBooking.Handle).Methods(http.MethodPost)

	// Получение бронирования по ID
	protected.HandleFunc("/bookings/{bookingId}", getBooking.Handle).Methods(http.MethodGet)

	// Отмена бронирования
	protected.HandleFunc("/bookings/{bookingId}/cancel", cancelBooking.Handle).Methods(http.MethodPatch)

	// История бронирований пользователя
	protected.HandleFunc("/users/{userId}/bookings", getUserBookings.Handle).Methods(http.MethodGet)

	// --- Управление компанией (для менеджеров) ---
	// Список бронирований компании
	protected.HandleFunc("/companies/{companyId}/bookings", getCompanyBookings.Handle).Methods(http.MethodGet)

	// Обновление конфигурации компании
	protected.HandleFunc("/companies/{companyId}/config", updateCompanyConfig.Handle).Methods(http.MethodPut)

	// Создаем HTTP сервер
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Info("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Останавливаем сбор метрик connection pool
	if cfg.Metrics.Enabled {
		close(stopMetricsCh)
		log.Info("Metrics collection stopped")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout)*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped gracefully")
}

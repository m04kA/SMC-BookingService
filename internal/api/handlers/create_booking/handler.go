package create_booking

import (
	"errors"
	"net/http"

	"github.com/m04kA/SMC-BookingService/internal/api/handlers"
	createBooking "github.com/m04kA/SMC-BookingService/internal/usecase/create_booking"
)

const (
	msgInvalidRequestBody  = "некорректное тело запроса"
	msgInvalidDate         = "некорректный формат даты бронирования, ожидается YYYY-MM-DD"
	msgInvalidTime         = "некорректный формат времени начала, ожидается HH:MM"
	msgSlotNotAvailable    = "выбранный временной слот недоступен"
	msgCompanyNotFound     = "компания не найдена"
	msgServiceNotFound     = "услуга не найдена"
	msgAddressNotFound     = "адрес не найден"
	msgCarNotFound         = "автомобиль не найден"
	msgCompanyClosed       = "компания закрыта в выбранную дату"
	msgInvalidBookingDate  = "некорректная дата бронирования"
	msgDateTooFar          = "дата бронирования слишком далеко в будущем"
	msgInvalidTimeSlot     = "некорректный временной слот"
	msgTooLateToBook       = "слишком поздно для бронирования этого слота"
	msgServiceNotAvailable = "услуга недоступна на выбранном адресе"
)

type Handler struct {
	useCase CreateBookingUseCase
	logger  Logger
}

func NewHandler(useCase CreateBookingUseCase, logger Logger) *Handler {
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Handle POST /api/v1/bookings
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		h.logger.Warn("POST /bookings - Invalid request body: %v", err)
		handlers.RespondBadRequest(w, msgInvalidRequestBody)
		return
	}

	// Конвертируем HTTP запрос в модель use case (с парсингом даты и времени)
	useCaseReq, err := req.ToUseCaseRequest()
	if err != nil {
		h.logger.Warn("POST /bookings - Failed to parse request: %v", err)
		// Определяем тип ошибки парсинга
		if err.Error() == "parsing time" || err.Error() == "invalid time string format" {
			handlers.RespondBadRequest(w, msgInvalidTime)
		} else {
			handlers.RespondBadRequest(w, msgInvalidDate)
		}
		return
	}

	// Вызываем use case
	result, err := h.useCase.Execute(r.Context(), useCaseReq)
	if err != nil {
		// Обработка ошибок use case
		switch {
		case errors.Is(err, createBooking.ErrSlotNotAvailable):
			h.logger.Warn("POST /bookings - Slot not available: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondError(w, http.StatusConflict, msgSlotNotAvailable)

		case errors.Is(err, createBooking.ErrCompanyNotFound):
			h.logger.Warn("POST /bookings - Company not found: company_id=%d", req.CompanyID)
			handlers.RespondNotFound(w, msgCompanyNotFound)

		case errors.Is(err, createBooking.ErrServiceNotFound):
			h.logger.Warn("POST /bookings - Service not found: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondNotFound(w, msgServiceNotFound)

		case errors.Is(err, createBooking.ErrAddressNotFound):
			h.logger.Warn("POST /bookings - Address not found: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondNotFound(w, msgAddressNotFound)

		case errors.Is(err, createBooking.ErrCarNotFound):
			h.logger.Warn("POST /bookings - Car not found: user_id=%d", req.UserID)
			handlers.RespondNotFound(w, msgCarNotFound)

		case errors.Is(err, createBooking.ErrCompanyClosed):
			h.logger.Warn("POST /bookings - Company closed: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondBadRequest(w, msgCompanyClosed)

		case errors.Is(err, createBooking.ErrInvalidDate):
			h.logger.Warn("POST /bookings - Invalid booking date: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondBadRequest(w, msgInvalidBookingDate)

		case errors.Is(err, createBooking.ErrDateTooFarInFuture):
			h.logger.Warn("POST /bookings - Date too far in future: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondBadRequest(w, msgDateTooFar)

		case errors.Is(err, createBooking.ErrInvalidTimeSlot):
			h.logger.Warn("POST /bookings - Invalid time slot: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondBadRequest(w, msgInvalidTimeSlot)

		case errors.Is(err, createBooking.ErrTooLateToBook):
			h.logger.Warn("POST /bookings - Too late to book: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondBadRequest(w, msgTooLateToBook)

		case errors.Is(err, createBooking.ErrServiceNotAvailableAtAddress):
			h.logger.Warn("POST /bookings - Service not available at address: user_id=%d, company_id=%d", req.UserID, req.CompanyID)
			handlers.RespondBadRequest(w, msgServiceNotAvailable)

		default:
			h.logger.Error("POST /bookings - Failed to create booking: user_id=%d, company_id=%d, error=%v",
				req.UserID, req.CompanyID, err)
			handlers.RespondInternalError(w)
		}
		return
	}

	// Формируем HTTP ответ
	response := FromUseCaseResponse(result)

	h.logger.Info("POST /bookings - Booking created successfully: booking_id=%d, user_id=%d, company_id=%d",
		result.ID, req.UserID, req.CompanyID)
	handlers.RespondJSON(w, http.StatusCreated, response)
}

package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client клиент для работы с UserService
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        Logger
}

// NewClient создает новый экземпляр клиента UserService
func NewClient(baseURL string, timeout time.Duration, log Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log,
	}
}

// GetSelectedCar получает выбранный автомобиль пользователя
func (c *Client) GetSelectedCar(ctx context.Context, tgUserID int64) (*Car, error) {
	url := fmt.Sprintf("%s/internal/users/%d/cars/selected", c.baseURL, tgUserID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrInternal, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to execute request: %v", ErrInternal, err)
	}
	defer resp.Body.Close()

	// Обработка статус-кодов
	switch resp.StatusCode {
	case http.StatusOK:
		// Продолжаем обработку
	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: invalid user ID format", ErrInvalidResponse)
	case http.StatusNotFound:
		return nil, ErrCarNotFound
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: unexpected status code %d: %s", ErrInvalidResponse, resp.StatusCode, string(body))
	}

	// Парсим ответ
	var car Car
	if err := json.NewDecoder(resp.Body).Decode(&car); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ErrInvalidResponse, err)
	}

	return &car, nil
}

// GetSelectedCarWithGracefulDegradation получает выбранный автомобиль пользователя с graceful degradation
// При недоступности UserService возвращает ErrServiceDegraded, что позволяет сервису использовать базовые цены
func (c *Client) GetSelectedCarWithGracefulDegradation(ctx context.Context, tgUserID int64) (*Car, error) {
	c.log.Info("Fetching selected car for tg_user_id=%d", tgUserID)

	car, err := c.GetSelectedCar(ctx, tgUserID)
	if err != nil {
		// Если это критичная бизнес-ошибка (не найден автомобиль),
		// пробрасываем её дальше
		if err == ErrCarNotFound {
			c.log.Info("No selected car found for tg_user_id=%d", tgUserID)
			return nil, err
		}

		// Для всех остальных ошибок (недоступность сервиса, timeout, ошибки парсинга и т.д.)
		// применяем graceful degradation - возвращаем ErrServiceDegraded с контекстом
		// Повышаем уровень логирования до ERROR, чтобы быстрее заметить проблему
		c.log.Error("UserService unavailable, applying graceful degradation for tg_user_id=%d: %v", tgUserID, err)
		return nil, fmt.Errorf("%w: tg_user_id=%d, error=%v", ErrServiceDegraded, tgUserID, err)
	}

	c.log.Info("Successfully fetched car for tg_user_id=%d, vehicle_class=%s", tgUserID, car.Size)
	return car, nil
}

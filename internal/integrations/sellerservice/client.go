package sellerservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client клиент для работы с SellerService
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        Logger
}

// NewClient создает новый экземпляр клиента SellerService
func NewClient(baseURL string, timeout time.Duration, log Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log,
	}
}

// GetCompany получает информацию о компании по ID
func (c *Client) GetCompany(ctx context.Context, companyID int64) (*Company, error) {
	url := fmt.Sprintf("%s/api/v1/companies/%d", c.baseURL, companyID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrInternal, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to execute request: %v", ErrInternal, err)
	}
	defer resp.Body.Close()

	// Обработка статус-кодов
	switch resp.StatusCode {
	case http.StatusOK:
		// Продолжаем обработку
	case http.StatusNotFound:
		return nil, ErrCompanyNotFound
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: unexpected status code %d: %s", ErrInvalidResponse, resp.StatusCode, string(body))
	}

	// Парсим ответ
	var company Company
	if err := json.NewDecoder(resp.Body).Decode(&company); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ErrInvalidResponse, err)
	}

	return &company, nil
}

// GetService получает информацию об услуге по ID
func (c *Client) GetService(ctx context.Context, companyID, serviceID int64) (*Service, error) {
	url := fmt.Sprintf("%s/api/v1/companies/%d/services/%d", c.baseURL, companyID, serviceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrInternal, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to execute request: %v", ErrInternal, err)
	}
	defer resp.Body.Close()

	// Обработка статус-кодов
	switch resp.StatusCode {
	case http.StatusOK:
		// Продолжаем обработку
	case http.StatusNotFound:
		return nil, ErrServiceNotFound
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: unexpected status code %d: %s", ErrInvalidResponse, resp.StatusCode, string(body))
	}

	// Парсим ответ
	var service Service
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ErrInvalidResponse, err)
	}

	c.log.Info("GetService: decoded service id=%d, address_ids=%v", service.ID, service.AddressIDs)
	return &service, nil
}

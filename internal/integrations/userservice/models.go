package userservice

// Car модель автомобиля из UserService
type Car struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"user_id"`
	Brand        string  `json:"brand"`
	Model        string  `json:"model"`
	LicensePlate string  `json:"license_plate"`
	Color        string  `json:"color"`
	Size         string  `json:"size"` // Класс автомобиля (A, B, C, D, E, F, J, M, S)
	IsSelected   bool    `json:"is_selected"`
}

// ErrorResponse модель ошибки от UserService
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

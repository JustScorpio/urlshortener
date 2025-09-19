// Пакет customerrors содержит кастомные типы ошибок
package customerrors

import (
	"fmt"
	"net/http"
)

// HTTPError - ошибка-ответ на HTTP-запрос
type HTTPError struct {
	Code int   // HTTP-статус код
	Err  error // Сообщение для клиента
}

// Error - Реализация интерфейса error
func (err *HTTPError) Error() string {
	return fmt.Sprintf("%v", err.Err)
}

// NewHTTPError - создать HTTP-ошибку
func NewHTTPError(err error, code int) error {
	return &HTTPError{
		Code: code,
		Err:  err,
	}
}

// NewHTTPError - создать ошибку с кодом 409
func NewAlreadyExistsError(err error) error {
	return &HTTPError{
		Code: http.StatusConflict,
		Err:  err,
	}
}

// NewHTTPError - создать ошибку с кодом 405
func NewNotAllowedError(err error) error {
	return &HTTPError{
		Code: http.StatusMethodNotAllowed,
		Err:  err,
	}
}

// NewHTTPError - создать ошибку с кодом 410
func NewGoneError(err error) error {
	return &HTTPError{
		Code: http.StatusGone,
		Err:  err,
	}
}

// NewHTTPError - создать ошибку с кодом 404
func NewNotFoundError(err error) error {
	return &HTTPError{
		Code: http.StatusNotFound,
		Err:  err,
	}
}

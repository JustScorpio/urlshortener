// Пакет customerrors содержит кастомные типы ошибок
package customerrors

import (
	"fmt"
	"net/http"
)

// HTTPError - ошибка-ответ на HTTP-запрос
type HTTPError struct {
	Err  error
	Code int
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

// NewAlreadyExistsError - создать ошибку с кодом 409
func NewAlreadyExistsError(err error) error {
	return &HTTPError{
		Code: http.StatusConflict,
		Err:  err,
	}
}

// NewNotAllowedError - создать ошибку с кодом 405
func NewNotAllowedError(err error) error {
	return &HTTPError{
		Code: http.StatusMethodNotAllowed,
		Err:  err,
	}
}

// NewGoneError - создать ошибку с кодом 410
func NewGoneError(err error) error {
	return &HTTPError{
		Code: http.StatusGone,
		Err:  err,
	}
}

// NewNotFoundError - создать ошибку с кодом 404
func NewNotFoundError(err error) error {
	return &HTTPError{
		Code: http.StatusNotFound,
		Err:  err,
	}
}

// NewHTTPError - создать ошибку с кодом 503
func NewServiceUnavailableError(err error) error {
	return &HTTPError{
		Code: http.StatusServiceUnavailable,
		Err:  err,
	}
}

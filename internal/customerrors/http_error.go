package customerrors

import (
	"fmt"
	"net/http"
)

type HTTPError struct {
	Code int   // HTTP-статус код
	Err  error // Сообщение для клиента
}

func (err *HTTPError) Error() string {
	return fmt.Sprintf("%v", err.Err)
}

func NewHTTPError(err error, code int) error {
	return &HTTPError{
		Code: code,
		Err:  err,
	}
}

func NewAlreadyExistsError(err error) error {
	return &HTTPError{
		Code: http.StatusConflict,
		Err:  err,
	}
}

func NewNotAllowedError(err error) error {
	return &HTTPError{
		Code: http.StatusMethodNotAllowed,
		Err:  err,
	}
}

func NewGoneError(err error) error {
	return &HTTPError{
		Code: http.StatusGone,
		Err:  err,
	}
}

func NewNotFoundError(err error) error {
	return &HTTPError{
		Code: http.StatusNotFound,
		Err:  err,
	}
}

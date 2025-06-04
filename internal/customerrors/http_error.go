package customerrors

import (
	"fmt"
	"net/http"
)

type HttpError struct {
	Code int   // HTTP-статус код
	Err  error // Сообщение для клиента
}

func (err *HttpError) Error() string {
	return fmt.Sprintf("%v", err.Err)
}

func NewHttpError(err error, code int) error {
	return &HttpError{
		Code: code,
		Err:  err,
	}
}

func NewAlreadyExistsError(err error) error {
	return &HttpError{
		Code: http.StatusConflict,
		Err:  err,
	}
}

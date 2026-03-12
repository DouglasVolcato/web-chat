package profile

import "net/http"

type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newValidationError(code, message string) *AppError {
	return &AppError{Code: code, Message: message, Status: http.StatusBadRequest}
}

func newUnauthorizedError() *AppError {
	return &AppError{Code: "unauthorized", Message: "Usuário não autenticado.", Status: http.StatusUnauthorized}
}

func newNotFoundError() *AppError {
	return &AppError{Code: "user_not_found", Message: "Usuário não encontrado.", Status: http.StatusNotFound}
}

func newInternalError(err error) *AppError {
	return &AppError{Code: "internal_error", Message: "Erro interno. Tente novamente.", Status: http.StatusInternalServerError, Err: err}
}

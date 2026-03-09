package errors

import "net/http"

type ServiceError interface {
	error
	Code() string
	Message() string
	Status() int
}

type serviceError struct {
	code    string
	message string
	status  int
}

func (e *serviceError) Error() string   { return e.message }
func (e *serviceError) Code() string    { return e.code }
func (e *serviceError) Message() string { return e.message }
func (e *serviceError) Status() int     { return e.status }

func NewServiceError(code, message string, status int) ServiceError {
	return &serviceError{
		code:    code,
		message: message,
		status:  status,
	}
}

var (
	ErrNotFound       = NewServiceError("not_found", "resource not found", http.StatusNotFound)
	ErrInternal       = NewServiceError("internal_server_error", "internal server error", http.StatusInternalServerError)
	ErrInvalidPayload = NewServiceError("invalid_payload", "invalid payload", http.StatusBadRequest)
	ErrUnauthorized   = NewServiceError("unauthorized", "authentication failed", http.StatusUnauthorized)
)

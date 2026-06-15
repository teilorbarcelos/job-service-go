package errors

import "errors"

type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Cause      error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code, message string, statusCode int) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode}
}

func Wrap(code, message string, statusCode int, cause error) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode, Cause: cause}
}

func NewConfigurationError(message string) *AppError {
	return New("CONFIGURATION_ERROR", message, 500)
}

func NewValidationError(message string) *AppError {
	return New("VALIDATION_ERROR", message, 400)
}

func NewConnectionError(service, message string) *AppError {
	return New("CONNECTION_ERROR", service+": "+message, 503)
}

// IsAppError checks if the given error is an AppError.
func IsAppError(err error) bool {
	var ae *AppError
	return errors.As(err, &ae)
}

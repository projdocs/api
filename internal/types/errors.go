package types

import "fmt"

// AppError is a structured error that carries an HTTP status code.
// Handlers return *AppError instead of writing to the context directly,
// keeping business logic free of gin dependencies.
//
//	if user == nil {
//	    return response.NewError(http.StatusNotFound, "user not found")
//	}
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Status, e.Message)
}

// NewError constructs an *AppError.
func NewError(status int, message string) *AppError {
	return &AppError{Status: status, Message: message}
}

// NewErrorf constructs an *AppError with a formatted message.
func NewErrorf(status int, format string, args ...any) *AppError {
	return &AppError{Status: status, Message: fmt.Sprintf(format, args...)}
}

package custom_error

import "fmt"

type UniqueViolationError struct {
	Message string // User-friendly error message
	Code    string // PostgreSQL error code (e.g., "23505")
}

func (e *UniqueViolationError) Error() string {
	return fmt.Sprintf("%s (code: %s)", e.Message, e.Code)
}

func WrapDBError(message, code string) error {
	return &UniqueViolationError{
		Message: message,
		Code:    code,
	}
}

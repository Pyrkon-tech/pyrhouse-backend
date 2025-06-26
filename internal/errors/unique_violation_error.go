package custom_error

import "fmt"

type CustomError interface {
	Error() string
}

type UniqueViolationError struct {
	message string
	code    string // PostgreSQL error code (e.g., "23505")
}

type ForeignKeyViolationError struct {
	message string
	code    string // PostgreSQL error code (e.g., "23503")
}

func (f *ForeignKeyViolationError) Error() string {
	return fmt.Sprintf("%s (code: %s)", f.message, f.code)
}

func (e *UniqueViolationError) Error() string {
	return fmt.Sprintf("%s (code: %s)", e.message, e.code)
}

func WrapDBError(message, code string) CustomError {
	switch code {
	case "23505":
		return &UniqueViolationError{
			message: message,
			code:    code,
		}
	case "23503":
		return &ForeignKeyViolationError{
			message: "Value is already used by other resources " + message,
			code:    code,
		}
	default:
		return fmt.Errorf("uncategorized error occurred with code %s: %s", code, message)
	}
}

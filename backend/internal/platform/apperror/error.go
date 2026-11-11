// MIVA-SHARED: small application utilities.
package apperror

type AppError struct {
	Status  int
	Message string
}

func NewError(status int, message string) error { return &AppError{status, message} }

func (e *AppError) Error() string { return e.Message }

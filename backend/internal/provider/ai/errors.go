package ai

import "english-app/internal/platform/apperror"

// requestError exposes a safe message while retaining the transport cause.
type requestError struct {
	cause   error
	status  int
	message string
}

func (e *requestError) Error() string { return e.message }
func (e *requestError) Unwrap() error { return e.cause }
func (e *requestError) Timeout() bool {
	return e.status == 408 || e.status == 504 || apperror.IsTimeout(e.cause)
}

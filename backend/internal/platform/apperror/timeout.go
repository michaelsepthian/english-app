package apperror

import (
	"context"
	"errors"
)

func IsTimeout(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	var timeout interface{ Timeout() bool }
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout())
}

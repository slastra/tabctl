package errors

import (
	"fmt"
)

// TransportError represents errors in native messaging transport
type TransportError struct {
	Message string
	Cause   error
}

func (e *TransportError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("transport error: %s (caused by: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("transport error: %s", e.Message)
}

func (e *TransportError) Unwrap() error {
	return e.Cause
}

// NewTransportError creates a new transport error
func NewTransportError(message string, cause error) *TransportError {
	return &TransportError{
		Message: message,
		Cause:   cause,
	}
}

// TimeoutError represents timeout errors
type TimeoutError struct {
	Operation string
	Duration  string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout error: operation '%s' timed out after %s", e.Operation, e.Duration)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(operation, duration string) *TimeoutError {
	return &TimeoutError{
		Operation: operation,
		Duration:  duration,
	}
}

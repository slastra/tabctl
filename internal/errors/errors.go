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

// ValidationError represents validation errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
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

// NotImplementedError represents features not yet implemented
type NotImplementedError struct {
	Feature string
}

func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("not implemented: %s", e.Feature)
}

// NewNotImplementedError creates a new not implemented error
func NewNotImplementedError(feature string) *NotImplementedError {
	return &NotImplementedError{
		Feature: feature,
	}
}

// ConnectionError represents connection errors
type ConnectionError struct {
	Host    string
	Port    int
	Message string
	Cause   error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error to %s:%d: %s", e.Host, e.Port, e.Message)
}

func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// NewConnectionError creates a new connection error
func NewConnectionError(host string, port int, message string, cause error) *ConnectionError {
	return &ConnectionError{
		Host:    host,
		Port:    port,
		Message: message,
		Cause:   cause,
	}
}
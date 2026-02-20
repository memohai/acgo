package acgo

import (
	"errors"
	"fmt"
)

// APIError represents an error response from the Docker-compatible API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error (status %d): %s", e.StatusCode, e.Message)
}

// NotFoundError indicates the requested resource was not found.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Resource, e.ID)
}

// ConflictError indicates a resource state conflict.
type ConflictError struct {
	Resource string
	ID       string
	Message  string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s %q conflict: %s", e.Resource, e.ID, e.Message)
}

// IsNotFound returns true if the error is a NotFoundError or a 404 APIError.
func IsNotFound(err error) bool {
	var nf *NotFoundError
	if errors.As(err, &nf) {
		return true
	}
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == 404
	}
	return false
}

// IsConflict returns true if the error is a ConflictError or a 409 APIError.
func IsConflict(err error) bool {
	var cf *ConflictError
	if errors.As(err, &cf) {
		return true
	}
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == 409
	}
	return false
}

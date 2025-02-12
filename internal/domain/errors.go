package domain

import "fmt"

// ValidationError represents a validation for domain objects
type ValidationError struct {
	Field string
	Err   error
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s: %v", e.Field, e.Err)
}

// file: internal/domain/error_test.go

package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		err      error
		expected string
	}{
		{
			name:     "simple error",
			field:    "Username",
			err:      errors.New("must not be empty"),
			expected: "validation failed for field Username: must not be empty",
		},
		{
			name:     "empty field",
			field:    "",
			err:      errors.New("general validation error"),
			expected: "validation failed for field : general validation error",
		},
		{
			name:     "nil error",
			field:    "Price",
			err:      nil,
			expected: "validation failed for field Price: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valErr := ValidationError{
				Field: tt.field,
				Err:   tt.err,
			}
			result := valErr.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

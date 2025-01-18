package tradeutils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateRSI(t *testing.T) {
	tests := []struct {
		name     string
		history  []float64
		period   int
		expected float64
	}{
		{
			name:     "Insufficient data points",
			history:  []float64{100},
			period:   14,
			expected: 0,
		},
		{
			name:     "Flat line",
			history:  []float64{50, 50, 50, 50},
			period:   14,
			expected: 50,
		},
		{
			name:     "Pure upward movement",
			history:  []float64{10, 20, 30, 40, 50},
			period:   5,
			expected: 100,
		},
		{
			name:     "Pure downward movement",
			history:  []float64{50, 40, 30, 20, 10},
			period:   5,
			expected: 0,
		},
		{
			name:     "General case",
			history:  []float64{44, 47, 46, 48, 49, 50, 48, 47, 46, 77, 46},
			period:   5,
			expected: 48.4375, // Approximate expected value
		},
		{
			name:     "Insufficient period",
			history:  []float64{100, 105, 110},
			period:   5,
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRSI(tt.history, tt.period)
			assert.Equal(t, tt.expected, result)
		})
	}
}

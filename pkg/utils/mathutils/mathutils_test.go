package mathutils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPercDiff(t *testing.T) {
	// Test cases for PercDiff
	tests := []struct {
		name     string
		curr     float64
		prev     float64
		decimals int
		expected float64
	}{
		{"Normal positive values", 120, 100, 2, 20.00},
		{"Negative percent difference", 80, 100, 2, -20.00},
		{"No change", 100, 100, 2, 0.00},
		{"Divide by zero", 100, 0, 2, 0.00},
		{"No rounding", 123.125, 8, -1, 1439.0625},
		{"Rounding to 1 decimal", 123.125, 8, 1, 1439.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PercDiff(tt.curr, tt.prev, tt.decimals)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClamp(t *testing.T) {
	// Test cases for Clamp
	tests := []struct {
		name     string
		val      float64
		minVal   float64
		maxVal   float64
		expected float64
	}{
		{"Value within range", 5, 0, 10, 5},
		{"Value below range", -5, 0, 10, 0},
		{"Value above range", 15, 0, 10, 10},
		{"Value at lower bound", 0, 0, 10, 0},
		{"Value at upper bound", 10, 0, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Clamp(tt.val, tt.minVal, tt.maxVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRound(t *testing.T) {
	// Test cases for Round
	tests := []struct {
		name     string
		val      float64
		decimals int
		expected float64
	}{
		{"Round to 2 decimals", 123.456, 2, 123.46},
		{"Round to 1 decimal", 123.456, 1, 123.5},
		{"Round to 0 decimals", 123.456, 0, 123},
		{"Round negative value", -123.456, 2, -123.46},
		{"Round to -1 decimals", 123.456, -1, 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Round(tt.val, tt.decimals)
			assert.Equal(t, tt.expected, result)
		})
	}
}

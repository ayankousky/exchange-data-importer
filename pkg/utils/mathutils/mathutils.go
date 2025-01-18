package mathutils

import (
	"math"
)

// PercDiff calculates a percent difference between curr and prev,
// then rounds to 'decimals' decimals. e.g. decimals=2 => 12.34
func PercDiff(curr, prev float64, decimals int) float64 {
	// Guard against divide by zero
	if prev == 0 {
		return 0
	}
	diff := (curr - prev) / prev * 100
	if decimals == -1 {
		return diff
	}
	return Round(diff, decimals)
}

// Clamp caps 'val' within [minVal, maxVal].
func Clamp(val, minVal, maxVal float64) float64 {
	if val < minVal {
		return minVal
	} else if val > maxVal {
		return maxVal
	}
	return val
}

// Round rounds a float64 to the specified number of decimal places.
func Round(val float64, decimals int) float64 {
	p := math.Pow10(decimals)
	return math.Round(val*p) / p
}

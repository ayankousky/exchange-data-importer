package tradeutils

// CalculateRSI calculates the Relative Strength Index for a given period
func CalculateRSI(history []float64, period int) float64 {
	// Require at least 2 data points. If fewer, just return 0 or 50—your call.
	if len(history) < 2 {
		return 0
	}

	// We'll take the last `period` items in history
	if len(history) < period {
		// Not enough for the full period, so fallback or just use all
		period = len(history)
	}
	slice := history[len(history)-period:]

	var up float64
	var down float64

	// Accumulate up/down moves
	for i := 1; i < len(slice); i++ {
		current, previous := slice[i], slice[i-1]
		if current > previous {
			up += current - previous
		} else {
			down += previous - current
		}
	}

	// Handle edge cases
	if up == 0 && down == 0 {
		// Flat line => RSI is 50
		return 50
	}
	if up == 0 {
		// Pure downward movement
		return 0
	}
	if down == 0 {
		// Pure upward movement
		return 100
	}

	// Standard RSI formula
	return 100.0 - (100.0 / (1.0 + (up / down)))
}

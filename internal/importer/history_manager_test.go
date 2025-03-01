package importer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestInitHistoryWithErrors(t *testing.T) {
	ts := setupTest()
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func()
		wantError  bool
	}{
		{
			name: "should handle repository error",
			setupMocks: func() {
				ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			wantError: true,
		},
		{
			name: "should handle empty history",
			setupMocks: func() {
				ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
					return []domain.Tick{}, nil
				}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := ts.importer.initHistory(ctx)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTickerHistoryDataRace(t *testing.T) {
	ts := setupTest()

	const numGoroutines = 10
	const numOperations = 100

	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				ticker := &domain.Ticker{
					Symbol:    "BTCUSDT",
					Ask:       float64(50000 + routineID*j),
					Bid:       float64(49900 + routineID*j),
					CreatedAt: startTime.Add(time.Duration(j) * time.Second),
				}
				ts.importer.addTickerHistory(ticker)
			}
		}(i)
	}

	wg.Wait()

	history := ts.importer.tickerHistory.Get("BTCUSDT")
	assert.LessOrEqual(t, history.Len(), domain.MaxTickHistory)

	// Verify no duplicates for the same minute
	minutes := make(map[time.Time]bool)
	for i := 0; i < history.Len(); i++ {
		ticker := history.At(i)
		minute := ticker.CreatedAt.Truncate(time.Minute)
		assert.False(t, minutes[minute], "Found duplicate minute in history")
		minutes[minute] = true
	}
}

func TestTickerHistory(t *testing.T) {
	ts := setupTest()
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 1000; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000 + float64(i),
			Bid:       49950 + float64(i),
			CreatedAt: startDate.Add(time.Second * time.Duration(i)),
		}
		ts.importer.addTickerHistory(ticker)
	}

	tickerHistory := ts.importer.tickerHistory.Get("BTCUSDT")
	lastItem, _ := tickerHistory.Last()

	assert.Equal(t, 17, tickerHistory.Len(), "Only 1 ticker per minute should be stored")
	assert.Equal(t, 39, lastItem.CreatedAt.Second(), "Last inserted ticker should be at the 39th second")
	assert.Equal(t, 59, tickerHistory.At(tickerHistory.Len()-2).CreatedAt.Second(), "Last second inserted ticker should be at the 59th second")
	assert.Equal(t, 59, tickerHistory.At(tickerHistory.Len()-3).CreatedAt.Second(), "Last third inserted ticker should be at the 59th second")

	// Test history limit
	for i := 0; i < (60+10)*domain.MaxTickHistory; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000 + float64(i),
			Bid:       49950 + float64(i),
			CreatedAt: startDate.Add(time.Second * time.Duration(i)),
		}
		ts.importer.addTickerHistory(ticker)
	}
	assert.Equal(t, domain.MaxTickHistory, ts.importer.tickerHistory.Get("BTCUSDT").Len(), "Ticker history should be limited")
}

func TestCorruptedData(t *testing.T) {
	ts := setupTest()
	startDate := time.Now().Truncate(time.Hour)

	for i := 0; i < 1500; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000,
			Bid:       49950,
			CreatedAt: startDate.Add(time.Second),
		}
		ts.importer.addTickerHistory(ticker)
	}

	history := ts.importer.tickerHistory.Get("BTCUSDT")
	assert.Equal(t, 1, history.Len(), "Only 1 ticker per minute should be stored")
}

func TestIsSameMinute(t *testing.T) {
	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected bool
	}{
		{
			name:     "same minute",
			t1:       time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
			t2:       time.Date(2025, 1, 1, 10, 5, 45, 0, time.UTC),
			expected: true,
		},
		{
			name:     "different minute",
			t1:       time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
			t2:       time.Date(2025, 1, 1, 10, 6, 30, 0, time.UTC),
			expected: false,
		},
		{
			name:     "different hour",
			t1:       time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
			t2:       time.Date(2025, 1, 1, 11, 5, 30, 0, time.UTC),
			expected: false,
		},
		{
			name:     "same second but nanoseconds different",
			t1:       time.Date(2025, 1, 1, 10, 5, 30, 100, time.UTC),
			t2:       time.Date(2025, 1, 1, 10, 5, 30, 200, time.UTC),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSameMinute(tt.t1, tt.t2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateMinuteData(t *testing.T) {
	tests := []struct {
		name           string
		existingTicker *domain.Ticker
		newTicker      *domain.Ticker
		expectedMax    float64
		expectedMin    float64
	}{
		{
			name: "update with higher ask",
			existingTicker: &domain.Ticker{
				Symbol:    "BTCUSDT",
				Ask:       50000,
				Bid:       49900,
				Max:       50000,
				Min:       49000,
				CreatedAt: time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
			},
			newTicker: &domain.Ticker{
				Symbol:    "BTCUSDT",
				Ask:       51000,
				Bid:       50900,
				CreatedAt: time.Date(2025, 1, 1, 10, 5, 45, 0, time.UTC),
			},
			expectedMax: 51000,
			expectedMin: 49000,
		},
		{
			name: "update with lower ask",
			existingTicker: &domain.Ticker{
				Symbol:    "BTCUSDT",
				Ask:       50000,
				Bid:       49900,
				Max:       51000,
				Min:       50000,
				CreatedAt: time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
			},
			newTicker: &domain.Ticker{
				Symbol:    "BTCUSDT",
				Ask:       49500,
				Bid:       49400,
				CreatedAt: time.Date(2025, 1, 1, 10, 5, 45, 0, time.UTC),
			},
			expectedMax: 51000,
			expectedMin: 49500,
		},
		{
			name: "update with same values",
			existingTicker: &domain.Ticker{
				Symbol:    "BTCUSDT",
				Ask:       50000,
				Bid:       49900,
				Max:       51000,
				Min:       49000,
				CreatedAt: time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
			},
			newTicker: &domain.Ticker{
				Symbol:    "BTCUSDT",
				Ask:       50000,
				Bid:       49900,
				CreatedAt: time.Date(2025, 1, 1, 10, 5, 45, 0, time.UTC),
			},
			expectedMax: 51000,
			expectedMin: 49000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateMinuteData(tt.existingTicker, tt.newTicker)

			// Check that existing ticker was updated
			assert.Equal(t, tt.newTicker.Ask, tt.existingTicker.Ask)
			assert.Equal(t, tt.newTicker.Bid, tt.existingTicker.Bid)
			assert.Equal(t, tt.newTicker.CreatedAt, tt.existingTicker.CreatedAt)
			assert.Equal(t, tt.expectedMax, tt.existingTicker.Max)
			assert.Equal(t, tt.expectedMin, tt.existingTicker.Min)

			// Check that new ticker has the correct information mirrored
			assert.Equal(t, tt.expectedMax, tt.newTicker.Max)
			assert.Equal(t, tt.expectedMin, tt.newTicker.Min)
		})
	}
}

func TestGetOrCreateBuffer(t *testing.T) {
	thm := newTickerHistoryMap()

	// Test getting a buffer that doesn't exist yet
	buffer1 := thm.getOrCreateBuffer("BTCUSDT")
	assert.NotNil(t, buffer1)
	assert.Equal(t, 0, buffer1.Len())
	assert.Equal(t, domain.MaxTickHistory, buffer1.Cap())

	// Add a ticker to the buffer
	ticker := &domain.Ticker{
		Symbol:    "BTCUSDT",
		Ask:       50000,
		Bid:       49900,
		CreatedAt: time.Now(),
	}
	buffer1.Push(ticker)

	// Get the same buffer again and verify it has the item
	buffer2 := thm.getOrCreateBuffer("BTCUSDT")
	assert.Equal(t, 1, buffer2.Len())

	// Verify it's the same buffer
	assert.Same(t, buffer1, buffer2)

	// Check a different ticker name creates a new buffer
	buffer3 := thm.getOrCreateBuffer("ETHUSDT")
	assert.NotNil(t, buffer3)
	assert.Equal(t, 0, buffer3.Len())
	assert.NotSame(t, buffer1, buffer3)
}

func TestTickerHistoryMap_UpdateTicker(t *testing.T) {
	thm := newTickerHistoryMap()

	// Test with nil ticker
	thm.UpdateTicker(nil)
	assert.Equal(t, 0, len(thm.data))

	// Test with first ticker
	ticker1 := &domain.Ticker{
		Symbol:    "BTCUSDT",
		Ask:       50000,
		Bid:       49900,
		CreatedAt: time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
	}
	thm.UpdateTicker(ticker1)

	// Verify ticker was added
	assert.Equal(t, 1, len(thm.data))
	buffer := thm.Get("BTCUSDT")
	assert.Equal(t, 1, buffer.Len())
	firstTicker, exists := buffer.Last()
	assert.True(t, exists)
	assert.Equal(t, ticker1.Ask, firstTicker.Ask)
	assert.Equal(t, ticker1.Ask, firstTicker.Max)
	assert.Equal(t, ticker1.Ask, firstTicker.Min)

	// Test with newer ticker in same minute
	ticker2 := &domain.Ticker{
		Symbol:    "BTCUSDT",
		Ask:       51000,
		Bid:       50900,
		CreatedAt: time.Date(2025, 1, 1, 10, 5, 45, 0, time.UTC),
	}
	thm.UpdateTicker(ticker2)

	// Verify buffer still has only one item but updated
	assert.Equal(t, 1, buffer.Len())

	lastTicker, _ := buffer.Last()
	assert.Equal(t, ticker2.Ask, lastTicker.Ask)
	assert.Equal(t, ticker2.Bid, lastTicker.Bid)
	assert.Equal(t, ticker2.CreatedAt, lastTicker.CreatedAt)
	assert.Equal(t, ticker2.Ask, lastTicker.Max)
	assert.Equal(t, ticker1.Ask, ticker2.Ask)
	assert.Equal(t, ticker1.Bid, ticker2.Bid)

	// Test with older ticker
	ticker3 := &domain.Ticker{
		Symbol:    "BTCUSDT",
		Ask:       52000,
		Bid:       51900,
		CreatedAt: time.Date(2025, 1, 1, 10, 4, 30, 0, time.UTC), // Older timestamp
	}
	thm.UpdateTicker(ticker3)

	// Verify older ticker is ignored
	assert.Equal(t, 1, buffer.Len())
	lastTicker, _ = buffer.Last()
	assert.Equal(t, ticker2.CreatedAt, lastTicker.CreatedAt)

	// Test with ticker in new minute
	ticker4 := &domain.Ticker{
		Symbol:    "BTCUSDT",
		Ask:       53000,
		Bid:       52900,
		CreatedAt: time.Date(2025, 1, 1, 10, 6, 15, 0, time.UTC), // New minute
	}
	thm.UpdateTicker(ticker4)

	// Verify new minute creates new entry
	assert.Equal(t, 2, buffer.Len())
	newTicker, _ := buffer.Last()
	assert.Equal(t, ticker4.Ask, newTicker.Ask)
	assert.Equal(t, ticker4.Ask, newTicker.Max)
	assert.Equal(t, ticker4.Ask, newTicker.Min)
}

func TestGetLastTick(t *testing.T) {
	ts := setupTest()

	// Should return error when no history
	_, err := ts.importer.getLastTick()
	assert.Error(t, err)

	// Add tick to history
	tick1 := &domain.Tick{
		StartAt: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
		Data: map[domain.TickerName]*domain.Ticker{
			"BTCUSDT": {
				Symbol: "BTCUSDT",
				Ask:    50000,
				Bid:    49900,
			},
		},
	}
	ts.importer.tickHistory.Push(tick1)

	// Should return tick now
	lastTick, err := ts.importer.getLastTick()
	assert.NoError(t, err)
	assert.Equal(t, tick1, lastTick)

	// Add newer tick
	tick2 := &domain.Tick{
		StartAt: time.Date(2025, 1, 1, 10, 0, 1, 0, time.UTC),
		Data: map[domain.TickerName]*domain.Ticker{
			"BTCUSDT": {
				Symbol: "BTCUSDT",
				Ask:    51000,
				Bid:    50900,
			},
		},
	}
	ts.importer.tickHistory.Push(tick2)

	// Should return newest tick
	lastTick, err = ts.importer.getLastTick()
	assert.NoError(t, err)
	assert.Equal(t, tick2, lastTick)
}

func TestAddTickHistory(t *testing.T) {
	ts := setupTest()

	// Should handle nil tick
	ts.importer.addTickHistory(nil)
	assert.Equal(t, 0, ts.importer.tickHistory.Len())

	// Add first tick
	tick1 := &domain.Tick{
		StartAt: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
	}
	ts.importer.addTickHistory(tick1)
	assert.Equal(t, 1, ts.importer.tickHistory.Len())

	// Add newer tick
	tick2 := &domain.Tick{
		StartAt: time.Date(2025, 1, 1, 10, 0, 1, 0, time.UTC),
	}
	ts.importer.addTickHistory(tick2)
	assert.Equal(t, 2, ts.importer.tickHistory.Len())

	// Add older tick (should be ignored)
	tick3 := &domain.Tick{
		StartAt: time.Date(2025, 1, 1, 9, 59, 59, 0, time.UTC),
	}
	ts.importer.addTickHistory(tick3)
	assert.Equal(t, 2, ts.importer.tickHistory.Len())

	// Last tick should still be tick2
	lastTick, exists := ts.importer.tickHistory.Last()
	assert.True(t, exists)
	assert.Equal(t, tick2, lastTick)
}

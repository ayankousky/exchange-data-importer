package importer

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/pkg/utils"
)

// tickHistory represents a historical record of ticks
type tickHistory struct {
	buffer *utils.RingBuffer[*domain.Tick]
}

func newTickHistory(size int) *tickHistory {
	return &tickHistory{
		buffer: utils.NewRingBuffer[*domain.Tick](size),
	}
}

func (th *tickHistory) Push(tick *domain.Tick) {
	th.buffer.Push(tick)
}

func (th *tickHistory) Last() (*domain.Tick, bool) {
	return th.buffer.Last()
}

func (th *tickHistory) Len() int {
	return th.buffer.Len()
}

func (th *tickHistory) At(index int) *domain.Tick {
	return th.buffer.At(index)
}

// tickerHistoryMap represents a thread-safe map of ticker histories
type tickerHistoryMap struct {
	data map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]
	mu   sync.RWMutex
}

func newTickerHistoryMap() *tickerHistoryMap {
	return &tickerHistoryMap{
		data: make(map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]),
	}
}

// Get returns the history buffer for a given ticker name
// If no buffer exists, it creates one
func (thm *tickerHistoryMap) Get(name domain.TickerName) *utils.RingBuffer[*domain.Ticker] {
	thm.mu.RLock()
	history, ok := thm.data[name]
	thm.mu.RUnlock()

	if !ok {
		thm.mu.Lock()
		// Double-check after acquiring write lock
		if history, ok = thm.data[name]; !ok {
			history = utils.NewRingBuffer[*domain.Ticker](domain.MaxTickHistory)
			thm.data[name] = history
		}
		thm.mu.Unlock()
	}
	return history
}

// UpdateTicker atomically updates or adds a new ticker to the history
func (thm *tickerHistoryMap) UpdateTicker(ticker *domain.Ticker) {
	if ticker == nil {
		return
	}

	thm.mu.Lock()
	defer thm.mu.Unlock()

	history := thm.getOrCreateBuffer(ticker.Symbol)
	lastTicker, exists := history.Last()

	// Skip older data
	if exists && lastTicker.CreatedAt.After(ticker.CreatedAt) {
		return
	}

	// Handle minute-based data organization
	if !exists || !isSameMinute(lastTicker.CreatedAt, ticker.CreatedAt) {
		// New minute or no previous data - create new entry
		ticker.Max = ticker.Ask
		ticker.Min = ticker.Ask
		history.Push(ticker)
		return
	}

	// Update existing minute data
	updateMinuteData(lastTicker, ticker)
}

// getOrCreateBuffer returns existing buffer or creates a new one (must be called under lock)
func (thm *tickerHistoryMap) getOrCreateBuffer(name domain.TickerName) *utils.RingBuffer[*domain.Ticker] {
	history, ok := thm.data[name]
	if !ok {
		history = utils.NewRingBuffer[*domain.Ticker](domain.MaxTickHistory)
		thm.data[name] = history
	}
	return history
}

// isSameMinute checks if two timestamps are in the same minute
func isSameMinute(t1, t2 time.Time) bool {
	return t1.Truncate(time.Minute).Equal(t2.Truncate(time.Minute))
}

// updateMinuteData updates the ticker data for the current minute
func updateMinuteData(existingTicker, newTicker *domain.Ticker) {
	existingTicker.Max = math.Max(existingTicker.Max, newTicker.Ask)
	existingTicker.Min = math.Min(existingTicker.Min, newTicker.Ask)
	existingTicker.Ask = newTicker.Ask
	existingTicker.Bid = newTicker.Bid
	existingTicker.CreatedAt = newTicker.CreatedAt

	// Mirror changes to the new ticker
	newTicker.Max = existingTicker.Max
	newTicker.Min = existingTicker.Min
}

// initHistory loads old data from repositories and populates ring buffers
func (i *Importer) initHistory(ctx context.Context) error {
	history, err := i.tickRepository.GetHistorySince(ctx, time.Now().Add(-domain.MaxTickHistory*time.Minute))
	if err != nil {
		return fmt.Errorf("GetHistorySince failed: %w", err)
	}

	for _, tick := range history {
		i.addTickHistory(&tick)
		for _, ticker := range tick.Data {
			i.addTickerHistory(ticker)
		}
	}

	return nil
}

func (i *Importer) addTickHistory(tick *domain.Tick) {
	if tick == nil {
		return
	}

	lastTick, exists := i.tickHistory.Last()
	if exists && lastTick.StartAt.After(tick.StartAt) {
		return
	}

	i.tickHistory.Push(tick)
}

// addTickerHistory updates the ring buffer for a particular ticker - 1 item per 1 minute
func (i *Importer) addTickerHistory(ticker *domain.Ticker) {
	if ticker == nil {
		return
	}

	i.tickerHistory.UpdateTicker(ticker)
}

func (i *Importer) getLastTick() (*domain.Tick, error) {
	lastTick, exists := i.tickHistory.Last()
	if !exists {
		return nil, fmt.Errorf("no tick history found")
	}
	return lastTick, nil
}

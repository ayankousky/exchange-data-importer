package importer

import (
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

func (thm *tickerHistoryMap) Get(name domain.TickerName) *utils.RingBuffer[*domain.Ticker] {
	thm.mu.RLock()
	history, ok := thm.data[name]
	thm.mu.RUnlock()

	if !ok {
		thm.mu.Lock()
		// Double-check after acquiring write lock
		history, ok = thm.data[name]
		if !ok {
			history = utils.NewRingBuffer[*domain.Ticker](domain.MaxTickHistory)
			thm.data[name] = history
		}
		thm.mu.Unlock()
	}
	return history
}

// UpdateTicker atomically updates or adds a new ticker to the history
func (thm *tickerHistoryMap) UpdateTicker(ticker *domain.Ticker) {
	thm.mu.Lock()
	defer thm.mu.Unlock()

	history := thm.getOrCreateBuffer(ticker.Symbol)
	lastTickerData, exists := history.Last()
	if exists && lastTickerData.CreatedAt.After(ticker.CreatedAt) {
		// Skip older data
		return
	}

	if !exists || !lastTickerData.CreatedAt.Truncate(time.Minute).Equal(ticker.CreatedAt.Truncate(time.Minute)) {
		// New minute or no previous data - create new entry
		ticker.Max = ticker.Ask
		ticker.Min = ticker.Ask
		history.Push(ticker)
		return
	}

	if lastTickerData.CreatedAt.After(ticker.CreatedAt) {
		// Skip older data
		return
	}

	// Update existing minute data
	updateMinuteData(lastTickerData, ticker)
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

// updateMinuteData updates the ticker data for the current minute
func updateMinuteData(existingTicker, newTicker *domain.Ticker) {
	existingTicker.Max = math.Max(existingTicker.Max, newTicker.Ask)
	existingTicker.Min = math.Min(existingTicker.Min, newTicker.Ask)
	existingTicker.Ask = newTicker.Ask
	existingTicker.Bid = newTicker.Bid
	existingTicker.CreatedAt = newTicker.CreatedAt

	// Mirror changes to the newTicker ticker
	newTicker.Max = existingTicker.Max
	newTicker.Min = existingTicker.Min
}

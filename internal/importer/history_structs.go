package importer

import (
	"sync"

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

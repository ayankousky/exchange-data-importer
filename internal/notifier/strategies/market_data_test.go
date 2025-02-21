package strategies

import (
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
	"github.com/stretchr/testify/assert"
)

func TestMarketDataStrategy_Format(t *testing.T) {
	tests := []struct {
		name          string
		input         *domain.Tick
		wantEvents    bool
		wantEventsCnt int
	}{
		{
			name: "should format market data correctly",
			input: &domain.Tick{
				CreatedAt: time.Now(),
				Data: map[domain.TickerName]*domain.Ticker{
					"BTCUSDT": {
						Symbol:   "BTCUSDT",
						Ask:      45000.00,
						Bid:      44990.00,
						Change1m: 0.5,
					},
					"ETHUSDT": {
						Symbol:   "ETHUSDT",
						Ask:      3000.00,
						Bid:      2999.00,
						Change1m: 0.3,
					},
				},
			},
			wantEvents:    true,
			wantEventsCnt: 2,
		},
		{
			name:       "should handle nil input",
			input:      nil,
			wantEvents: false,
		},
		{
			name: "should handle empty data map",
			input: &domain.Tick{
				CreatedAt: time.Now(),
				Data:      map[domain.TickerName]*domain.Ticker{},
			},
			wantEvents:    true,
			wantEventsCnt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &MarketDataStrategy{}
			events := strategy.Format(tt.input)
			if !tt.wantEvents {
				assert.Empty(t, events)
				return
			}

			assert.Len(t, events, tt.wantEventsCnt)
			for _, event := range events {
				assert.Equal(t, string(notifier.MarketDataTopic), event.EventType)
				assert.NotNil(t, event.Data)
			}
		})
	}
}

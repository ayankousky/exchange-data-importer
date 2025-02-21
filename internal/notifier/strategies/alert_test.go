package strategies

import (
	"testing"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
	"github.com/stretchr/testify/assert"
)

func TestAlertStrategy_Format(t *testing.T) {
	tests := []struct {
		name       string
		thresholds AlertStrategyThresholds
		input      *domain.Tick
		wantEvents bool
		wantData   string
	}{
		{
			name: "should generate alert for significant market move",
			thresholds: AlertStrategyThresholds{
				AvgPrice1mChange:    1.0,
				AvgPrice20mChange:   1000,
				TickerPrice1mChange: 1000,
			},
			input: &domain.Tick{
				Avg: domain.TickAvg{
					Change1m:     1.5,
					TickersCount: 10,
				},
			},
			wantEvents: true,
		},
		{
			name: "should not generate alert for normal market",
			thresholds: AlertStrategyThresholds{
				AvgPrice1mChange:    2.0,
				AvgPrice20mChange:   1000,
				TickerPrice1mChange: 1000,
			},
			input: &domain.Tick{
				Avg: domain.TickAvg{
					Change1m: 0.5,
				},
			},
			wantEvents: false,
		},
		{
			name: "should handle nil input",
			thresholds: AlertStrategyThresholds{
				AvgPrice1mChange:    1.0,
				AvgPrice20mChange:   1000,
				TickerPrice1mChange: 1000,
			},
			input:      nil,
			wantEvents: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewAlertStrategy(tt.thresholds)
			events := strategy.Format(tt.input)

			if !tt.wantEvents {
				assert.Empty(t, events)
				return
			}

			assert.NotEmpty(t, events)
			assert.Equal(t, string(notifier.AlertTopic), events[0].EventType)
		})
	}
}

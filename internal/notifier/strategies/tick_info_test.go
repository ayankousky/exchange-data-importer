package strategies

import (
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
	"github.com/stretchr/testify/assert"
)

func TestTickInfoStrategy_Format(t *testing.T) {
	tests := []struct {
		name       string
		input      *domain.Tick
		wantEvents bool
	}{
		{
			name: "should format tick info correctly",
			input: &domain.Tick{
				CreatedAt: time.Now(),
				Avg: domain.TickAvg{
					TickersCount: 10,
					Change1m:     1.5,
					Change20m:    2.5,
				},
				AvgBuy10: 100.50,
				LL5:      5,
				LL60:     60,
				SL2:      2,
				SL10:     10,
			},
			wantEvents: true,
		},
		{
			name:       "should handle nil input",
			input:      nil,
			wantEvents: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &TickInfoStrategy{}
			events := strategy.Format(tt.input)

			if !tt.wantEvents {
				assert.Empty(t, events)
				return
			}

			assert.NotEmpty(t, events)
			assert.Equal(t, string(notifier.TickInfoTopic), events[0].EventType)
			assert.NotEmpty(t, events[0].Data)
		})
	}
}

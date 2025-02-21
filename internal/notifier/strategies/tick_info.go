package strategies

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
)

const (
	headerFormat = "%-8s | %4s | %8s | %8s | %8s | %6s | %6s | %6s | %6s\n"
	dataFormat   = "%-8s | %4d | %8.2f | %8.2f | %8.2f | %6d | %6d | %6d | %6d\n"
)

// TickInfoStrategy creates common tick information in the stdout
type TickInfoStrategy struct {
	printCount atomic.Int64
}

// Format formats the tick data into a human-readable format
func (s *TickInfoStrategy) Format(data any) []notify.Event {
	tick, ok := data.(*domain.Tick)
	if !ok {
		return nil
	}

	if tick == nil {
		return nil
	}

	var output strings.Builder
	count := s.printCount.Add(1)

	if count%10 == 0 {
		fmt.Fprintf(&output, headerFormat,
			"TIME",
			"MKTS",
			"1M CHG%",
			"20M CHG%",
			"AVG BUY",
			"LL5",
			"LL60",
			"SL2",
			"SL10",
		)
	}

	fmt.Fprintf(&output, dataFormat,
		tick.CreatedAt.Format("15:04:05"),
		tick.Avg.TickersCount,
		tick.Avg.Change1m,
		tick.Avg.Change20m,
		tick.AvgBuy10,
		tick.LL5,
		tick.LL60,
		tick.SL2,
		tick.SL10,
	)

	return []notify.Event{{
		Time:      time.Now(),
		EventType: string(notifier.TickInfoTopic),
		Data:      output.String(),
	}}
}

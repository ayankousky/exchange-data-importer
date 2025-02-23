package notifier

import (
	"context"
	"testing"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	notifyMocks "github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNotifier_Subscribe(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		client   notify.Client
		strategy notify.Strategy
		wantLen  int
	}{
		{
			name:     "subscribe to invalid topic",
			topic:    "INVALID_TOPIC",
			client:   &notifyMocks.ClientMock{},
			strategy: &notifyMocks.StrategyMock{},
			wantLen:  0,
		},
		{
			name:     "subscribe with nil client",
			topic:    string(MarketDataTopic),
			client:   nil,
			strategy: &notifyMocks.StrategyMock{},
			wantLen:  0,
		},
		{
			name:     "subscribe with nil strategy",
			topic:    string(MarketDataTopic),
			client:   &notifyMocks.ClientMock{},
			strategy: nil,
			wantLen:  0,
		},
		{
			name:     "subscribe to valid topic",
			topic:    string(MarketDataTopic),
			client:   &notifyMocks.ClientMock{},
			strategy: &notifyMocks.StrategyMock{},
			wantLen:  1,
		},
		{
			name:     "subscribe to valid topic",
			topic:    string(TickInfoTopic),
			client:   &notifyMocks.ClientMock{},
			strategy: &notifyMocks.StrategyMock{},
			wantLen:  1,
		},
		{
			name:     "multiple subscriptions to same topic",
			topic:    string(AlertTopic),
			client:   &notifyMocks.ClientMock{},
			strategy: &notifyMocks.StrategyMock{},
			wantLen:  2, // Adding to existing subscription
		},
	}

	for _, tt := range tests {
		n := New(zap.NewNop())

		// Pre-subscribe one handler to AlertTopic for testing multiple subscriptions
		n.Subscribe(string(AlertTopic), &notifyMocks.ClientMock{}, &notifyMocks.StrategyMock{})

		t.Run(tt.name, func(t *testing.T) {
			n.Subscribe(tt.topic, tt.client, tt.strategy)
			assert.Len(t, n.handlers[Topic(tt.topic)], tt.wantLen)
		})
	}
}

func TestNotifier_Notify(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		setup         func(*Notifier, *notifyMocks.ClientMock, *notifyMocks.StrategyMock)
		input         any
		expectEvents  bool
		expectedCalls int
	}{
		{
			name: "notify subscribers with valid data",
			setup: func(n *Notifier, mc *notifyMocks.ClientMock, ms *notifyMocks.StrategyMock) {
				event := notify.Event{
					EventType: string(MarketDataTopic),
					Data:      "test data",
				}
				ms.FormatFunc = func(data any) []notify.Event {
					return []notify.Event{event}
				}
				mc.SendFunc = func(ctx context.Context, event notify.Event) error {
					return nil
				}
				n.Subscribe(string(MarketDataTopic), mc, ms)
			},
			input: &domain.Tick{
				Avg: domain.TickAvg{
					TickersCount: 1,
				},
			},
			expectEvents:  true,
			expectedCalls: 1,
		},
		{
			name: "handle nil data",
			setup: func(n *Notifier, mc *notifyMocks.ClientMock, ms *notifyMocks.StrategyMock) {
				ms.FormatFunc = func(data any) []notify.Event {
					return nil
				}
				n.Subscribe(string(MarketDataTopic), mc, ms)
			},
			input:         nil,
			expectEvents:  false,
			expectedCalls: 0,
		},
		{
			name: "handle strategy returning no events",
			setup: func(n *Notifier, mc *notifyMocks.ClientMock, ms *notifyMocks.StrategyMock) {
				ms.FormatFunc = func(data any) []notify.Event {
					return []notify.Event{}
				}
				n.Subscribe(string(MarketDataTopic), mc, ms)
			},
			input:         &domain.Tick{},
			expectEvents:  false,
			expectedCalls: 0,
		},
		{
			name: "notify multiple subscribers",
			setup: func(n *Notifier, mc *notifyMocks.ClientMock, ms *notifyMocks.StrategyMock) {
				event := notify.Event{
					EventType: string(MarketDataTopic),
					Data:      "test data",
				}
				ms.FormatFunc = func(data any) []notify.Event {
					return []notify.Event{event}
				}
				mc.SendFunc = func(ctx context.Context, event notify.Event) error {
					return nil
				}

				// Subscribe twice to same topic
				n.Subscribe(string(MarketDataTopic), mc, ms)
				n.Subscribe(string(MarketDataTopic), mc, ms)
			},
			input:         &domain.Tick{},
			expectEvents:  true,
			expectedCalls: 2,
		},
		{
			name: "handle client send error",
			setup: func(n *Notifier, mc *notifyMocks.ClientMock, ms *notifyMocks.StrategyMock) {
				event := notify.Event{
					EventType: string(MarketDataTopic),
					Data:      "test data",
				}
				ms.FormatFunc = func(data any) []notify.Event {
					return []notify.Event{event}
				}
				mc.SendFunc = func(ctx context.Context, event notify.Event) error {
					return assert.AnError
				}
				n.Subscribe(string(MarketDataTopic), mc, ms)
			},
			input:         &domain.Tick{},
			expectEvents:  true,
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := New(zap.NewNop())
			mockClient := &notifyMocks.ClientMock{}
			mockStrategy := &notifyMocks.StrategyMock{}

			tt.setup(n, mockClient, mockStrategy)
			n.Notify(ctx, tt.input)

			if tt.expectEvents {
				assert.Equal(t, tt.expectedCalls, len(mockStrategy.FormatCalls()))
				assert.Equal(t, tt.expectedCalls, len(mockClient.SendCalls()))
			} else {
				assert.Empty(t, mockClient.SendCalls())
			}
		})
	}
}

func TestTopic_Validate(t *testing.T) {
	tests := []struct {
		name    string
		topic   Topic
		wantErr bool
	}{
		{
			name:    "valid market data topic",
			topic:   MarketDataTopic,
			wantErr: false,
		},
		{
			name:    "valid alert topic",
			topic:   AlertTopic,
			wantErr: false,
		},
		{
			name:    "valid tick info topic",
			topic:   TickInfoTopic,
			wantErr: false,
		},
		{
			name:    "invalid topic",
			topic:   "INVALID_TOPIC",
			wantErr: true,
		},
		{
			name:    "empty topic",
			topic:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.topic.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid topic")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

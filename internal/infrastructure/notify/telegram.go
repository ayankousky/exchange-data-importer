package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// MinIntervalSeconds is the default minimum interval between notifications
const MinIntervalSeconds = 10

// TelegramNotifier is a Telegram-based implementation of domain.NotificationService
type TelegramNotifier struct {
	botToken string
	chatID   string
	baseURL  string

	interval     time.Duration
	lastSentTime time.Time
	mu           sync.Mutex
}

// NewTelegramNotifier creates a new TelegramNotifier
func NewTelegramNotifier(botToken, chatID string, intervalSeconds int) (*TelegramNotifier, error) {
	if botToken == "" || chatID == "" {
		return nil, fmt.Errorf("bot token and chat ID are required")
	}

	if intervalSeconds <= 0 {
		intervalSeconds = MinIntervalSeconds
	}

	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		baseURL:  "https://api.telegram.org/bot",

		interval: time.Duration(intervalSeconds) * time.Second,
	}, nil
}

// Send sends a notification to a Telegram chat
func (t *TelegramNotifier) Send(ctx context.Context, event Event) error {
	t.mu.Lock()
	now := time.Now()
	if now.Sub(t.lastSentTime) < t.interval {
		t.mu.Unlock()
		return nil
	}
	t.lastSentTime = now
	t.mu.Unlock()

	message, ok := event.Data.(string)
	if !ok {
		return fmt.Errorf("telegram notifier expects string data, got %T", event.Data)
	}

	apiURL := fmt.Sprintf("%s%s/sendMessage", t.baseURL, t.botToken)
	params := url.Values{}
	params.Add("chat_id", t.chatID)
	params.Add("text", message)
	params.Add("parse_mode", "HTML")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating telegram request: %w", err)
	}
	req.URL.RawQuery = params.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

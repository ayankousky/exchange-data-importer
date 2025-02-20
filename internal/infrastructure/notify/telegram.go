package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// TelegramNotifier is a Telegram-based implementation of domain.NotificationService
type TelegramNotifier struct {
	botToken string
	chatID   string
	baseURL  string
}

// NewTelegramNotifier creates a new TelegramNotifier
func NewTelegramNotifier(botToken, chatID string) (*TelegramNotifier, error) {
	if botToken == "" || chatID == "" {
		return nil, fmt.Errorf("bot token and chat ID are required")
	}
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		baseURL:  "https://api.telegram.org/bot",
	}, nil
}

// Send sends a notification to a Telegram chat
func (t *TelegramNotifier) Send(ctx context.Context, event Event) error {
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

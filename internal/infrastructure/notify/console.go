package notify

import (
	"context"
	"fmt"
)

// ConsoleNotifier sends notifications to stdout
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a new console notifier
func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

// Send sends a notification to stdout
func (n *ConsoleNotifier) Send(_ context.Context, event Event) error {
	str, ok := event.Data.(string)
	if !ok {
		return fmt.Errorf("console notifier expects string data, got %T", event.Data)
	}
	fmt.Print(str)
	return nil
}

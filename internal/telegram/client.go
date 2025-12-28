// Package telegram provides a client for sending messages via the Telegram Bot API.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is a wrapper for the Telegram Bot API.
type Client struct {
	token  string
	chatID string
}

// NewClient creates a new Telegram client.
func NewClient(token, chatID string) *Client {
	return &Client{token: token, chatID: chatID}
}

// SendMessage sends a text message to the configured Telegram chat.
func (c *Client) SendMessage(text string) error {
	if c.token == "" || c.chatID == "" {
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)
	payload := map[string]string{
		"chat_id": c.chatID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body)) // #nosec G107
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// Package email provides a client for sending email notifications via SMTP.
package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Client is a wrapper for sending email notifications.
type Client struct {
	smtpHost   string
	smtpPort   string
	from       string
	to         []string
	username   string
	password   string
}

// NewClient creates a new email client.
func NewClient(smtpHost, smtpPort, from, username, password string, to []string) *Client {
	return &Client{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		from:     from,
		to:       to,
		username: username,
		password: password,
	}
}

// SendMessage sends an email notification.
func (c *Client) SendMessage(subject, body string) error {
	if c.smtpHost == "" || len(c.to) == 0 {
		return nil
	}

	addr := fmt.Sprintf("%s:%s", c.smtpHost, c.smtpPort)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		c.from,
		strings.Join(c.to, ", "),
		subject,
		body,
	)

	auth := smtp.PlainAuth("", c.username, c.password, c.smtpHost)
	if err := smtp.SendMail(addr, auth, c.from, c.to, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

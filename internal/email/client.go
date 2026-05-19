// Package email provides a client for sending email notifications via SMTP.
package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// Client is a wrapper for sending email notifications.
type Client struct {
	smtpHost string
	smtpPort string
	from     string
	to       []string
	username string
	password string
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

	addr := net.JoinHostPort(c.smtpHost, c.smtpPort)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, c.smtpHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS via STARTTLS if the server supports it (same as nodemailer secure:false)
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: c.smtpHost}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if c.username != "" {
		auth := smtp.PlainAuth("", c.username, c.password, c.smtpHost)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err = client.Mail(c.from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	for _, to := range c.to {
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("RCPT TO %s: %w", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}

	msg := "From: " + c.from + "\r\n" +
		"To: " + strings.Join(c.to, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
		body

	if _, err = fmt.Fprint(w, msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("end data: %w", err)
	}

	return client.Quit()
}

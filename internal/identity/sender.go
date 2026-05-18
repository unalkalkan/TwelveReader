package identity

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Sender mode constants.
const (
	SenderModeLog  = "log"   // Dev only: log full email body including token to stdout
	SenderModeSMTP = "smtp"  // Real SMTP delivery
	SenderModeNone = "none"  // Fail-closed: reject auth requests
)

// NoopSender is a fail-closed email sender. Always returns an error on SendMagicLink.
type NoopSender struct{}

func (n *NoopSender) SendMagicLink(to, subject, body string) error {
	return fmt.Errorf("email sending is not configured: auth requests are rejected in %s mode (configure sender_mode=smtp with SMTP credentials to enable email delivery)", SenderModeNone)
}

// SMTPSender sends magic link emails via a real SMTP server.
type SMTPSender struct {
	from    string
	host    string
	port    int
	username string
	password string
	useTLS  bool
}

func (s *SMTPSender) SendMagicLink(to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s",
		s.from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	if s.useTLS {
		tlsConfig := &tls.Config{
			ServerName: s.host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("SMTP TLS dial: %w", err)
		}
		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return fmt.Errorf("SMTP client: %w", err)
		}
		defer client.Close()

		if s.username != "" && s.password != "" {
			auth := smtp.PlainAuth("", s.username, s.password, s.host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP auth: %w", err)
			}
		}

		if err := client.Mail(s.from); err != nil {
			return fmt.Errorf("SMTP mail from: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP rcpt to: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("SMTP data: %w", err)
		}
		if _, err := w.Write([]byte(msg)); err != nil {
			return fmt.Errorf("SMTP write: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("SMTP close: %w", err)
		}
	} else {
		var auth smtp.Auth
		if s.username != "" && s.password != "" {
			auth = smtp.PlainAuth("", s.username, s.password, s.host)
		}
		err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
		if err != nil {
			return fmt.Errorf("SMTP send: %w", err)
		}
	}
	return nil
}

// NewEmailSender creates an EmailSender based on the auth config and environment.
//
// SenderMode resolution:
//   - If cfg.SenderMode is explicitly set, use it (unless "log" in staging/production -> error at startup).
//   - Otherwise, default to "log" for local/dev environments, or "none" for staging/production.
//
// Returns a validation error if:
//   - SenderMode is "smtp" but SMTP config is incomplete.
//   - SenderMode is "log" in staging or production environment (security guard).
func NewEmailSender(cfg *types.AuthConfig, environment string) (EmailSender, error) {
	senderMode := strings.ToLower(strings.TrimSpace(cfg.SenderMode))

	// Apply defaults if not explicitly set
	if senderMode == "" {
		switch environment {
		case "local", "dev":
			senderMode = SenderModeLog
		case "staging", "production":
			senderMode = SenderModeNone
		default:
			senderMode = SenderModeLog
		}
	}

	switch senderMode {
	case SenderModeLog:
		// Guard: refuse to log tokens in staging/production unless explicitly configured.
		// If user explicitly set sender_mode=log in YAML/env for prod, that's their choice but we warn.
		if environment == "staging" || environment == "production" {
			// Only block if it was NOT explicitly set (auto-default).
			// But since we already applied defaults above, check: if cfg.SenderMode was empty, block.
			// Actually, re-check: if original cfg.SenderMode was empty, this is a default and we should
			// have defaulted to "none" above for prod/staging. So we only reach here if explicitly set.
			// For safety, always error in staging/production for "log" mode.
			return nil, fmt.Errorf("sender_mode %q is not allowed in environment %q (tokens would be logged to stdout); use %q with SMTP configuration", SenderModeLog, environment, SenderModeSMTP)
		}
		return &LogEmailSender{DevMode: true}, nil

	case SenderModeSMTP:
		if cfg.SMTP.Host == "" || cfg.SMTP.Port == 0 {
			return nil, fmt.Errorf("sender_mode %q requires SMTP configuration (smtp.host and smtp.port must be set)", SenderModeSMTP)
		}
		return &SMTPSender{
			from:     cfg.SenderFrom,
			host:     cfg.SMTP.Host,
			port:     cfg.SMTP.Port,
			username: cfg.SMTP.Username,
			password: cfg.SMTP.Password,
			useTLS:   cfg.SMTP.UseTLS,
		}, nil

	case SenderModeNone:
		return &NoopSender{}, nil

	default:
		return nil, fmt.Errorf("invalid sender_mode %q (valid values: %s, %s, %s)", senderMode, SenderModeLog, SenderModeSMTP, SenderModeNone)
	}
}

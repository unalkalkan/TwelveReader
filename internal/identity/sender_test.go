package identity

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

const (
	testSessionTTL      = 24 * time.Hour
	testRefreshTokenTTL = 7 * 24 * time.Hour
	testLinkExpiry      = 15 * time.Minute
)

// --- NewEmailSender factory tests ---

func TestNewEmailSender_LogMode_Local(t *testing.T) {
	cfg := &types.AuthConfig{SenderMode: "log"}
	sender, err := NewEmailSender(cfg, "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logSender, ok := sender.(*LogEmailSender)
	if !ok {
		t.Fatalf("expected *LogEmailSender, got %T", sender)
	}
	if !logSender.DevMode {
		t.Fatal("DevMode should be true for log mode")
	}
}

func TestNewEmailSender_LogMode_Dev(t *testing.T) {
	cfg := &types.AuthConfig{SenderMode: "log"}
	sender, err := NewEmailSender(cfg, "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logSender, ok := sender.(*LogEmailSender)
	if !ok {
		t.Fatalf("expected *LogEmailSender, got %T", sender)
	}
	if !logSender.DevMode {
		t.Fatal("DevMode should be true for log mode")
	}
}

func TestNewEmailSender_LogMode_BlockedInProduction(t *testing.T) {
	cfg := &types.AuthConfig{SenderMode: "log"}
	_, err := NewEmailSender(cfg, "production")
	if err == nil {
		t.Fatal("expected error when using log mode in production")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("error should mention not allowed, got: %v", err)
	}
}

func TestNewEmailSender_LogMode_BlockedInStaging(t *testing.T) {
	cfg := &types.AuthConfig{SenderMode: "log"}
	_, err := NewEmailSender(cfg, "staging")
	if err == nil {
		t.Fatal("expected error when using log mode in staging")
	}
}

func TestNewEmailSender_SmtpMode_RequiresConfig(t *testing.T) {
	cfg := &types.AuthConfig{SenderMode: "smtp"}
	_, err := NewEmailSender(cfg, "production")
	if err == nil {
		t.Fatal("expected error for SMTP mode without config")
	}
	if !strings.Contains(err.Error(), "SMTP configuration") {
		t.Fatalf("error should mention SMTP config, got: %v", err)
	}
}

func TestNewEmailSender_SmtpMode_WithConfig(t *testing.T) {
	cfg := &types.AuthConfig{
		SenderMode: "smtp",
		SenderFrom: "noreply@example.com",
		SMTP: types.SMTPConfig{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user",
			Password: "pass",
			UseTLS:   true,
		},
	}
	sender, err := NewEmailSender(cfg, "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	smtpSender, ok := sender.(*SMTPSender)
	if !ok {
		t.Fatalf("expected *SMTPSender, got %T", sender)
	}
	if smtpSender.host != "smtp.example.com" {
		t.Fatalf("host = %q, want smtp.example.com", smtpSender.host)
	}
	if smtpSender.port != 587 {
		t.Fatalf("port = %d, want 587", smtpSender.port)
	}
}

func TestNewEmailSender_SmtpMode_MissingHost(t *testing.T) {
	cfg := &types.AuthConfig{
		SenderMode: "smtp",
		SMTP: types.SMTPConfig{
			Port: 587, // port set but no host
		},
	}
	_, err := NewEmailSender(cfg, "production")
	if err == nil {
		t.Fatal("expected error for SMTP mode with missing host")
	}
}

func TestNewEmailSender_SmtpMode_MissingPort(t *testing.T) {
	cfg := &types.AuthConfig{
		SenderMode: "smtp",
		SMTP: types.SMTPConfig{
			Host: "smtp.example.com", // host set but no port
		},
	}
	_, err := NewEmailSender(cfg, "production")
	if err == nil {
		t.Fatal("expected error for SMTP mode with missing port")
	}
}

func TestNewEmailSender_NoneMode(t *testing.T) {
	cfg := &types.AuthConfig{SenderMode: "none"}
	sender, err := NewEmailSender(cfg, "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := sender.(*NoopSender)
	if !ok {
		t.Fatalf("expected *NoopSender, got %T", sender)
	}
}

func TestNewEmailSender_InvalidMode(t *testing.T) {
	cfg := &types.AuthConfig{SenderMode: "fax"}
	_, err := NewEmailSender(cfg, "local")
	if err == nil {
		t.Fatal("expected error for invalid sender mode")
	}
	if !strings.Contains(err.Error(), "invalid sender_mode") {
		t.Fatalf("error should mention invalid sender_mode, got: %v", err)
	}
}

func TestNewEmailSender_EmptyMode_DefaultsToLocal(t *testing.T) {
	cfg := &types.AuthConfig{} // empty SenderMode
	sender, err := NewEmailSender(cfg, "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logSender, ok := sender.(*LogEmailSender)
	if !ok {
		t.Fatalf("expected *LogEmailSender for empty mode in local env, got %T", sender)
	}
	if !logSender.DevMode {
		t.Fatal("DevMode should be true for default log mode")
	}
}

func TestNewEmailSender_EmptyMode_DefaultsToNoneInProd(t *testing.T) {
	cfg := &types.AuthConfig{} // empty SenderMode
	sender, err := NewEmailSender(cfg, "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := sender.(*NoopSender)
	if !ok {
		t.Fatalf("expected *NoopSender for empty mode in production env, got %T", sender)
	}
}

func TestNewEmailSender_EmptyMode_DefaultsToNoneInStaging(t *testing.T) {
	cfg := &types.AuthConfig{} // empty SenderMode
	sender, err := NewEmailSender(cfg, "staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := sender.(*NoopSender)
	if !ok {
		t.Fatalf("expected *NoopSender for empty mode in staging env, got %T", sender)
	}
}

func TestNewEmailSender_EmptyMode_DefaultsToLogInDev(t *testing.T) {
	cfg := &types.AuthConfig{} // empty SenderMode
	sender, err := NewEmailSender(cfg, "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := sender.(*LogEmailSender)
	if !ok {
		t.Fatalf("expected *LogEmailSender for empty mode in dev env, got %T", sender)
	}
}

// --- NoopSender fail-closed tests ---

func TestNoopSender_FailsOnSend(t *testing.T) {
	sender := &NoopSender{}
	err := sender.SendMagicLink("test@example.com", "Subject", "Body with token")
	if err == nil {
		t.Fatal("expected NoopSender to always return an error")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error should mention not configured, got: %v", err)
	}
}

// --- LogEmailSender dev mode is tested in auth_security_test.go ---
// (TestLogEmailSender_DevMode_LogsBody, TestLogEmailSender_NonDevMode_SuppressesBody)

// --- Integration: auth service with different senders ---

func TestAuthService_RequestMagicLink_WithLogSender(t *testing.T) {
	pool := newTestPool(t)
	var buf strings.Builder
	sender := &LogEmailSender{DevMode: true, output: &buf}
	svc := NewAuthService(pool, sender, "http://localhost:3000", "noreply@example.com",
		testSessionTTL, testRefreshTokenTTL, testLinkExpiry)

	ctx := context.Background()
	token, err := svc.RequestMagicLink(ctx, "logtest@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink with log sender: %v", err)
	}
	if len(token) == 0 {
		t.Fatal("expected non-empty token from RequestMagicLink")
	}

	out := buf.String()
	if !strings.Contains(out, token) {
		t.Fatal("Log sender in DevMode should log the magic link token")
	}
	if !strings.Contains(out, "/api/v1/auth/verify?token=") {
		t.Fatal("Log sender should log the full verify URL")
	}

	// Verify the token actually works (proves end-to-end flow)
	result, err := svc.VerifyMagicLink(ctx, token, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}
	if result.User == nil || result.User.Email != "logtest@example.com" {
		t.Fatal("verified user mismatch")
	}
}

func TestAuthService_RequestMagicLink_WithNoopSender(t *testing.T) {
	pool := newTestPool(t)
	sender := &NoopSender{}
	svc := NewAuthService(pool, sender, "http://localhost:3000", "noreply@example.com",
		testSessionTTL, testRefreshTokenTTL, testLinkExpiry)

	ctx := context.Background()
	_, err := svc.RequestMagicLink(ctx, "noop@example.com")
	if err == nil {
		t.Fatal("expected error when using NoopSender (fail-closed)")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error should mention not configured, got: %v", err)
	}
}

func TestAuthService_VerifyMagicLink_RemainsIntact(t *testing.T) {
	// Verify that core auth/verify flow still works with the new sender selection.
	pool := newTestPool(t)
	var buf strings.Builder
	sender := &LogEmailSender{DevMode: true, output: &buf}
	svc := NewAuthService(pool, sender, "http://localhost:3000", "noreply@example.com",
		testSessionTTL, testRefreshTokenTTL, testLinkExpiry)

	ctx := context.Background()

	// Request + verify flow
	token, err := svc.RequestMagicLink(ctx, "verifytest@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	result, err := svc.VerifyMagicLink(ctx, token, "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}
	if result.User == nil {
		t.Fatal("user is nil")
	}
	if result.SessionToken == "" || result.RefreshToken == "" {
		t.Fatal("tokens should be non-empty after verify")
	}

	// Session lookup works
	session, err := svc.GetSessionByTokenHash(ctx, result.SessionToken)
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if session.UserID != result.User.ID {
		t.Fatal("session user mismatch")
	}

	// Refresh works
	newResult, err := svc.RefreshSession(ctx, result.RefreshToken, "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}
	if newResult.SessionToken == result.SessionToken {
		t.Fatal("session token should rotate on refresh")
	}

	// Logout works
	err = svc.Logout(ctx, newResult.Session.ID)
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Session invalid after logout
	_, err = svc.GetSessionByTokenHash(ctx, newResult.SessionToken)
	if err == nil {
		t.Fatal("session should be invalid after logout")
	}
}

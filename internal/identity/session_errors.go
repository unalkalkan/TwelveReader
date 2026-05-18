package identity

import "errors"

var (
	// ErrSessionExpired is returned when a session has passed its expiry time.
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionRevoked is returned when a session has been explicitly revoked.
	ErrSessionRevoked = errors.New("session revoked")

	// ErrRefreshTokenExpired is returned when a refresh token has passed its expiry time.
	ErrRefreshTokenExpired = errors.New("refresh token expired")
)

// IsSessionExpired checks if an error indicates an expired session.
func IsSessionExpired(err error) bool {
	return err != nil && (errors.Is(err, ErrSessionExpired) || contains(err.Error(), "expired"))
}

// IsSessionRevoked checks if an error indicates a revoked session.
func IsSessionRevoked(err error) bool {
	return err != nil && (errors.Is(err, ErrSessionRevoked) || contains(err.Error(), "revoked"))
}

// SessionError wraps a session validation failure with a machine-readable cause.
type SessionError struct {
	Cause   string // "expired", "revoked", "invalid"
	Message string
}

func (e *SessionError) Error() string {
	return e.Message
}

func newSessionError(cause, message string) *SessionError {
	return &SessionError{Cause: cause, Message: message}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

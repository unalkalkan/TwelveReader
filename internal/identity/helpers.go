package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// GenerateID creates a new UUID v4 using crypto/rand.
func GenerateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	// Version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Variant RFC 4122
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// HashToken hashes a raw token string with SHA-256 for storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// formatTimeUTC returns a UTC RFC3339 time string.
func formatTimeUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// parseJSONMetadata decodes JSON bytes from metadata column.
func parseJSONMetadata(data []byte) (map[string]string, error) {
	if len(data) == 0 || string(data) == "{}" {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return m, nil
}

// marshalJSONMetadata encodes a map to JSON bytes.
func marshalJSONMetadata(m map[string]string) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

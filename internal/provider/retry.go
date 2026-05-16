package provider

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

const (
	defaultMaxRetries     = 3
	defaultRetryBackoffMs = 500
	maxBackoffMs          = 30000
)

func isRetryableStatusCode(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func computeBackoff(attempt int, baseBackoffMs int) time.Duration {
	ms := float64(baseBackoffMs) * math.Pow(2, float64(attempt))
	if ms > maxBackoffMs {
		ms = maxBackoffMs
	}
	return time.Duration(ms) * time.Millisecond
}

func parseRetryOptions(options map[string]string) (maxRetries int, retryBackoffMs int) {
	maxRetries = defaultMaxRetries
	retryBackoffMs = defaultRetryBackoffMs

	if v, ok := options["max_retries"]; ok {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n >= 0 {
			maxRetries = n
		}
	}
	if v, ok := options["retry_backoff_ms"]; ok {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			retryBackoffMs = n
		}
	}
	return
}

func sleepBeforeRetry(ctx context.Context, backoff time.Duration) error {
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func newJSONPostRequest(ctx context.Context, endpoint string, body []byte, apiKey string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
	return req, nil
}
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStructuredErrorCodeConstants(t *testing.T) {
	expectedCodes := map[string]int{
		ErrCodeNotFound:           http.StatusNotFound,
		ErrCodeMethodNotAllowed:   http.StatusMethodNotAllowed,
		ErrCodeBadRequest:         http.StatusBadRequest,
		ErrCodeUnauthorized:       http.StatusUnauthorized,
		ErrCodeForbidden:          http.StatusForbidden,
		ErrCodeConflict:           http.StatusConflict,
		ErrCodeServiceUnavailable: http.StatusServiceUnavailable,
		ErrCodeTooManyRequests:    http.StatusTooManyRequests,
	}

	for code, expectedStatus := range expectedCodes {
		actualStatus := HTTPStatusCodeForCode(code)
		if actualStatus != expectedStatus {
			t.Errorf("HTTPStatusCodeForCode(%q) = %d; want %d", code, actualStatus, expectedStatus)
		}
	}

	// Default case: unknown code -> 500
	if got := HTTPStatusCodeForCode("UNKNOWN_CODE"); got != http.StatusInternalServerError {
		t.Errorf("HTTPStatusCodeForCode(unknown) = %d; want 500", got)
	}
}

func TestWriteStructuredError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	WriteStructuredError(w, req, ErrCodeNotFound, "resource not found", http.StatusNotFound)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d; want 404", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type = %q; want application/json", contentType)
	}

	var errResp StructuredError
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("error.code = %q; want %q", errResp.Error.Code, ErrCodeNotFound)
	}
	if errResp.Error.Message != "resource not found" {
		t.Errorf("error.message = %q; want %q", errResp.Error.Message, "resource not found")
	}
	// No request ID in context, so RequestID should be empty string (omitempty -> absent)
	if errResp.Error.RequestID != "" {
		t.Errorf("error.request_id = %q; want empty when no ctx", errResp.Error.RequestID)
	}
}

func TestWriteStructuredErrorWithRequestID(t *testing.T) {
	rc := &RequestContext{}
	var handlerWrote bool

	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteStructuredError(w, r, ErrCodeBadRequest, "invalid input", http.StatusBadRequest)
		handlerWrote = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set(RequestIDHeader, "test-req-id-12345")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !handlerWrote {
		t.Fatal("handler was not called")
	}

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d; want 400", resp.StatusCode)
	}

	var errResp StructuredError
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if errResp.Error.Code != ErrCodeBadRequest {
		t.Errorf("error.code = %q; want %q", errResp.Error.Code, ErrCodeBadRequest)
	}
	if errResp.Error.Message != "invalid input" {
		t.Errorf("error.message = %q; want %q", errResp.Error.Message, "invalid input")
	}
	if errResp.Error.RequestID != "test-req-id-12345" {
		t.Errorf("error.request_id = %q; want %q", errResp.Error.RequestID, "test-req-id-12345")
	}

	// Also verify response header has the request ID
	respHeaderReqID := resp.Header.Get(RequestIDHeader)
	if respHeaderReqID != "test-req-id-12345" {
		t.Errorf("response X-Request-ID header = %q; want test-req-id-12345", respHeaderReqID)
	}
}

func TestWriteMethodNotAllowedError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	WriteMethodNotAllowedError(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d; want 405", resp.StatusCode)
	}

	var errResp StructuredError
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if errResp.Error.Code != ErrCodeMethodNotAllowed {
		t.Errorf("error.code = %q; want %q", errResp.Error.Code, ErrCodeMethodNotAllowed)
	}
}

func TestWriteNotFoundError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	WriteNotFoundError(w, req, "book")

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d; want 404", resp.StatusCode)
	}

	var errResp StructuredError
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("error.code = %q; want %q", errResp.Error.Code, ErrCodeNotFound)
	}
	if !strings.Contains(errResp.Error.Message, "book") {
		t.Errorf("error.message should mention 'book', got %q", errResp.Error.Message)
	}
	if !strings.Contains(errResp.Error.Message, "not found") {
		t.Errorf("error.message should mention 'not found', got %q", errResp.Error.Message)
	}
}

func TestWriteStructuredErrorWithGeneratedRequestID(t *testing.T) {
	rc := &RequestContext{}

	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteStructuredError(w, r, ErrCodeInternal, "something went wrong", http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// No X-Request-ID header — middleware should generate one
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()

	// Response header should have a generated request ID
	respHeaderReqID := resp.Header.Get(RequestIDHeader)
	if respHeaderReqID == "" {
		t.Fatal("expected generated X-Request-ID in response header")
	}

	var errResp StructuredError
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Error body request_id should match the header
	if errResp.Error.RequestID != respHeaderReqID {
		t.Errorf("error.request_id = %q; want header value %q", errResp.Error.RequestID, respHeaderReqID)
	}
}

func TestWriteStructuredErrorf(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	WriteStructuredErrorf(w, req, ErrCodeBadRequest, "invalid value: %s (expected %d)", []interface{}{"abc", 42}, http.StatusBadRequest)

	resp := w.Result()
	var errResp StructuredError
	json.NewDecoder(resp.Body).Decode(&errResp)

	expectedMsg := "invalid value: abc (expected 42)"
	if errResp.Error.Message != expectedMsg {
		t.Errorf("error.message = %q; want %q", errResp.Error.Message, expectedMsg)
	}
}

func TestStructuredErrorJSONMarshal(t *testing.T) {
	ctx := context.Background()
	reqID := GenerateRequestID()
	ctx = context.WithValue(ctx, requestIDKey{}, reqID)

	err := StructuredError{
		Error: ErrorBody{
			Code:      ErrCodeNotFound,
			Message:   "item not found",
			RequestID: reqID,
		},
	}

	data, errMarshal := json.Marshal(err)
	if errMarshal != nil {
		t.Fatalf("marshal failed: %v", errMarshal)
	}

	// Verify JSON structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	errObj, ok := raw["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected top-level 'error' key with object value")
	}

	if errObj["code"] != ErrCodeNotFound {
		t.Errorf("JSON error.code = %v; want %q", errObj["code"], ErrCodeNotFound)
	}
	if errObj["message"] != "item not found" {
		t.Errorf("JSON error.message = %v; want %q", errObj["message"], "item not found")
	}
	if errObj["request_id"] != reqID {
		t.Errorf("JSON error.request_id = %v; want %q", errObj["request_id"], reqID)
	}
}

func TestStructuredErrorOmitEmptyRequestID(t *testing.T) {
	err := StructuredError{
		Error: ErrorBody{
			Code:    ErrCodeNotFound,
			Message: "not found",
			// RequestID intentionally omitted (empty string -> omitempty)
		},
	}

	data, _ := json.Marshal(err)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	errObj := raw["error"].(map[string]interface{})
	if _, exists := errObj["request_id"]; exists {
		t.Error("expected request_id to be omitted when empty")
	}
}

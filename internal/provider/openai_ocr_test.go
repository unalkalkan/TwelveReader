package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestNewOpenAIOCRProvider(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		if provider.Name() != "test-openai-ocr" {
			t.Errorf("Expected name 'test-openai-ocr', got '%s'", provider.Name())
		}

		if provider.model != "gpt-4o" {
			t.Errorf("Expected model 'gpt-4o', got '%s'", provider.model)
		}
	})

	t.Run("MissingEndpoint", func(t *testing.T) {
		cfg := types.OCRProviderConfig{
			Name:    "test-openai-ocr",
			Enabled: true,
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		_, err := NewOpenAIOCRProvider(cfg)
		if err == nil {
			t.Error("Expected error for missing endpoint")
		}
		if !strings.Contains(err.Error(), "endpoint is required") {
			t.Errorf("Expected error about endpoint, got: %v", err)
		}
	})

	t.Run("MissingModel", func(t *testing.T) {
		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options:  map[string]string{},
		}

		_, err := NewOpenAIOCRProvider(cfg)
		if err == nil {
			t.Error("Expected error for missing model")
		}
		if !strings.Contains(err.Error(), "model is required") {
			t.Errorf("Expected error about model, got: %v", err)
		}
	})

	t.Run("CustomTimeout", func(t *testing.T) {
		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options: map[string]string{
				"model":   "gpt-4o",
				"timeout": "60",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		if provider.httpClient.Timeout.Seconds() != 60 {
			t.Errorf("Expected timeout 60s, got %v", provider.httpClient.Timeout.Seconds())
		}
	})

	t.Run("RequestLimitOptions", func(t *testing.T) {
		provider, err := NewOpenAIOCRProvider(types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options: map[string]string{
				"model":              "gpt-4o",
				"max_image_bytes":    "128",
				"max_response_bytes": "256",
				"max_tokens":         "512",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}
		if provider.maxImageBytes != 128 {
			t.Errorf("Expected max image bytes 128, got %d", provider.maxImageBytes)
		}
		if provider.maxRespBytes != 256 {
			t.Errorf("Expected max response bytes 256, got %d", provider.maxRespBytes)
		}
		if provider.maxTokens != 512 {
			t.Errorf("Expected max tokens 512, got %d", provider.maxTokens)
		}
	})
}

func TestOpenAIOCRProvider_ExtractText(t *testing.T) {
	t.Run("SuccessfulOCRExtraction", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
				t.Errorf("Expected /chat/completions endpoint, got %s", r.URL.Path)
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-key" {
				t.Errorf("Expected 'Bearer test-key', got '%s'", authHeader)
			}

			var reqBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Errorf("Failed to decode request: %v", err)
			}

			if model, ok := reqBody["model"].(string); !ok || model != "gpt-4o" {
				t.Errorf("Expected model 'gpt-4o', got %v", reqBody["model"])
			}
			if maxTokens, ok := reqBody["max_tokens"].(float64); !ok || maxTokens != defaultOCRMaxTokens {
				t.Errorf("Expected default max_tokens %d, got %v", defaultOCRMaxTokens, reqBody["max_tokens"])
			}

			messages, ok := reqBody["messages"].([]interface{})
			if !ok || len(messages) == 0 {
				t.Errorf("Expected messages array, got %v", reqBody["messages"])
			}

			resp := chatCompletionResponse{
				ID:      "ocr-test-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o",
				Choices: []choice{
					{
						Index: 0,
						Message: message{
							Role:    "assistant",
							Content: "This is the extracted text from the image.",
						},
						FinishReason: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake png image data for testing"),
			Language:  "en",
		}

		resp, err := provider.ExtractText(ctx, req)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}

		if resp.Text != "This is the extracted text from the image." {
			t.Errorf("Expected extracted text, got '%s'", resp.Text)
		}
		if resp.Confidence < 0 || resp.Confidence > 1 {
			t.Errorf("Expected confidence between 0 and 1, got %f", resp.Confidence)
		}
	})

	t.Run("LanguageAwarePrompt", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var reqBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Errorf("Failed to decode request: %v", err)
			}

			messages, ok := reqBody["messages"].([]interface{})
			if !ok || len(messages) < 2 {
				t.Fatalf("Expected at least 2 messages, got %v", messages)
			}

			userMsg, ok := messages[1].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected user message as map, got %T", messages[1])
			}

			content, ok := userMsg["content"].([]interface{})
			if !ok {
				t.Fatalf("Expected content array, got %T", userMsg["content"])
			}

			textPart, ok := content[0].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected text part as map, got %T", content[0])
			}

			textContent, ok := textPart["text"].(string)
			if !ok {
				t.Fatalf("Expected text as string, got %T", textPart["text"])
			}

			if !strings.Contains(textContent, "Spanish") && !strings.Contains(textContent, "es") {
				if !strings.Contains(textContent, "language") {
					t.Errorf("Expected language-aware prompt to mention language, got: %s", textContent)
				}
			}

			resp := chatCompletionResponse{
				ID:     "ocr-test-id",
				Object: "chat.completion",
				Model:  "gpt-4o",
				Choices: []choice{
					{
						Index: 0,
						Message: message{
							Role:    "assistant",
							Content: "Texto extraído de la imagen.",
						},
						FinishReason: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake image data"),
			Language:  "es",
		}

		_, err = provider.ExtractText(ctx, req)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
	})

	t.Run("Base64DataURLRequestFormat", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var reqBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Errorf("Failed to decode request: %v", err)
			}

			messages, ok := reqBody["messages"].([]interface{})
			if !ok || len(messages) < 2 {
				t.Fatalf("Expected at least 2 messages, got %v", messages)
			}

			userMsg, ok := messages[1].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected user message as map, got %T", messages[1])
			}

			content, ok := userMsg["content"].([]interface{})
			if !ok {
				t.Fatalf("Expected content array, got %T", userMsg["content"])
			}

			if len(content) != 2 {
				t.Fatalf("Expected 2 content parts (text + image), got %d", len(content))
			}

			imagePart, ok := content[1].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected image part as map, got %T", content[1])
			}

			if imagePart["type"] != "image_url" {
				t.Errorf("Expected second content part type 'image_url', got '%v'", imagePart["type"])
			}

			imageURL, ok := imagePart["image_url"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected image_url object, got %T", imagePart["image_url"])
			}

			urlStr, ok := imageURL["url"].(string)
			if !ok {
				t.Fatalf("Expected url string, got %T", imageURL["url"])
			}

			if !strings.HasPrefix(urlStr, "data:image/") {
				t.Errorf("Expected base64 data URL prefix 'data:image/', got '%s'", urlStr[:30])
			}

			if !strings.Contains(urlStr, ";base64,") {
				t.Errorf("Expected base64 encoding marker in data URL, got '%s'", urlStr[:60])
			}

			resp := chatCompletionResponse{
				Choices: []choice{{
					Message:      message{Role: "assistant", Content: "extracted text"},
					FinishReason: "stop",
				}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake image data"),
			Language:  "en",
		}

		_, err = provider.ExtractText(ctx, req)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
	})

	t.Run("RejectsImageDataOverConfiguredLimit", func(t *testing.T) {
		provider, err := NewOpenAIOCRProvider(types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options: map[string]string{
				"model":           "gpt-4o",
				"max_image_bytes": "4",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		_, err = provider.ExtractText(context.Background(), OCRRequest{ImageData: []byte("too large")})
		if err == nil {
			t.Fatal("Expected image size limit error")
		}
		if !strings.Contains(err.Error(), "exceeds OCR limit") {
			t.Errorf("Expected image limit error, got %v", err)
		}
	})

	t.Run("EmptyImageData", func(t *testing.T) {
		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte{},
			Language:  "en",
		}

		_, err = provider.ExtractText(ctx, req)
		if err == nil {
			t.Error("Expected error for empty image data")
		}
		if !strings.Contains(err.Error(), "image data") {
			t.Errorf("Expected error about image data, got: %v", err)
		}
	})

	t.Run("Non2xxResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			resp := apiErrorResponse{}
			resp.Error.Message = "Invalid API key"
			resp.Error.Type = "invalid_request_error"
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "invalid-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake image data"),
			Language:  "en",
		}

		_, err = provider.ExtractText(ctx, req)
		if err == nil {
			t.Error("Expected error for non-2xx response")
		}
		if !strings.Contains(err.Error(), "Invalid API key") {
			t.Errorf("Expected error to contain 'Invalid API key', got: %v", err)
		}
	})

	t.Run("EmptyChoicesResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				ID:      "ocr-test-id",
				Object:  "chat.completion",
				Model:   "gpt-4o",
				Choices: []choice{},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake image data"),
			Language:  "en",
		}

		_, err = provider.ExtractText(ctx, req)
		if err == nil {
			t.Error("Expected error for empty choices")
		}
		if !strings.Contains(err.Error(), "no choices") {
			t.Errorf("Expected error about no choices, got: %v", err)
		}
	})

	t.Run("EmptyExtractedText", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				ID:     "ocr-test-id",
				Object: "chat.completion",
				Model:  "gpt-4o",
				Choices: []choice{
					{
						Index: 0,
						Message: message{
							Role:    "assistant",
							Content: "   ",
						},
						FinishReason: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake image data"),
			Language:  "en",
		}

		_, err = provider.ExtractText(ctx, req)
		if err == nil {
			t.Error("Expected error for empty extracted text")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("Expected error about empty text, got: %v", err)
		}
	})

	t.Run("EmptyJSONExtractedText", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				Choices: []choice{{
					Message:      message{Role: "assistant", Content: `{"text":"   ","confidence":0.91}`},
					FinishReason: "stop",
				}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, err := NewOpenAIOCRProvider(types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options:  map[string]string{"model": "gpt-4o"},
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		_, err = provider.ExtractText(context.Background(), OCRRequest{ImageData: []byte("fake image data")})
		if err == nil {
			t.Fatal("Expected error for empty JSON text field")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("Expected error about empty JSON text field, got: %v", err)
		}
	})

	t.Run("JSONTextWithoutConfidenceDefaultsToOne", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				Choices: []choice{{
					Message:      message{Role: "assistant", Content: `{"text":"JSON extracted text"}`},
					FinishReason: "stop",
				}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, err := NewOpenAIOCRProvider(types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options:  map[string]string{"model": "gpt-4o"},
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		resp, err := provider.ExtractText(context.Background(), OCRRequest{ImageData: []byte("fake image data")})
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
		if resp.Text != "JSON extracted text" {
			t.Errorf("Expected JSON extracted text, got %q", resp.Text)
		}
		if resp.Confidence != 1.0 {
			t.Errorf("Expected default confidence 1.0 for JSON without confidence, got %f", resp.Confidence)
		}
	})

	t.Run("ConfidenceParsingFromJSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				ID:     "ocr-test-id",
				Object: "chat.completion",
				Model:  "gpt-4o",
				Choices: []choice{
					{
						Index: 0,
						Message: message{
							Role:    "assistant",
							Content: `{"text": "Extracted text content", "confidence": 0.92}`,
						},
						FinishReason: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake image data"),
			Language:  "en",
		}

		resp, err := provider.ExtractText(ctx, req)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}

		if resp.Text != "Extracted text content" {
			t.Errorf("Expected 'Extracted text content', got '%s'", resp.Text)
		}
		if resp.Confidence != 0.92 {
			t.Errorf("Expected confidence 0.92, got %f", resp.Confidence)
		}
	})

	t.Run("PlainTextResponseDefaultConfidence", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				Choices: []choice{{
					Message:      message{Role: "assistant", Content: "Plain extracted text"},
					FinishReason: "stop",
				}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := OCRRequest{
			ImageData: []byte("fake image data"),
		}

		resp, err := provider.ExtractText(ctx, req)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}

		if resp.Text != "Plain extracted text" {
			t.Errorf("Expected 'Plain extracted text', got '%s'", resp.Text)
		}
		if resp.Confidence != 1.0 {
			t.Errorf("Expected default confidence 1.0 for plain text response, got %f", resp.Confidence)
		}
	})

	t.Run("JPEGMimeTypeDetection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			messages := reqBody["messages"].([]interface{})
			userMsg := messages[1].(map[string]interface{})
			content := userMsg["content"].([]interface{})
			imagePart := content[1].(map[string]interface{})
			imageURL := imagePart["image_url"].(map[string]interface{})
			urlStr := imageURL["url"].(string)

			if !strings.HasPrefix(urlStr, "data:image/jpeg;base64,") && !strings.HasPrefix(urlStr, "data:image/jpg;base64,") {
				t.Errorf("Expected data URL with image/jpeg MIME type for JPEG data, got: %s", urlStr[:50])
			}

			resp := chatCompletionResponse{
				Choices: []choice{{Message: message{Role: "assistant", Content: "text"}, FinishReason: "stop"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.OCRProviderConfig{
			Name:     "test-openai-ocr",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o",
			},
		}

		provider, err := NewOpenAIOCRProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()

		jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
		req := OCRRequest{ImageData: jpegData}
		_, err = provider.ExtractText(ctx, req)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
	})
}

func TestOpenAIOCRProvider_HelperFunctions(t *testing.T) {
	if got := detectImageMimeType([]byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'}); got != "image/webp" {
		t.Errorf("Expected WEBP MIME type, got %q", got)
	}
	if got := detectImageMimeType([]byte{'G', 'I', 'F', '8'}); got != "image/gif" {
		t.Errorf("Expected GIF MIME type, got %q", got)
	}
	if got := detectImageMimeType([]byte{'B', 'M', 0, 0}); got != "image/bmp" {
		t.Errorf("Expected BMP MIME type, got %q", got)
	}
	if got := normalizeConfidence(-0.1); got != 1.0 {
		t.Errorf("Expected negative confidence to normalize to 1.0, got %f", got)
	}
	if got := normalizeConfidence(1.2); got != 1.0 {
		t.Errorf("Expected over-one confidence to normalize to 1.0, got %f", got)
	}
	if got := normalizeConfidence(0.42); got != 0.42 {
		t.Errorf("Expected valid confidence to pass through, got %f", got)
	}
}

func TestOpenAIOCRProvider_ResponseSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"` + strings.Repeat("x", 80) + `"}}]}`))
	}))
	defer server.Close()

	provider, err := NewOpenAIOCRProvider(types.OCRProviderConfig{
		Name:     "test-openai-ocr",
		Enabled:  true,
		Endpoint: server.URL,
		APIKey:   "test-key",
		Options: map[string]string{
			"model":              "gpt-4o",
			"max_response_bytes": "32",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	_, err = provider.ExtractText(context.Background(), OCRRequest{ImageData: []byte("image data")})
	if err == nil {
		t.Fatal("Expected response size limit error")
	}
	if !strings.Contains(err.Error(), "response exceeded") {
		t.Errorf("Expected response size limit error, got %v", err)
	}
}

func TestOpenAIOCRProvider_RetryOnTransientStatus(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("temporarily unavailable"))
			return
		}

		resp := chatCompletionResponse{
			Choices: []choice{{Message: message{Role: "assistant", Content: "OCR text"}, FinishReason: "stop"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewOpenAIOCRProvider(types.OCRProviderConfig{
		Name:     "test-openai-ocr",
		Enabled:  true,
		Endpoint: server.URL,
		APIKey:   "test-key",
		Options: map[string]string{
			"model":            "gpt-4o",
			"max_retries":      "1",
			"retry_backoff_ms": "1",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	resp, err := provider.ExtractText(context.Background(), OCRRequest{ImageData: []byte("image data"), Language: "en"})
	if err != nil {
		t.Fatalf("ExtractText failed after retry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("Expected 2 attempts, got %d", attempts)
	}
	if resp.Text != "OCR text" {
		t.Fatalf("Unexpected OCR text: %q", resp.Text)
	}
}

func TestOpenAIOCRProvider_ContextStopsRetryBackoff(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	provider, err := NewOpenAIOCRProvider(types.OCRProviderConfig{
		Name:     "test-openai-ocr",
		Enabled:  true,
		Endpoint: server.URL,
		APIKey:   "test-key",
		Options: map[string]string{
			"model":            "gpt-4o",
			"max_retries":      "3",
			"retry_backoff_ms": "1000",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = provider.ExtractText(ctx, OCRRequest{ImageData: []byte("image data")})
	if err == nil {
		t.Fatal("Expected context/backoff error")
	}
	if attempts != 1 {
		t.Fatalf("Expected retry backoff to stop after first attempt, got %d", attempts)
	}
}

func TestOpenAIOCRProvider_Close(t *testing.T) {
	cfg := types.OCRProviderConfig{
		Name:     "test-openai-ocr",
		Enabled:  true,
		Endpoint: "https://api.openai.com/v1",
		APIKey:   "test-key",
		Options: map[string]string{
			"model": "gpt-4o",
		},
	}

	provider, err := NewOpenAIOCRProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	err = provider.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestRegistryWithOpenAIOCRProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatCompletionResponse{
			Choices: []choice{{Message: message{Role: "assistant", Content: "Extracted OCR text"}, FinishReason: "stop"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	registry := NewRegistry()

	cfg := types.ProvidersConfig{
		OCR: []types.OCRProviderConfig{
			{
				Name:     "openai-ocr",
				Enabled:  true,
				Endpoint: server.URL,
				APIKey:   "test-key",
				Options: map[string]string{
					"model": "gpt-4o",
				},
			},
			{
				Name:    "stub-ocr",
				Enabled: true,
			},
		},
	}

	err := registry.InitializeProviders(cfg)
	if err != nil {
		t.Fatalf("InitializeProviders failed: %v", err)
	}

	ocrList := registry.ListOCR()
	if len(ocrList) != 2 {
		t.Fatalf("Expected 2 OCR providers, got %d", len(ocrList))
	}

	openaiProvider, err := registry.GetOCR("openai-ocr")
	if err != nil {
		t.Fatalf("Failed to get OpenAI OCR provider: %v", err)
	}

	ctx := context.Background()
	req := OCRRequest{
		ImageData: []byte("test image"),
		Language:  "en",
	}

	resp, err := openaiProvider.ExtractText(ctx, req)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	if resp.Text != "Extracted OCR text" {
		t.Errorf("Expected 'Extracted OCR text', got '%s'", resp.Text)
	}

	stubProvider, err := registry.GetOCR("stub-ocr")
	if err != nil {
		t.Fatalf("Failed to get stub provider: %v", err)
	}

	stubResp, err := stubProvider.ExtractText(ctx, OCRRequest{ImageData: []byte("test")})
	if err != nil {
		t.Fatalf("Stub ExtractText failed: %v", err)
	}
	if stubResp.Text == "" {
		t.Error("Expected non-empty text from stub")
	}
}

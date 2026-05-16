package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Create stub providers
	llmCfg := types.LLMProviderConfig{
		Name:    "test-llm",
		Enabled: true,
	}
	ttsCfg := types.TTSProviderConfig{
		Name:    "test-tts",
		Enabled: true,
	}
	ocrCfg := types.OCRProviderConfig{
		Name:    "test-ocr",
		Enabled: true,
	}

	llmProvider := NewStubLLMProvider(llmCfg)
	ttsProvider := NewStubTTSProvider(ttsCfg)
	ocrProvider := NewStubOCRProvider(ocrCfg)

	// Test registration
	t.Run("RegisterLLM", func(t *testing.T) {
		err := registry.RegisterLLM(llmProvider)
		if err != nil {
			t.Fatalf("Failed to register LLM provider: %v", err)
		}

		// Try to register again - should fail
		err = registry.RegisterLLM(llmProvider)
		if err == nil {
			t.Error("Expected error when registering duplicate provider")
		}
	})

	t.Run("RegisterTTS", func(t *testing.T) {
		err := registry.RegisterTTS(ttsProvider)
		if err != nil {
			t.Fatalf("Failed to register TTS provider: %v", err)
		}
	})

	t.Run("RegisterOCR", func(t *testing.T) {
		err := registry.RegisterOCR(ocrProvider)
		if err != nil {
			t.Fatalf("Failed to register OCR provider: %v", err)
		}
	})

	// Test retrieval
	t.Run("GetLLM", func(t *testing.T) {
		provider, err := registry.GetLLM("test-llm")
		if err != nil {
			t.Fatalf("Failed to get LLM provider: %v", err)
		}
		if provider.Name() != "test-llm" {
			t.Errorf("Expected name 'test-llm', got '%s'", provider.Name())
		}

		// Try to get non-existent provider
		_, err = registry.GetLLM("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent provider")
		}
	})

	t.Run("GetTTS", func(t *testing.T) {
		provider, err := registry.GetTTS("test-tts")
		if err != nil {
			t.Fatalf("Failed to get TTS provider: %v", err)
		}
		if provider.Name() != "test-tts" {
			t.Errorf("Expected name 'test-tts', got '%s'", provider.Name())
		}
	})

	t.Run("GetOCR", func(t *testing.T) {
		provider, err := registry.GetOCR("test-ocr")
		if err != nil {
			t.Fatalf("Failed to get OCR provider: %v", err)
		}
		if provider.Name() != "test-ocr" {
			t.Errorf("Expected name 'test-ocr', got '%s'", provider.Name())
		}
	})

	// Test listing
	t.Run("List", func(t *testing.T) {
		llmList := registry.ListLLM()
		if len(llmList) != 1 || llmList[0] != "test-llm" {
			t.Errorf("Expected LLM list ['test-llm'], got %v", llmList)
		}

		ttsList := registry.ListTTS()
		if len(ttsList) != 1 || ttsList[0] != "test-tts" {
			t.Errorf("Expected TTS list ['test-tts'], got %v", ttsList)
		}

		ocrList := registry.ListOCR()
		if len(ocrList) != 1 || ocrList[0] != "test-ocr" {
			t.Errorf("Expected OCR list ['test-ocr'], got %v", ocrList)
		}
	})

	// Test Close
	t.Run("Close", func(t *testing.T) {
		err := registry.Close()
		if err != nil {
			t.Fatalf("Failed to close registry: %v", err)
		}
	})
}

func TestStubProviders(t *testing.T) {
	ctx := context.Background()

	t.Run("StubLLMProvider", func(t *testing.T) {
		cfg := types.LLMProviderConfig{
			Name:    "test-llm",
			Enabled: true,
		}
		provider := NewStubLLMProvider(cfg)

		req := SegmentRequest{
			Text: "Test text",
		}
		resp, err := provider.Segment(ctx, req)
		if err != nil {
			t.Fatalf("Segment failed: %v", err)
		}
		if len(resp.Segments) != 1 {
			t.Errorf("Expected 1 segment, got %d", len(resp.Segments))
		}
		if resp.Segments[0].Text != "Test text" {
			t.Errorf("Expected text 'Test text', got '%s'", resp.Segments[0].Text)
		}
	})

	t.Run("StubTTSProvider", func(t *testing.T) {
		cfg := types.TTSProviderConfig{
			Name:    "test-tts",
			Enabled: true,
		}
		provider := NewStubTTSProvider(cfg)

		req := TTSRequest{
			Text:    "Test text",
			VoiceID: "test-voice",
		}
		resp, err := provider.Synthesize(ctx, req)
		if err != nil {
			t.Fatalf("Synthesize failed: %v", err)
		}
		if len(resp.AudioData) == 0 {
			t.Error("Expected non-empty audio data")
		}
		if resp.Format != "wav" {
			t.Errorf("Expected format 'wav', got '%s'", resp.Format)
		}
	})

	t.Run("StubOCRProvider", func(t *testing.T) {
		cfg := types.OCRProviderConfig{
			Name:    "test-ocr",
			Enabled: true,
		}
		provider := NewStubOCRProvider(cfg)

		req := OCRRequest{
			ImageData: []byte("fake image data"),
		}
		resp, err := provider.ExtractText(ctx, req)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
		if resp.Text == "" {
			t.Error("Expected non-empty text")
		}
		if resp.Confidence <= 0 || resp.Confidence > 1 {
			t.Errorf("Expected confidence between 0 and 1, got %f", resp.Confidence)
		}
	})
}

func TestInitializeProviders(t *testing.T) {
	registry := NewRegistry()

	cfg := types.ProvidersConfig{
		LLM: []types.LLMProviderConfig{
			{Name: "llm1", Enabled: true},
			{Name: "llm2", Enabled: false},
		},
		TTS: []types.TTSProviderConfig{
			{Name: "tts1", Enabled: true},
		},
		OCR: []types.OCRProviderConfig{
			{Name: "ocr1", Enabled: true},
		},
	}

	err := registry.InitializeProviders(cfg)
	if err != nil {
		t.Fatalf("InitializeProviders failed: %v", err)
	}

	llmList := registry.ListLLM()
	if len(llmList) != 1 || llmList[0] != "llm1" {
		t.Errorf("Expected LLM list ['llm1'], got %v", llmList)
	}

	ttsList := registry.ListTTS()
	if len(ttsList) != 1 || ttsList[0] != "tts1" {
		t.Errorf("Expected TTS list ['tts1'], got %v", ttsList)
	}

	ocrList := registry.ListOCR()
	if len(ocrList) != 1 || ocrList[0] != "ocr1" {
		t.Errorf("Expected OCR list ['ocr1'], got %v", ocrList)
	}
}

func TestInitializeProviders_OCRSelection(t *testing.T) {
	t.Run("StubFallbackWhenNoEndpoint", func(t *testing.T) {
		registry := NewRegistry()

		cfg := types.ProvidersConfig{
			OCR: []types.OCRProviderConfig{
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

		provider, err := registry.GetOCR("stub-ocr")
		if err != nil {
			t.Fatalf("Failed to get OCR provider: %v", err)
		}

		resp, err := provider.ExtractText(context.Background(), OCRRequest{ImageData: []byte("test")})
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
		if resp.Text != "Stub OCR extracted text" {
			t.Errorf("Expected stub response, got '%s'", resp.Text)
		}
	})

	t.Run("EndpointWithoutModelReturnsConfigurationError", func(t *testing.T) {
		registry := NewRegistry()

		cfg := types.ProvidersConfig{
			OCR: []types.OCRProviderConfig{
				{
					Name:     "stub-ocr-no-model",
					Enabled:  true,
					Endpoint: "https://api.openai.com/v1",
					Options:  map[string]string{},
				},
			},
		}

		err := registry.InitializeProviders(cfg)
		if err == nil {
			t.Fatal("Expected OCR provider configuration error")
		}
		if !strings.Contains(err.Error(), "model is required") {
			t.Fatalf("Expected missing model error, got %v", err)
		}
	})

	t.Run("RealProviderWhenEndpointAndModel", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				Choices: []choice{{Message: message{Role: "assistant", Content: "OCR text"}, FinishReason: "stop"}},
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
					Options: map[string]string{
						"model": "gpt-4o",
					},
				},
			},
		}

		err := registry.InitializeProviders(cfg)
		if err != nil {
			t.Fatalf("InitializeProviders failed: %v", err)
		}

		provider, err := registry.GetOCR("openai-ocr")
		if err != nil {
			t.Fatalf("Failed to get OCR provider: %v", err)
		}

		ctx := context.Background()
		resp, err := provider.ExtractText(ctx, OCRRequest{ImageData: []byte("test image"), Language: "en"})
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
		if resp.Text != "OCR text" {
			t.Errorf("Expected 'OCR text', got '%s'", resp.Text)
		}
	})

	t.Run("DisabledProviderNotRegistered", func(t *testing.T) {
		registry := NewRegistry()

		cfg := types.ProvidersConfig{
			OCR: []types.OCRProviderConfig{
				{Name: "disabled-ocr", Enabled: false},
			},
		}

		err := registry.InitializeProviders(cfg)
		if err != nil {
			t.Fatalf("InitializeProviders failed: %v", err)
		}

		ocrList := registry.ListOCR()
		if len(ocrList) != 0 {
			t.Errorf("Expected no OCR providers, got %v", ocrList)
		}
	})
}

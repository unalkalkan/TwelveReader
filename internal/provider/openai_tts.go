package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// OpenAITTSProvider implements TTSProvider using OpenAI-compatible TTS APIs
type OpenAITTSProvider struct {
	name       string
	config     types.TTSProviderConfig
	httpClient *http.Client
	model      string
}

// NewOpenAITTSProvider creates a new OpenAI-compatible TTS provider
func NewOpenAITTSProvider(config types.TTSProviderConfig) (*OpenAITTSProvider, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for OpenAI TTS provider")
	}

	// Get model from options
	model, ok := config.Options["model"]
	if !ok || model == "" {
		return nil, fmt.Errorf("model is required for OpenAI TTS provider (set in options.model)")
	}

	// Configure timeout from options or use default
	timeout := 120 * time.Second // TTS can take longer than LLM calls
	if timeoutStr, ok := config.Options["timeout"]; ok {
		var timeoutSec int
		if _, err := fmt.Sscanf(timeoutStr, "%d", &timeoutSec); err == nil && timeoutSec > 0 {
			timeout = time.Duration(timeoutSec) * time.Second
		}
	}

	return &OpenAITTSProvider{
		name:   config.Name,
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		model: model,
	}, nil
}

func (o *OpenAITTSProvider) Name() string {
	return o.name
}

// Synthesize converts text to speech using OpenAI-compatible API
func (o *OpenAITTSProvider) Synthesize(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	// Build the API request
	apiReq := ttsAPIRequest{
		Model: o.model,
		Input: req.Text,
		Voice: req.VoiceID,
	}

	// Add instructions if voice description is provided
	if req.VoiceDescription != "" {
		apiReq.Instructions = req.VoiceDescription
	}

	// Note: Language field is not used in the API request as OpenAI TTS API
	// doesn't have a direct language parameter. The model infers language from input.
	// This can be handled later if needed.

	// Call the API
	audioData, err := o.callTTSAPI(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call TTS API: %w", err)
	}

	// Return the response
	// Note: OpenAI TTS API doesn't provide word-level timestamps by default
	return &TTSResponse{
		AudioData:  audioData,
		Format:     "mp3",             // OpenAI returns MP3 by default
		Timestamps: []WordTimestamp{}, // Empty for now
	}, nil
}

// ListVoices returns available voices from the OpenAI TTS provider
func (o *OpenAITTSProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	// Build endpoint URL
	endpoint := o.config.Endpoint
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	endpoint += "voices"

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add model query parameter from provider config
	if o.model != "" {
		q := httpReq.URL.Query()
		q.Add("model", o.model)
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set headers
	if o.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.config.APIKey))
	}

	// Execute request
	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse as error response
		var errResp ttsAPIErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var apiResp voicesAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert to Voice structs
	voices := make([]Voice, 0, len(apiResp.Data))
	for _, v := range apiResp.Data {
		// Parse languages
		languages := v.Languages
		if len(languages) == 0 && v.Language != "" {
			languages = []string{v.Language}
		}

		voices = append(voices, Voice{
			ID:          v.ID,
			Name:        v.Name,
			Languages:   languages,
			Gender:      v.Gender,
			Accent:      v.Accent,
			Description: v.Description,
		})
	}

	return voices, nil
}

func (o *OpenAITTSProvider) Close() error {
	// Close HTTP client connections
	o.httpClient.CloseIdleConnections()
	return nil
}

// ttsAPIRequest represents the OpenAI TTS API request structure
type ttsAPIRequest struct {
	Model        string `json:"model"`
	Input        string `json:"input"`
	Voice        string `json:"voice"`
	Instructions string `json:"instructions,omitempty"`
}

// ttsAPIErrorResponse represents an error response from the TTS API
type ttsAPIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// voicesAPIResponse represents the response from the voices list endpoint
type voicesAPIResponse struct {
	Object string      `json:"object"`
	Data   []voiceData `json:"data"`
}

// voiceData represents voice metadata from the API
type voiceData struct {
	ID          string   `json:"id"`
	Object      string   `json:"object"`
	Name        string   `json:"name"`
	Language    string   `json:"language"`
	Languages   []string `json:"languages"`
	Gender      string   `json:"gender"`
	Accent      string   `json:"accent"`
	Description string   `json:"description"`
}

// callTTSAPI calls the OpenAI-compatible TTS endpoint
func (o *OpenAITTSProvider) callTTSAPI(ctx context.Context, req ttsAPIRequest) ([]byte, error) {
	// Encode request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build endpoint URL
	endpoint := o.config.Endpoint
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	endpoint += "audio/speech"

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if o.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.config.APIKey))
	}

	// Execute request
	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse as error response
		var errResp ttsAPIErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

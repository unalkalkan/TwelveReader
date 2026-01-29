package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// OpenAILLMProvider implements LLMProvider using OpenAI-compatible APIs
type OpenAILLMProvider struct {
	name       string
	config     types.LLMProviderConfig
	httpClient *http.Client
}

// NewOpenAILLMProvider creates a new OpenAI-compatible LLM provider
func NewOpenAILLMProvider(config types.LLMProviderConfig) (*OpenAILLMProvider, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for OpenAI LLM provider")
	}
	if config.Model == "" {
		return nil, fmt.Errorf("model is required for OpenAI LLM provider")
	}

	// Configure timeout from options or use default
	timeout := 60 * time.Second
	if timeoutStr, ok := config.Options["timeout"]; ok {
		var timeoutSec int
		if _, err := fmt.Sscanf(timeoutStr, "%d", &timeoutSec); err == nil && timeoutSec > 0 {
			timeout = time.Duration(timeoutSec) * time.Second
		}
	}

	return &OpenAILLMProvider{
		name:   config.Name,
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (o *OpenAILLMProvider) Name() string {
	return o.name
}

// Segment calls the OpenAI-compatible API to segment text
func (o *OpenAILLMProvider) Segment(ctx context.Context, req SegmentRequest) (*SegmentResponse, error) {
	// Build the prompt for segmentation
	prompt := o.buildSegmentationPrompt(req)

	// Call the OpenAI-compatible API
	apiResp, err := o.callChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM API: %w", err)
	}

	// Parse the response
	segments, err := o.parseSegmentationResponse(apiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return &SegmentResponse{
		Segments: segments,
	}, nil
}

func (o *OpenAILLMProvider) Close() error {
	// Close HTTP client connections
	o.httpClient.CloseIdleConnections()
	return nil
}

// buildSegmentationPrompt creates the prompt for the LLM
func (o *OpenAILLMProvider) buildSegmentationPrompt(req SegmentRequest) string {
	var sb strings.Builder

	sb.WriteString("You are a text segmentation expert. Your task is to analyze the given text and identify different speakers or narrative segments.\n\n")
	sb.WriteString("For each segment, provide:\n")
	sb.WriteString("1. The text of the segment\n")
	sb.WriteString("2. The person/speaker identifier (e.g., 'narrator', 'character1', 'dialogue_speaker')\n")
	sb.WriteString("3. The language (ISO-639-1 code, e.g., 'en', 'es')\n")
	sb.WriteString("4. A voice description (e.g., 'neutral', 'excited', 'somber')\n\n")

	if len(req.ContextBefore) > 0 {
		sb.WriteString("Previous context:\n")
		for _, ctx := range req.ContextBefore {
			sb.WriteString(fmt.Sprintf("- %s\n", ctx))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Text to segment:\n")
	sb.WriteString(req.Text)
	sb.WriteString("\n\n")

	if len(req.ContextAfter) > 0 {
		sb.WriteString("Following context:\n")
		for _, ctx := range req.ContextAfter {
			sb.WriteString(fmt.Sprintf("- %s\n", ctx))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Please respond with a JSON array of segments. Each segment should have the following structure:\n")
	sb.WriteString(`{"text": "segment text", "person": "speaker_id", "language": "en", "voice_description": "description"}`)
	sb.WriteString("\n\nProvide ONLY the JSON array, no additional text.")

	return sb.String()
}

// OpenAI API structures
type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// callChatCompletion calls the OpenAI-compatible chat completion endpoint
func (o *OpenAILLMProvider) callChatCompletion(ctx context.Context, prompt string) (string, error) {
	// Prepare request - parse temperature with default
	temperature := 0.0
	hasTemperature := false
	if tempStr, ok := o.config.Options["temperature"]; ok {
		var temp float64
		if _, err := fmt.Sscanf(tempStr, "%f", &temp); err == nil {
			temperature = temp
			hasTemperature = true
		} else {
			log.Printf("[LLM-%s] Warning: Failed to parse temperature value '%s', ignoring", o.name, tempStr)
		}
	}

	reqBody := chatCompletionRequest{
		Model: o.config.Model,
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Only set temperature if explicitly configured
	if hasTemperature {
		reqBody.Temperature = temperature
	}

	// Encode request
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	endpoint := o.config.Endpoint
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	endpoint += "chat/completions"

	// Log request details
	log.Printf("[LLM-%s] Request: POST %s", o.name, endpoint)
	log.Printf("[LLM-%s] Request payload: model=%s, temperature=%.2f, prompt_length=%d chars", o.name, o.config.Model, temperature, len(prompt))
	log.Printf("[LLM-%s] Request prompt (truncated): %s", o.name, truncateForLog(prompt, 500))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[LLM-%s] Failed to create request: %v", o.name, err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if o.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.config.APIKey))
	}

	// Execute request
	startTime := time.Now()
	resp, err := o.httpClient.Do(httpReq)
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("[LLM-%s] Request failed after %v: %v", o.name, duration, err)
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[LLM-%s] Response: %d %s (took %v)", o.name, resp.StatusCode, resp.Status, duration)

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LLM-%s] Failed to read response body: %v", o.name, err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errResp apiErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			log.Printf("[LLM-%s] API error: %s (type: %s, code: %s)", o.name, errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
			return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		log.Printf("[LLM-%s] API request failed: %s", o.name, truncateForLog(string(body), 500))
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp chatCompletionResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("[LLM-%s] Failed to parse response JSON: %v", o.name, err)
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		log.Printf("[LLM-%s] No choices in API response", o.name)
		return "", fmt.Errorf("no choices in API response")
	}

	content := apiResp.Choices[0].Message.Content
	log.Printf("[LLM-%s] Response payload: tokens(prompt=%d, completion=%d, total=%d), finish_reason=%s",
		o.name, apiResp.Usage.PromptTokens, apiResp.Usage.CompletionTokens, apiResp.Usage.TotalTokens, apiResp.Choices[0].FinishReason)
	log.Printf("[LLM-%s] Response content (truncated): %s", o.name, truncateForLog(content, 500))

	return content, nil
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	// Remove newlines for cleaner logs
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// parseSegmentationResponse parses the LLM response into segments
func (o *OpenAILLMProvider) parseSegmentationResponse(response string) ([]Segment, error) {
	// Trim whitespace and try to extract JSON array
	response = strings.TrimSpace(response)

	// Try to find JSON array in the response
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		// If no JSON array found, treat entire text as single segment
		return []Segment{
			{
				Text:             response,
				Person:           "narrator",
				Language:         "en",
				VoiceDescription: "neutral",
			},
		}, nil
	}

	jsonStr := response[startIdx : endIdx+1]

	// Define a temporary structure for parsing
	type tempSegment struct {
		Text             string `json:"text"`
		Person           string `json:"person"`
		Language         string `json:"language"`
		VoiceDescription string `json:"voice_description"`
	}

	var tempSegments []tempSegment
	if err := json.Unmarshal([]byte(jsonStr), &tempSegments); err != nil {
		// If parsing fails, return the full response as a single segment
		return []Segment{
			{
				Text:             response,
				Person:           "narrator",
				Language:         "en",
				VoiceDescription: "neutral",
			},
		}, nil
	}

	// Convert to Segment type
	segments := make([]Segment, 0, len(tempSegments))
	for _, ts := range tempSegments {
		// Set defaults if fields are empty
		person := ts.Person
		if person == "" {
			person = "narrator"
		}
		language := ts.Language
		if language == "" {
			language = "en"
		}
		voiceDesc := ts.VoiceDescription
		if voiceDesc == "" {
			voiceDesc = "neutral"
		}

		segments = append(segments, Segment{
			Text:             ts.Text,
			Person:           person,
			Language:         language,
			VoiceDescription: voiceDesc,
		})
	}

	return segments, nil
}

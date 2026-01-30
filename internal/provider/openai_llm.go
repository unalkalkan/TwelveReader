package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	// Configure timeout from options or use default (5 minutes)
	timeout := 300 * time.Second
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
	systemPrompt := o.buildSegmentationSystemPrompt()
	prompt := o.buildSegmentationPrompt(req)

	// Call the OpenAI-compatible API
	apiResp, err := o.callChatCompletion(ctx, []message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	})
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

	appendKnownPersons(&sb, req.KnownPersons)

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

func (o *OpenAILLMProvider) buildSegmentationSystemPrompt() string {
	return strings.Join([]string{
		"You are a text segmentation expert.",
		"You will be given a list of known people for the book.",
		"Always reuse the exact identifiers from that list when they match the speaker, including for quoted speech or internal thoughts.",
		"Do not create variants by changing spacing, underscores, casing, or adding suffixes like '(thought)' or '_spoken'.",
		"Only create a new person if none of the known people fit; when you do, use a concise snake_case identifier.",
		"Follow the output format exactly and return only valid JSON.",
	}, "\n")
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
func (o *OpenAILLMProvider) callChatCompletion(ctx context.Context, messages []message) (string, error) {
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
		Model:    o.config.Model,
		Messages: messages,
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
	promptLength := 0
	for _, msg := range messages {
		promptLength += len(msg.Content)
	}
	log.Printf("[LLM-%s] Request payload: model=%s, temperature=%.2f, message_count=%d, prompt_length=%d chars", o.name, o.config.Model, temperature, len(messages), promptLength)
	if len(messages) > 0 {
		log.Printf("[LLM-%s] Request prompt (truncated): %s", o.name, truncateForLog(messages[len(messages)-1].Content, 500))
	}

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

// BatchSegment processes multiple paragraphs in a single LLM call for efficiency
func (o *OpenAILLMProvider) BatchSegment(ctx context.Context, req BatchSegmentRequest) (*BatchSegmentResponse, error) {
	if len(req.Paragraphs) == 0 {
		return &BatchSegmentResponse{Results: []BatchParagraphResult{}}, nil
	}

	// Build the batch prompt
	systemPrompt := o.buildSegmentationSystemPrompt()
	prompt := o.buildBatchSegmentationPrompt(req)

	// Call the OpenAI-compatible API
	apiResp, err := o.callChatCompletion(ctx, []message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		// Check for token limit errors
		if isTokenLimitError(err) {
			return nil, &TokenLimitError{Err: err}
		}
		return nil, fmt.Errorf("failed to call LLM API: %w", err)
	}

	// Parse the batch response
	results, err := o.parseBatchSegmentationResponse(apiResp, req.Paragraphs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM batch response: %w", err)
	}

	return &BatchSegmentResponse{
		Results: results,
	}, nil
}

// TokenLimitError indicates the request exceeded token limits
type TokenLimitError struct {
	Err error
}

func (e *TokenLimitError) Error() string {
	return fmt.Sprintf("token limit exceeded: %v", e.Err)
}

func (e *TokenLimitError) Unwrap() error {
	return e.Err
}

// IsTokenLimitError checks if an error is a token limit error
func IsTokenLimitError(err error) bool {
	var tokenErr *TokenLimitError
	return errors.As(err, &tokenErr)
}

// isTokenLimitError checks API error for token limit issues
func isTokenLimitError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "token") &&
		(strings.Contains(errStr, "limit") ||
			strings.Contains(errStr, "exceed") ||
			strings.Contains(errStr, "maximum") ||
			strings.Contains(errStr, "too long") ||
			strings.Contains(errStr, "context_length"))
}

// buildBatchSegmentationPrompt creates a prompt for batch segmentation
func (o *OpenAILLMProvider) buildBatchSegmentationPrompt(req BatchSegmentRequest) string {
	var sb strings.Builder

	sb.WriteString("You are a text segmentation expert. Your task is to analyze multiple paragraphs and identify different speakers or narrative segments in each.\n\n")
	sb.WriteString("For each segment, provide:\n")
	sb.WriteString("1. The text of the segment\n")
	sb.WriteString("2. The person/speaker identifier (e.g., 'narrator', 'character1', 'dialogue_speaker')\n")
	sb.WriteString("3. The language (ISO-639-1 code, e.g., 'en', 'es')\n")
	sb.WriteString("4. A voice description (e.g., 'neutral', 'excited', 'somber')\n\n")

	appendKnownPersons(&sb, req.KnownPersons)

	sb.WriteString("I will provide multiple paragraphs numbered with their indices. Process each paragraph and return results grouped by paragraph index.\n\n")

	sb.WriteString("PARAGRAPHS TO PROCESS:\n")
	sb.WriteString("========================\n\n")

	for _, p := range req.Paragraphs {
		sb.WriteString(fmt.Sprintf("--- PARAGRAPH %d ---\n", p.Index))

		if len(p.ContextBefore) > 0 {
			sb.WriteString("Previous context:\n")
			for _, ctx := range p.ContextBefore {
				sb.WriteString(fmt.Sprintf("  > %s\n", truncateForLog(ctx, 200)))
			}
		}

		sb.WriteString(fmt.Sprintf("Text: %s\n", p.Text))

		if len(p.ContextAfter) > 0 {
			sb.WriteString("Following context:\n")
			for _, ctx := range p.ContextAfter {
				sb.WriteString(fmt.Sprintf("  > %s\n", truncateForLog(ctx, 200)))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("========================\n\n")
	sb.WriteString("Please respond with a JSON object containing results for each paragraph.\n")
	sb.WriteString("Format:\n")
	sb.WriteString(`{
  "paragraphs": [
    {
      "index": 0,
      "segments": [
        {"text": "segment text", "person": "speaker_id", "language": "en", "voice_description": "description"}
      ]
    }
  ]
}`)
	sb.WriteString("\n\nProvide ONLY the JSON object, no additional text.")

	return sb.String()
}

func appendKnownPersons(sb *strings.Builder, persons []string) {
	if len(persons) == 0 {
		return
	}
	sb.WriteString("Known people (reuse exact ids when applicable; create new only if none fit):\n")
	for _, person := range persons {
		sb.WriteString(fmt.Sprintf("- %s\n", person))
	}
	sb.WriteString("\n")
}

// parseBatchSegmentationResponse parses the LLM batch response
func (o *OpenAILLMProvider) parseBatchSegmentationResponse(response string, paragraphs []BatchParagraph) ([]BatchParagraphResult, error) {
	response = strings.TrimSpace(response)

	// Try to find JSON object in the response
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		// Fallback: return each paragraph as a single narrator segment
		log.Printf("[LLM-%s] No valid JSON in batch response, using fallback", o.name)
		return o.createFallbackBatchResults(paragraphs), nil
	}

	jsonStr := response[startIdx : endIdx+1]

	// Parse the batch response
	type tempBatchResponse struct {
		Paragraphs []struct {
			Index    int `json:"index"`
			Segments []struct {
				Text             string `json:"text"`
				Person           string `json:"person"`
				Language         string `json:"language"`
				VoiceDescription string `json:"voice_description"`
			} `json:"segments"`
		} `json:"paragraphs"`
	}

	var batchResp tempBatchResponse
	if err := json.Unmarshal([]byte(jsonStr), &batchResp); err != nil {
		log.Printf("[LLM-%s] Failed to parse batch JSON: %v, using fallback", o.name, err)
		return o.createFallbackBatchResults(paragraphs), nil
	}

	// Build result map for quick lookup
	resultMap := make(map[int][]Segment)
	for _, p := range batchResp.Paragraphs {
		segments := make([]Segment, 0, len(p.Segments))
		for _, s := range p.Segments {
			person := s.Person
			if person == "" {
				person = "narrator"
			}
			language := s.Language
			if language == "" {
				language = "en"
			}
			voiceDesc := s.VoiceDescription
			if voiceDesc == "" {
				voiceDesc = "neutral"
			}
			segments = append(segments, Segment{
				Text:             s.Text,
				Person:           person,
				Language:         language,
				VoiceDescription: voiceDesc,
			})
		}
		resultMap[p.Index] = segments
	}

	// Build results preserving original order
	results := make([]BatchParagraphResult, 0, len(paragraphs))
	for _, p := range paragraphs {
		segments, ok := resultMap[p.Index]
		if !ok || len(segments) == 0 {
			// Fallback for missing paragraphs
			segments = []Segment{
				{
					Text:             p.Text,
					Person:           "narrator",
					Language:         "en",
					VoiceDescription: "neutral",
				},
			}
		}
		results = append(results, BatchParagraphResult{
			ParagraphIndex: p.Index,
			Segments:       segments,
		})
	}

	return results, nil
}

// createFallbackBatchResults creates fallback results when LLM fails
func (o *OpenAILLMProvider) createFallbackBatchResults(paragraphs []BatchParagraph) []BatchParagraphResult {
	results := make([]BatchParagraphResult, 0, len(paragraphs))
	for _, p := range paragraphs {
		results = append(results, BatchParagraphResult{
			ParagraphIndex: p.Index,
			Segments: []Segment{
				{
					Text:             p.Text,
					Person:           "narrator",
					Language:         "en",
					VoiceDescription: "neutral",
				},
			},
		})
	}
	return results
}

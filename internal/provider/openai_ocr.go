package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

const (
	defaultOCRMaxImageBytes    = 20 * 1024 * 1024
	defaultOCRMaxResponseBytes = 4 * 1024 * 1024
	defaultOCRMaxTokens        = 8192
)

type OpenAIOCRProvider struct {
	name           string
	config         types.OCRProviderConfig
	httpClient     *http.Client
	model          string
	maxImageBytes  int64
	maxRespBytes   int64
	maxTokens      int
	maxRetries     int
	retryBackoffMs int
}

func NewOpenAIOCRProvider(config types.OCRProviderConfig) (*OpenAIOCRProvider, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for OpenAI OCR provider")
	}

	model, ok := config.Options["model"]
	if !ok || model == "" {
		return nil, fmt.Errorf("model is required for OpenAI OCR provider (set in options.model)")
	}

	timeout := 300 * time.Second
	if timeoutStr, ok := config.Options["timeout"]; ok {
		var timeoutSec int
		if _, err := fmt.Sscanf(timeoutStr, "%d", &timeoutSec); err == nil && timeoutSec > 0 {
			timeout = time.Duration(timeoutSec) * time.Second
		}
	}

	maxRetries, retryBackoffMs := parseRetryOptions(config.Options)
	maxImageBytes := parseInt64Option(config.Options, "max_image_bytes", defaultOCRMaxImageBytes)
	maxRespBytes := parseInt64Option(config.Options, "max_response_bytes", defaultOCRMaxResponseBytes)
	maxTokens := parseIntOption(config.Options, "max_tokens", defaultOCRMaxTokens)

	return &OpenAIOCRProvider{
		name:           config.Name,
		config:         config,
		httpClient:     &http.Client{Timeout: timeout},
		model:          model,
		maxImageBytes:  maxImageBytes,
		maxRespBytes:   maxRespBytes,
		maxTokens:      maxTokens,
		maxRetries:     maxRetries,
		retryBackoffMs: retryBackoffMs,
	}, nil
}

func (o *OpenAIOCRProvider) Name() string {
	return o.name
}

func (o *OpenAIOCRProvider) ExtractText(ctx context.Context, req OCRRequest) (*OCRResponse, error) {
	if len(req.ImageData) == 0 {
		return nil, fmt.Errorf("image data is required for OCR extraction")
	}
	if o.maxImageBytes > 0 && int64(len(req.ImageData)) > o.maxImageBytes {
		return nil, fmt.Errorf("image data exceeds OCR limit: %d bytes > %d bytes", len(req.ImageData), o.maxImageBytes)
	}

	dataURL := buildImageDataURL(req.ImageData)
	systemPrompt := o.buildSystemPrompt(req.Language)
	userPrompt := o.buildUserPrompt(req.Language)

	content := []ocrContentPart{
		{Type: "text", Text: userPrompt},
		{Type: "image_url", ImageURL: &ocrImageURL{URL: dataURL}},
	}

	ocrMessages := []ocrChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: content},
	}

	result, err := o.callVisionCompletion(ctx, ocrMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to call OCR API: %w", err)
	}

	return o.parseOCRResponse(result)
}

func (o *OpenAIOCRProvider) Close() error {
	o.httpClient.CloseIdleConnections()
	return nil
}

func (o *OpenAIOCRProvider) buildSystemPrompt(language string) string {
	prompt := "You are an expert OCR assistant. Your task is to extract all visible text from the provided image with high accuracy."
	if language != "" {
		langName := languageNameFromCode(language)
		prompt += fmt.Sprintf(" The primary language of the document is %s.", langName)
	}
	prompt += " Return only the extracted text. Do not add commentary, descriptions, or explanations."
	return prompt
}

func (o *OpenAIOCRProvider) buildUserPrompt(language string) string {
	if language != "" {
		langName := languageNameFromCode(language)
		return fmt.Sprintf("Please extract all text from this image. The document is in %s. Return the extracted text exactly as it appears.", langName)
	}
	return "Please extract all text from this image. Return the extracted text exactly as it appears."
}

func (o *OpenAIOCRProvider) parseOCRResponse(content string) (*OCRResponse, error) {
	text := strings.TrimSpace(content)
	if text == "" {
		return nil, fmt.Errorf("empty extracted text from OCR")
	}

	var structured struct {
		Text       string   `json:"text"`
		Confidence *float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(text), &structured); err == nil {
		structuredText := strings.TrimSpace(structured.Text)
		if structuredText == "" {
			return nil, fmt.Errorf("empty extracted text from OCR")
		}
		confidence := 1.0
		if structured.Confidence != nil {
			confidence = normalizeConfidence(*structured.Confidence)
		}
		return &OCRResponse{
			Text:       structuredText,
			Confidence: confidence,
		}, nil
	}

	return &OCRResponse{
		Text:       text,
		Confidence: 1.0,
	}, nil
}

type ocrContentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *ocrImageURL `json:"image_url,omitempty"`
}

type ocrImageURL struct {
	URL string `json:"url"`
}

type ocrChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ocrChatCompletionRequest struct {
	Model     string           `json:"model"`
	Messages  []ocrChatMessage `json:"messages"`
	MaxTokens int              `json:"max_tokens,omitempty"`
}

func (o *OpenAIOCRProvider) callVisionCompletion(ctx context.Context, messages []ocrChatMessage) (string, error) {
	reqBody := ocrChatCompletionRequest{
		Model:     o.model,
		Messages:  messages,
		MaxTokens: o.maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := o.config.Endpoint
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	endpoint += "chat/completions"

	log.Printf("[OCR-%s] Request: POST %s", o.name, endpoint)
	log.Printf("[OCR-%s] Request payload: model=%s, message_count=%d", o.name, o.model, len(messages))

	var body []byte
	for attempt := 0; attempt <= o.maxRetries; attempt++ {
		httpReq, err := newJSONPostRequest(ctx, endpoint, jsonData, o.config.APIKey)
		if err != nil {
			log.Printf("[OCR-%s] Failed to create request: %v", o.name, err)
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		startTime := time.Now()
		resp, err := o.httpClient.Do(httpReq)
		duration := time.Since(startTime)
		if err != nil {
			log.Printf("[OCR-%s] Request attempt %d/%d failed after %v: %v", o.name, attempt+1, o.maxRetries+1, duration, err)
			if attempt < o.maxRetries {
				if waitErr := sleepBeforeRetry(ctx, computeBackoff(attempt, o.retryBackoffMs)); waitErr != nil {
					return "", fmt.Errorf("failed to execute request: %w", err)
				}
				continue
			}
			return "", fmt.Errorf("failed to execute request: %w", err)
		}

		log.Printf("[OCR-%s] Response attempt %d/%d: %d %s (took %v)", o.name, attempt+1, o.maxRetries+1, resp.StatusCode, resp.Status, duration)

		body, err = io.ReadAll(io.LimitReader(resp.Body, o.maxRespBytes+1))
		resp.Body.Close()
		if err != nil {
			log.Printf("[OCR-%s] Failed to read response body: %v", o.name, err)
			return "", fmt.Errorf("failed to read response: %w", err)
		}
		if o.maxRespBytes > 0 && int64(len(body)) > o.maxRespBytes {
			return "", fmt.Errorf("OCR API response exceeded %d bytes", o.maxRespBytes)
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		if isRetryableStatusCode(resp.StatusCode) && attempt < o.maxRetries {
			log.Printf("[OCR-%s] Retryable API status %d; retrying after backoff", o.name, resp.StatusCode)
			if waitErr := sleepBeforeRetry(ctx, computeBackoff(attempt, o.retryBackoffMs)); waitErr != nil {
				return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			}
			continue
		}

		var errResp apiErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			log.Printf("[OCR-%s] API error: %s (type: %s, code: %s)", o.name, errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
			return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		log.Printf("[OCR-%s] API request failed: %s", o.name, truncateForLog(string(body), 500))
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp chatCompletionResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("[OCR-%s] Failed to parse response JSON: %v", o.name, err)
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		log.Printf("[OCR-%s] No choices in API response", o.name)
		return "", fmt.Errorf("no choices in API response")
	}

	content := apiResp.Choices[0].Message.Content
	log.Printf("[OCR-%s] Response: tokens(prompt=%d, completion=%d, total=%d)", o.name, apiResp.Usage.PromptTokens, apiResp.Usage.CompletionTokens, apiResp.Usage.TotalTokens)
	log.Printf("[OCR-%s] Response content (truncated): %s", o.name, truncateForLog(content, 500))

	return content, nil
}

func buildImageDataURL(imageData []byte) string {
	mimeType := detectImageMimeType(imageData)
	encoded := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

func normalizeConfidence(confidence float64) float64 {
	if confidence < 0 || confidence > 1 {
		return 1.0
	}
	return confidence
}

func parseIntOption(options map[string]string, key string, fallback int) int {
	value, ok := options[key]
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func parseInt64Option(options map[string]string, key string, fallback int64) int64 {
	value, ok := options[key]
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func detectImageMimeType(data []byte) string {
	if len(data) < 4 {
		return "image/png"
	}

	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	if len(data) >= 4 && string(data[0:4]) == "RIFF" && len(data) >= 12 && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	if len(data) >= 3 && (string(data[0:3]) == "GIF") {
		return "image/gif"
	}
	if len(data) >= 2 && data[0] == 0x42 && data[1] == 0x4D {
		return "image/bmp"
	}
	return "image/png"
}

func languageNameFromCode(code string) string {
	names := map[string]string{
		"en": "English", "es": "Spanish", "fr": "French", "de": "German",
		"it": "Italian", "pt": "Portuguese", "ru": "Russian", "zh": "Chinese",
		"ja": "Japanese", "ko": "Korean", "ar": "Arabic", "hi": "Hindi",
		"nl": "Dutch", "pl": "Polish", "sv": "Swedish", "tr": "Turkish",
		"vi": "Vietnamese", "th": "Thai", "uk": "Ukrainian", "cs": "Czech",
		"ro": "Romanian", "hu": "Hungarian", "el": "Greek", "da": "Danish",
		"fi": "Finnish", "no": "Norwegian", "bg": "Bulgarian", "hr": "Croatian",
		"sk": "Slovak", "sl": "Slovenian", "et": "Estonian", "lv": "Latvian",
		"lt": "Lithuanian", "id": "Indonesian", "ms": "Malay", "he": "Hebrew",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

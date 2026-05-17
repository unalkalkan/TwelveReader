package provider

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// OpenAITTSProvider implements TTSProvider using OpenAI-compatible TTS APIs
type OpenAITTSProvider struct {
	name           string
	config         types.TTSProviderConfig
	httpClient     *http.Client
	model          string
	maxRetries     int
	retryBackoffMs int
	responseFormat string
	maxNewTokens   int
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

	// Configure timeout from options or use default (5 minutes)
	timeout := 300 * time.Second // TTS can take longer than LLM calls
	if timeoutStr, ok := config.Options["timeout"]; ok {
		var timeoutSec int
		if _, err := fmt.Sscanf(timeoutStr, "%d", &timeoutSec); err == nil && timeoutSec > 0 {
			timeout = time.Duration(timeoutSec) * time.Second
		}
	}

	maxRetries, retryBackoffMs := parseRetryOptions(config.Options)
	responseFormat := strings.TrimSpace(config.Options["response_format"])
	if responseFormat == "" {
		responseFormat = "wav"
	}
	maxNewTokens := 0
	if raw := strings.TrimSpace(config.Options["max_new_tokens"]); raw != "" {
		fmt.Sscanf(raw, "%d", &maxNewTokens)
	}

	return &OpenAITTSProvider{
		name:           config.Name,
		config:         config,
		httpClient:     &http.Client{Timeout: timeout},
		model:          model,
		maxRetries:     maxRetries,
		retryBackoffMs: retryBackoffMs,
		responseFormat: responseFormat,
		maxNewTokens:   maxNewTokens,
	}, nil
}

func (o *OpenAITTSProvider) Name() string {
	return o.name
}

// Synthesize converts text to speech using OpenAI-compatible API
func (o *OpenAITTSProvider) Synthesize(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	chunks := splitTextForTTS(req.Text, o.config.MaxSegmentSize)
	if len(chunks) == 0 {
		chunks = []string{req.Text}
	}

	var audioChunks [][]byte
	var format string
	for i, chunk := range chunks {
		apiReq := ttsAPIRequest{
			Model:          o.model,
			Input:          chunk,
			Voice:          req.VoiceID,
			ResponseFormat: o.responseFormat,
		}
		if o.maxNewTokens > 0 {
			apiReq.MaxNewTokens = o.maxNewTokens
		}
		if req.Language != "" {
			apiReq.Language = normalizeTTSLanguage(req.Language)
		}
		if req.VoiceDescription != "" {
			apiReq.Instructions = req.VoiceDescription
		}

		audioData, detectedFormat, err := o.callTTSAPI(ctx, apiReq)
		if err != nil {
			return nil, fmt.Errorf("failed to call TTS API for chunk %d/%d: %w", i+1, len(chunks), err)
		}
		if format == "" {
			format = detectedFormat
		}
		if detectedFormat != format {
			return nil, fmt.Errorf("TTS chunks returned mixed formats: %s then %s", format, detectedFormat)
		}
		audioChunks = append(audioChunks, audioData)
	}

	audioData := audioChunks[0]
	if len(audioChunks) > 1 {
		var err error
		switch format {
		case "wav":
			audioData, err = concatWAV(audioChunks)
		default:
			return nil, fmt.Errorf("cannot concatenate %d TTS chunks with format %s", len(audioChunks), format)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to concatenate TTS chunks: %w", err)
		}
	}

	return &TTSResponse{
		AudioData:  audioData,
		Format:     format,
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
		log.Printf("[TTS-%s] Failed to create request: %v", o.name, err)
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

	log.Printf("[TTS-%s] Request: GET %s", o.name, httpReq.URL.String())

	// Execute request
	startTime := time.Now()
	resp, err := o.httpClient.Do(httpReq)
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("[TTS-%s] Request failed after %v: %v", o.name, duration, err)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[TTS-%s] Response: %d %s (took %v)", o.name, resp.StatusCode, resp.Status, duration)

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[TTS-%s] Failed to read response body: %v", o.name, err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[TTS-%s] Response payload: %s", o.name, truncateString(string(body), 500))

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse as error response
		var errResp ttsAPIErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			log.Printf("[TTS-%s] API error: %s (type: %s, code: %s)", o.name, errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var apiResp voicesAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("[TTS-%s] Failed to parse response JSON: %v", o.name, err)
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

	log.Printf("[TTS-%s] Parsed %d voices from response", o.name, len(voices))
	return voices, nil
}

func (o *OpenAITTSProvider) Close() error {
	// Close HTTP client connections
	o.httpClient.CloseIdleConnections()
	return nil
}

// ttsAPIRequest represents the OpenAI TTS API request structure
type ttsAPIRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	Instructions   string `json:"instructions,omitempty"`
	Language       string `json:"language,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	MaxNewTokens   int    `json:"max_new_tokens,omitempty"`
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
func (o *OpenAITTSProvider) callTTSAPI(ctx context.Context, req ttsAPIRequest) ([]byte, string, error) {
	// Encode request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build endpoint URL
	endpoint := o.config.Endpoint
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	endpoint += "audio/speech"

	log.Printf("[TTS-%s] Request: POST %s", o.name, endpoint)
	log.Printf("[TTS-%s] Request payload: model=%s, voice=%s, input_length=%d chars", o.name, req.Model, req.Voice, len(req.Input))
	log.Printf("[TTS-%s] Request input (truncated): %s", o.name, truncateString(req.Input, 200))

	var body []byte
	for attempt := 0; attempt <= o.maxRetries; attempt++ {
		httpReq, err := newJSONPostRequest(ctx, endpoint, jsonData, o.config.APIKey)
		if err != nil {
			log.Printf("[TTS-%s] Failed to create request: %v", o.name, err)
			return nil, "", fmt.Errorf("failed to create request: %w", err)
		}

		startTime := time.Now()
		resp, err := o.httpClient.Do(httpReq)
		duration := time.Since(startTime)
		if err != nil {
			log.Printf("[TTS-%s] Request attempt %d/%d failed after %v: %v", o.name, attempt+1, o.maxRetries+1, duration, err)
			if attempt < o.maxRetries {
				if waitErr := sleepBeforeRetry(ctx, computeBackoff(attempt, o.retryBackoffMs)); waitErr != nil {
					return nil, "", fmt.Errorf("failed to execute request: %w", err)
				}
				continue
			}
			return nil, "", fmt.Errorf("failed to execute request: %w", err)
		}

		log.Printf("[TTS-%s] Response attempt %d/%d: %d %s (took %v)", o.name, attempt+1, o.maxRetries+1, resp.StatusCode, resp.Status, duration)

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[TTS-%s] Failed to read response body: %v", o.name, err)
			return nil, "", fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		if isRetryableStatusCode(resp.StatusCode) && attempt < o.maxRetries {
			log.Printf("[TTS-%s] Retryable API status %d; retrying after backoff", o.name, resp.StatusCode)
			if waitErr := sleepBeforeRetry(ctx, computeBackoff(attempt, o.retryBackoffMs)); waitErr != nil {
				return nil, "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			}
			continue
		}

		// Try to parse as error response
		var errResp ttsAPIErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			log.Printf("[TTS-%s] API error: %s (type: %s, code: %s)", o.name, errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
			return nil, "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		log.Printf("[TTS-%s] API request failed: %s", o.name, truncateString(string(body), 500))
		return nil, "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	format := audioFormatFromBytes(body)
	log.Printf("[TTS-%s] Response payload: audio_size=%d bytes, detected_format=%s", o.name, len(body), format)
	return body, format, nil
}

func splitTextForTTS(text string, maxChars int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxChars <= 0 || utf8.RuneCountInString(text) <= maxChars {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		chunks = append(chunks, strings.TrimSpace(current.String()))
		current.Reset()
	}

	for _, word := range words {
		for utf8.RuneCountInString(word) > maxChars {
			flush()
			prefix, rest := splitAtRuneLimit(word, maxChars)
			chunks = append(chunks, prefix)
			word = rest
		}
		if word == "" {
			continue
		}
		candidate := word
		if current.Len() > 0 {
			candidate = current.String() + " " + word
		}
		if utf8.RuneCountInString(candidate) > maxChars {
			flush()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(word)
	}
	flush()

	return splitChunksAtSentenceBoundaries(chunks, maxChars)
}

func splitChunksAtSentenceBoundaries(chunks []string, maxChars int) []string {
	var out []string
	endSentence := regexp.MustCompile(`(?s)^(.+[.!?])\s+(.+)$`)
	for _, chunk := range chunks {
		if utf8.RuneCountInString(chunk) <= maxChars {
			out = append(out, chunk)
			continue
		}
		parts := endSentence.FindStringSubmatch(chunk)
		if len(parts) == 3 && utf8.RuneCountInString(parts[1]) <= maxChars && utf8.RuneCountInString(parts[2]) <= maxChars {
			out = append(out, strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]))
			continue
		}
		out = append(out, chunk)
	}
	return out
}

func splitAtRuneLimit(s string, limit int) (string, string) {
	if limit <= 0 {
		return "", s
	}
	count := 0
	for idx := range s {
		if count == limit {
			return s[:idx], s[idx:]
		}
		count++
	}
	return s, ""
}

func normalizeTTSLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "en", "eng":
		return "English"
	case "zh", "zh-cn", "zh-tw", "ch", "chi", "zho":
		return "Chinese"
	case "ja", "jp", "jpn":
		return "Japanese"
	case "ko", "kor":
		return "Korean"
	case "fr", "fra", "fre":
		return "French"
	case "de", "deu", "ger":
		return "German"
	case "it", "ita":
		return "Italian"
	case "pt", "por":
		return "Portuguese"
	case "ru", "rus":
		return "Russian"
	case "es", "spa":
		return "Spanish"
	case "", "auto":
		return "Auto"
	default:
		return language
	}
}

func concatWAV(chunks [][]byte) ([]byte, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no wav chunks")
	}
	if len(chunks) == 1 {
		return chunks[0], nil
	}

	first, err := parseSimpleWAV(chunks[0])
	if err != nil {
		return nil, err
	}
	var data bytes.Buffer
	data.Write(first.data)
	for i, chunk := range chunks[1:] {
		parsed, err := parseSimpleWAV(chunk)
		if err != nil {
			return nil, fmt.Errorf("chunk %d: %w", i+2, err)
		}
		if !bytes.Equal(parsed.fmtChunk, first.fmtChunk) {
			return nil, fmt.Errorf("chunk %d has different WAV format", i+2)
		}
		data.Write(parsed.data)
	}

	dataBytes := data.Bytes()
	out := make([]byte, 44+len(dataBytes))
	copy(out[0:4], "RIFF")
	binary.LittleEndian.PutUint32(out[4:8], uint32(36+len(dataBytes)))
	copy(out[8:12], "WAVE")
	copy(out[12:16], "fmt ")
	binary.LittleEndian.PutUint32(out[16:20], 16)
	copy(out[20:36], first.fmtChunk)
	copy(out[36:40], "data")
	binary.LittleEndian.PutUint32(out[40:44], uint32(len(dataBytes)))
	copy(out[44:], dataBytes)
	return out, nil
}

type simpleWAV struct {
	fmtChunk []byte
	data     []byte
}

func parseSimpleWAV(body []byte) (*simpleWAV, error) {
	if len(body) < 44 || string(body[0:4]) != "RIFF" || string(body[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a RIFF/WAVE file")
	}
	if string(body[12:16]) != "fmt " {
		return nil, fmt.Errorf("unsupported WAV layout: missing fmt chunk")
	}
	fmtLen := int(binary.LittleEndian.Uint32(body[16:20]))
	if fmtLen < 16 || len(body) < 20+fmtLen+8 {
		return nil, fmt.Errorf("invalid WAV fmt chunk")
	}
	dataHeader := 20 + fmtLen
	if string(body[dataHeader:dataHeader+4]) != "data" {
		return nil, fmt.Errorf("unsupported WAV layout: missing data chunk")
	}
	dataLen := int(binary.LittleEndian.Uint32(body[dataHeader+4 : dataHeader+8]))
	dataStart := dataHeader + 8
	if len(body) < dataStart+dataLen {
		return nil, fmt.Errorf("truncated WAV data")
	}
	return &simpleWAV{
		fmtChunk: append([]byte(nil), body[20:20+fmtLen]...),
		data:     append([]byte(nil), body[dataStart:dataStart+dataLen]...),
	}, nil
}

func audioFormatFromBytes(body []byte) string {
	if len(body) >= 12 && string(body[0:4]) == "RIFF" && string(body[8:12]) == "WAVE" {
		return "wav"
	}
	if len(body) >= 3 && string(body[0:3]) == "ID3" {
		return "mp3"
	}
	if len(body) >= 2 && body[0] == 0xFF && body[1]&0xE0 == 0xE0 {
		return "mp3"
	}
	if len(body) >= 4 && string(body[0:4]) == "OggS" {
		return "ogg"
	}
	if len(body) >= 4 && string(body[0:4]) == "fLaC" {
		return "flac"
	}
	return "mp3"
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

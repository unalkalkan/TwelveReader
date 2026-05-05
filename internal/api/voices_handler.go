package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
)

const defaultVoicePreviewText = `In my life, why do I give valuable time
To people who don't care if I live or die?`

var safeVoiceSamplePartRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// VoicesHandler handles TTS voice-related API endpoints
type VoicesHandler struct {
	providerReg *provider.Registry
	sampleStore storage.SampleStore
	sampleLocks sync.Map
}

// NewVoicesHandler creates a new voices handler
func NewVoicesHandler(providerReg *provider.Registry) *VoicesHandler {
	return &VoicesHandler{
		providerReg: providerReg,
	}
}

// NewVoicesHandlerWithSampleStorage creates a voices handler with persistent preview sample storage.
func NewVoicesHandlerWithSampleStorage(providerReg *provider.Registry, sampleStore storage.SampleStore) *VoicesHandler {
	return &VoicesHandler{
		providerReg: providerReg,
		sampleStore: sampleStore,
	}
}

// VoiceResponse represents a voice in the API response
type VoiceResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Languages   []string `json:"languages"`
	Gender      string   `json:"gender,omitempty"`
	Accent      string   `json:"accent,omitempty"`
	Description string   `json:"description,omitempty"`
	Provider    string   `json:"provider"`
}

type VoicePreviewRequest struct {
	Provider         string `json:"provider"`
	VoiceID          string `json:"voice_id"`
	Text             string `json:"text"`
	Language         string `json:"language,omitempty"`
	VoiceDescription string `json:"voice_description,omitempty"`
}

type VoicePreviewResponse struct {
	AudioBase64 string `json:"audio_base64"`
	MimeType    string `json:"mime_type"`
	Format      string `json:"format"`
	Cached      bool   `json:"cached"`
}

// ListVoices handles GET /api/v1/voices
func (h *VoicesHandler) ListVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get optional provider query parameter
	providerName := r.URL.Query().Get("provider")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var allVoices []VoiceResponse

	// If provider is specified, get voices from that provider only
	if providerName != "" {
		ttsProvider, err := h.providerReg.GetTTS(providerName)
		if err != nil {
			respondError(w, fmt.Sprintf("Provider '%s' not found: %v", providerName, err), http.StatusNotFound)
			return
		}

		voices, err := ttsProvider.ListVoices(ctx)
		if err != nil {
			log.Printf("Failed to get voices from provider %s: %v", providerName, err)
			respondError(w, fmt.Sprintf("Failed to get voices from provider: %v", err), http.StatusInternalServerError)
			return
		}

		for _, v := range voices {
			allVoices = append(allVoices, VoiceResponse{
				ID:          v.ID,
				Name:        v.Name,
				Languages:   v.Languages,
				Gender:      v.Gender,
				Accent:      v.Accent,
				Description: v.Description,
				Provider:    providerName,
			})
		}
	} else {
		// Get voices from all TTS providers
		ttsProviders := h.providerReg.ListTTS()
		if len(ttsProviders) == 0 {
			respondError(w, "No TTS providers configured", http.StatusServiceUnavailable)
			return
		}

		for _, provName := range ttsProviders {
			ttsProvider, err := h.providerReg.GetTTS(provName)
			if err != nil {
				log.Printf("Failed to get TTS provider %s: %v", provName, err)
				continue
			}

			voices, err := ttsProvider.ListVoices(ctx)
			if err != nil {
				log.Printf("Failed to get voices from provider %s: %v", provName, err)
				// Continue with other providers instead of failing completely
				continue
			}

			for _, v := range voices {
				allVoices = append(allVoices, VoiceResponse{
					ID:          v.ID,
					Name:        v.Name,
					Languages:   v.Languages,
					Gender:      v.Gender,
					Accent:      v.Accent,
					Description: v.Description,
					Provider:    provName,
				})
			}
		}
	}

	// Return the voices list
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"voices": allVoices,
		"count":  len(allVoices),
	}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// PreviewVoice handles POST /api/v1/voices/preview
func (h *VoicesHandler) PreviewVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VoicePreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Provider == "" {
		respondError(w, "provider is required", http.StatusBadRequest)
		return
	}
	if req.VoiceID == "" {
		respondError(w, "voice_id is required", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		respondError(w, "text is required", http.StatusBadRequest)
		return
	}

	ttsProvider, err := h.providerReg.GetTTS(req.Provider)
	if err != nil {
		respondError(w, fmt.Sprintf("Provider '%s' not found: %v", req.Provider, err), http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	sample, err := h.getOrCreateVoiceSample(ctx, ttsProvider, req.Provider, req.VoiceID, req.Language, req.VoiceDescription)
	if err != nil {
		respondError(w, fmt.Sprintf("Failed to synthesize preview: %v", err), http.StatusInternalServerError)
		return
	}

	resp := VoicePreviewResponse{
		AudioBase64: base64.StdEncoding.EncodeToString(sample.AudioData),
		MimeType:    audioMimeType(sample.Format),
		Format:      sample.Format,
		Cached:      sample.Cached,
	}

	respondJSON(w, resp, http.StatusOK)
}

type voiceSample struct {
	AudioData []byte
	Format    string
	Cached    bool
}

// PreGenerateVoiceSamples creates missing persistent preview samples for every available TTS voice.
func (h *VoicesHandler) PreGenerateVoiceSamples(ctx context.Context) error {
	if h.sampleStore == nil {
		return nil
	}

	for _, providerName := range h.providerReg.ListTTS() {
		ttsProvider, err := h.providerReg.GetTTS(providerName)
		if err != nil {
			log.Printf("Failed to get TTS provider %s for sample generation: %v", providerName, err)
			continue
		}

		voices, err := ttsProvider.ListVoices(ctx)
		if err != nil {
			log.Printf("Failed to list voices from provider %s for sample generation: %v", providerName, err)
			continue
		}

		for _, voice := range voices {
			language := ""
			if len(voice.Languages) > 0 {
				language = voice.Languages[0]
			}
			if _, err := h.getOrCreateVoiceSample(ctx, ttsProvider, providerName, voice.ID, language, voice.Description); err != nil {
				log.Printf("Failed to generate sample for voice %s/%s: %v", providerName, voice.ID, err)
			}
		}
	}
	return nil
}

func (h *VoicesHandler) getOrCreateVoiceSample(ctx context.Context, ttsProvider provider.TTSProvider, providerName, voiceID, language, voiceDescription string) (*voiceSample, error) {
	if h.sampleStore == nil {
		resp, err := synthesizeVoiceSample(ctx, ttsProvider, voiceID, language, voiceDescription)
		if err != nil {
			return nil, err
		}
		return &voiceSample{AudioData: resp.AudioData, Format: resp.Format, Cached: false}, nil
	}

	key := h.voiceSampleKey(providerName, voiceID, language, voiceDescription)
	lockIface, _ := h.sampleLocks.LoadOrStore(key, &sync.Mutex{})
	lock := lockIface.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	format, audioData, ok, err := h.loadVoiceSample(ctx, key)
	if err != nil {
		return nil, err
	}
	if ok {
		return &voiceSample{AudioData: audioData, Format: format, Cached: true}, nil
	}

	resp, err := synthesizeVoiceSample(ctx, ttsProvider, voiceID, language, voiceDescription)
	if err != nil {
		return nil, err
	}
	path := voiceSamplePathForKey(key, resp.Format)
	if err := h.sampleStore.Put(ctx, path, resp.AudioData); err != nil {
		return nil, fmt.Errorf("failed to store voice sample: %w", err)
	}
	return &voiceSample{AudioData: resp.AudioData, Format: resp.Format, Cached: false}, nil
}

func synthesizeVoiceSample(ctx context.Context, ttsProvider provider.TTSProvider, voiceID, language, voiceDescription string) (*provider.TTSResponse, error) {
	return ttsProvider.Synthesize(ctx, provider.TTSRequest{
		Text:             defaultVoicePreviewText,
		VoiceID:          voiceID,
		Language:         language,
		VoiceDescription: voiceDescription,
	})
}

func (h *VoicesHandler) loadVoiceSample(ctx context.Context, key string) (string, []byte, bool, error) {
	for _, format := range []string{"wav", "mp3", "ogg", "flac"} {
		path := voiceSamplePathForKey(key, format)
		exists, err := h.sampleStore.Exists(ctx, path)
		if err != nil {
			return "", nil, false, err
		}
		if !exists {
			continue
		}
		data, err := h.sampleStore.Get(ctx, path)
		if err != nil {
			return "", nil, false, err
		}
		return format, data, true, nil
	}
	return "", nil, false, nil
}

func (h *VoicesHandler) voiceSampleKey(providerName, voiceID, language, voiceDescription string) string {
	base := fmt.Sprintf("%s_%s_%s", safeVoiceSamplePart(providerName), safeVoiceSamplePart(voiceID), safeVoiceSamplePart(language))
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(voiceDescription))
	return fmt.Sprintf("%s_%08x", base, hasher.Sum32())
}

func safeVoiceSamplePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	value = safeVoiceSamplePartRe.ReplaceAllString(value, "_")
	value = strings.Trim(value, "._-")
	if value == "" {
		return "default"
	}
	if len(value) > 80 {
		value = value[:80]
	}
	return value
}

func voiceSamplePathForKey(key, format string) string {
	return filepath.Join("voice-samples", fmt.Sprintf("%s.%s", key, format))
}

func audioMimeType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "ogg":
		return "audio/ogg"
	case "flac":
		return "audio/flac"
	default:
		return "application/octet-stream"
	}
}

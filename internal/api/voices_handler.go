package api

import (
	"encoding/base64"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/provider"
)

// VoicesHandler handles TTS voice-related API endpoints
type VoicesHandler struct {
	providerReg *provider.Registry
}

// NewVoicesHandler creates a new voices handler
func NewVoicesHandler(providerReg *provider.Registry) *VoicesHandler {
	return &VoicesHandler{
		providerReg: providerReg,
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

	ttsResp, err := ttsProvider.Synthesize(ctx, provider.TTSRequest{
		Text:             req.Text,
		VoiceID:          req.VoiceID,
		Language:         req.Language,
		VoiceDescription: req.VoiceDescription,
	})
	if err != nil {
		respondError(w, fmt.Sprintf("Failed to synthesize preview: %v", err), http.StatusInternalServerError)
		return
	}

	resp := VoicePreviewResponse{
		AudioBase64: base64.StdEncoding.EncodeToString(ttsResp.AudioData),
		MimeType:    audioMimeType(ttsResp.Format),
		Format:      ttsResp.Format,
	}

	respondJSON(w, resp, http.StatusOK)
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

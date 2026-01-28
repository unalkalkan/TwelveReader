package api

import (
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

// ListVoices handles GET /api/v1/voices
func (h *VoicesHandler) ListVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get optional query parameters
	providerName := r.URL.Query().Get("provider")
	model := r.URL.Query().Get("model")

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

		voices, err := ttsProvider.ListVoices(ctx, model)
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

			voices, err := ttsProvider.ListVoices(ctx, model)
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

package provider

import (
	"fmt"
	"sync"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Registry manages provider instances
type Registry struct {
	llmProviders map[string]LLMProvider
	ttsProviders map[string]TTSProvider
	ocrProviders map[string]OCRProvider
	mu           sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		llmProviders: make(map[string]LLMProvider),
		ttsProviders: make(map[string]TTSProvider),
		ocrProviders: make(map[string]OCRProvider),
	}
}

// RegisterLLM registers an LLM provider
func (r *Registry) RegisterLLM(provider LLMProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.llmProviders[name]; exists {
		return fmt.Errorf("LLM provider already registered: %s", name)
	}

	r.llmProviders[name] = provider
	return nil
}

// RegisterTTS registers a TTS provider
func (r *Registry) RegisterTTS(provider TTSProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.ttsProviders[name]; exists {
		return fmt.Errorf("TTS provider already registered: %s", name)
	}

	r.ttsProviders[name] = provider
	return nil
}

// RegisterOCR registers an OCR provider
func (r *Registry) RegisterOCR(provider OCRProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.ocrProviders[name]; exists {
		return fmt.Errorf("OCR provider already registered: %s", name)
	}

	r.ocrProviders[name] = provider
	return nil
}

// GetLLM retrieves an LLM provider by name
func (r *Registry) GetLLM(name string) (LLMProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.llmProviders[name]
	if !exists {
		return nil, fmt.Errorf("LLM provider not found: %s", name)
	}

	return provider, nil
}

// GetTTS retrieves a TTS provider by name
func (r *Registry) GetTTS(name string) (TTSProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.ttsProviders[name]
	if !exists {
		return nil, fmt.Errorf("TTS provider not found: %s", name)
	}

	return provider, nil
}

// GetOCR retrieves an OCR provider by name
func (r *Registry) GetOCR(name string) (OCRProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.ocrProviders[name]
	if !exists {
		return nil, fmt.Errorf("OCR provider not found: %s", name)
	}

	return provider, nil
}

// ListLLM returns all registered LLM provider names
func (r *Registry) ListLLM() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.llmProviders))
	for name := range r.llmProviders {
		names = append(names, name)
	}
	return names
}

// ListTTS returns all registered TTS provider names
func (r *Registry) ListTTS() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.ttsProviders))
	for name := range r.ttsProviders {
		names = append(names, name)
	}
	return names
}

// ListOCR returns all registered OCR provider names
func (r *Registry) ListOCR() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.ocrProviders))
	for name := range r.ocrProviders {
		names = append(names, name)
	}
	return names
}

// Close closes all registered providers
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error

	// Close LLM providers
	for name, provider := range r.llmProviders {
		if err := provider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close LLM provider %s: %w", name, err))
		}
	}

	// Close TTS providers
	for name, provider := range r.ttsProviders {
		if err := provider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close TTS provider %s: %w", name, err))
		}
	}

	// Close OCR providers
	for name, provider := range r.ocrProviders {
		if err := provider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close OCR provider %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %v", errs)
	}

	return nil
}

// InitializeProviders creates provider instances from configuration
func (r *Registry) InitializeProviders(cfg types.ProvidersConfig) error {
	// Initialize LLM providers
	for _, llmCfg := range cfg.LLM {
		if !llmCfg.Enabled {
			continue
		}
		// Create OpenAI-compatible provider if endpoint is configured
		var provider LLMProvider
		var err error
		if llmCfg.Endpoint != "" && llmCfg.Model != "" {
			provider, err = NewOpenAILLMProvider(llmCfg)
			if err != nil {
				return fmt.Errorf("failed to create OpenAI LLM provider %s: %w", llmCfg.Name, err)
			}
		} else {
			// Fallback to stub provider for backward compatibility
			provider = NewStubLLMProvider(llmCfg)
		}
		if err := r.RegisterLLM(provider); err != nil {
			return err
		}
	}

	// Initialize TTS providers
	for _, ttsCfg := range cfg.TTS {
		if !ttsCfg.Enabled {
			continue
		}
		// Create stub provider for now
		provider := NewStubTTSProvider(ttsCfg)
		if err := r.RegisterTTS(provider); err != nil {
			return err
		}
	}

	// Initialize OCR providers
	for _, ocrCfg := range cfg.OCR {
		if !ocrCfg.Enabled {
			continue
		}
		// Create stub provider for now
		provider := NewStubOCRProvider(ocrCfg)
		if err := r.RegisterOCR(provider); err != nil {
			return err
		}
	}

	return nil
}

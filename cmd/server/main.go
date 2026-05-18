package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/api"
	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/config"
	"github.com/unalkalkan/TwelveReader/internal/features"
	"github.com/unalkalkan/TwelveReader/internal/health"
	"github.com/unalkalkan/TwelveReader/internal/parser"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

const version = "0.1.0-milestone4"

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config/dev.example.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting TwelveReader Server v%s (environment: %s)", version, cfg.Environment)
	log.Printf("Configuration loaded from: %s", *configPath)

	// Initialize storage adapter
	storageAdapter, err := storage.NewAdapter(cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()
	log.Printf("Storage adapter initialized: %s", cfg.Storage.Adapter)

	// Initialize provider registry
	providerRegistry := provider.NewRegistry()
	if err := providerRegistry.InitializeProviders(cfg.Providers); err != nil {
		log.Fatalf("Failed to initialize providers: %v", err)
	}
	defer providerRegistry.Close()

	log.Printf("Providers initialized:")
	log.Printf("  LLM: %v", providerRegistry.ListLLM())
	log.Printf("  TTS: %v", providerRegistry.ListTTS())
	log.Printf("  OCR: %v", providerRegistry.ListOCR())

	// Initialize book repository
	bookRepo := book.NewRepository(storageAdapter)
	log.Printf("Book repository initialized")

	// Initialize parser factory
	parserFactory := parser.NewFactory()
	log.Printf("Parser factory initialized")

	// Initialize health checks
	healthHandler := health.NewHandler(version)

	// Register health checks
	healthHandler.Register("storage", func(ctx context.Context) (health.Status, error) {
		// Check if storage is accessible
		exists, err := storageAdapter.Exists(ctx, ".healthcheck")
		if err != nil {
			return health.StatusUnhealthy, err
		}
		_ = exists // Ignore result, just checking connectivity
		return health.StatusHealthy, nil
	})

	healthHandler.Register("providers", func(ctx context.Context) (health.Status, error) {
		// Check if at least one provider of each type is registered
		if len(providerRegistry.ListLLM()) == 0 && len(providerRegistry.ListTTS()) == 0 {
			return health.StatusDegraded, fmt.Errorf("no providers registered")
		}
		return health.StatusHealthy, nil
	})

	// Initialize feature flags (Milestone 0)
	featureStore := features.NewStore(map[string]bool{
		"saas_auth":      false,
		"usage_metering": false,
		"quota_engine":   false,
		"repository_pub": false,
		"user_accounts":  false,
		"billing":        false,
	})
	log.Printf("Feature flags initialized")

	// Create V1 system handler (Milestone 0)
	v1System := api.NewV1SystemHandler(
		healthHandler,
		providerRegistry,
		featureStore,
		version,
		cfg.Environment,
		cfg.Storage.Adapter,
		cfg.Pipeline.WorkerPoolSize,
	)

	// Request ID middleware (Milestone 0): applied to ALL /api/v1 routes via sub-mux
	reqCtx := &api.RequestContext{}

	// Set up HTTP server and routes
	mux := http.NewServeMux()

	// Health endpoints (legacy, non-versioned)
	mux.HandleFunc("/health/live", healthHandler.LivenessHandler())
	mux.HandleFunc("/health/ready", healthHandler.ReadinessHandler())
	mux.HandleFunc("/health", healthHandler.HealthHandler())

	// --- /api/v1 sub-mux (all routes get request ID middleware) ---
	v1Mux := http.NewServeMux()

	// Milestone 0: Versioned system endpoints
	v1Mux.HandleFunc("/api/v1/health", v1System.HealthHandler())
	v1Mux.HandleFunc("/api/v1/server-info", v1System.ServerInfoHandler())
	v1Mux.HandleFunc("/api/v1/features", v1System.FeaturesHandler())

	// API endpoints (stubs for now)
	v1Mux.HandleFunc("/api/v1/info", infoHandler(version, cfg))
	v1Mux.HandleFunc("/api/v1/providers", providersHandler(providerRegistry))

	// Voices API endpoint (Milestone 4)
	voicesHandler := api.NewVoicesHandlerWithRepositoryAndSampleStorage(providerRegistry, bookRepo, storage.NewAdapterSampleStore(storageAdapter))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := voicesHandler.PreGenerateVoiceSamples(ctx); err != nil {
			log.Printf("Failed to pre-generate voice samples: %v", err)
		}
	}()
	v1Mux.HandleFunc("/api/v1/voices", voicesHandler.ListVoices)
	v1Mux.HandleFunc("/api/v1/voices/default", voicesHandler.DefaultVoice)
	v1Mux.HandleFunc("/api/v1/voices/preview", voicesHandler.PreviewVoice)

	// Book API endpoints (Milestone 3)
	bookHandler := api.NewBookHandler(bookRepo, parserFactory, providerRegistry, storageAdapter)
	debugHandler := api.NewDebugHandler(bookRepo, storageAdapter)
	v1Mux.HandleFunc("/api/v1/books", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			bookHandler.UploadBook(w, r)
			return
		}
		if r.Method == http.MethodGet {
			bookHandler.ListBooks(w, r)
			return
		}
		api.WriteMethodNotAllowedError(w, r)
	})
	v1Mux.HandleFunc("/api/v1/books/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if r.Method == http.MethodDelete {
			bookHandler.DeleteBook(w, r)
		} else if strings.HasSuffix(path, "/status") {
			bookHandler.GetBookStatus(w, r)
		} else if strings.HasSuffix(path, "/segments") {
			bookHandler.ListSegments(w, r)
		} else if strings.HasSuffix(path, "/voice-map") {
			if r.Method == http.MethodPost {
				bookHandler.SetVoiceMap(w, r)
			} else {
				bookHandler.GetVoiceMap(w, r)
			}
		} else if strings.HasSuffix(path, "/stream") {
			bookHandler.StreamSegments(w, r)
		} else if strings.HasSuffix(path, "/download") {
			bookHandler.DownloadBook(w, r)
		} else if strings.Contains(path, "/pipeline/status") {
			bookHandler.GetPipelineStatus(w, r)
		} else if strings.HasSuffix(path, "/personas") {
			bookHandler.GetPersonas(w, r)
		} else if strings.Contains(path, "/audio/") {
			bookHandler.GetAudio(w, r)
		} else {
			bookHandler.GetBook(w, r)
		}
	})
	v1Mux.HandleFunc("/api/v1/debug/events", debugHandler.Events)
	v1Mux.HandleFunc("/api/v1/debug/stream", debugHandler.EventStream)
	v1Mux.HandleFunc("/api/v1/debug/books/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/synth-jobs") {
			debugHandler.ListSynthJobs(w, r)
		} else if strings.HasSuffix(path, "/audio-validation") {
			debugHandler.AudioValidation(w, r)
		} else if strings.HasSuffix(path, "/playback-events") {
			debugHandler.PlaybackEvents(w, r)
		} else if strings.HasSuffix(path, "/user-progress") {
			debugHandler.UserProgress(w, r)
		} else if strings.HasSuffix(path, "/events") {
			debugHandler.Events(w, r)
		} else if strings.HasSuffix(path, "/stream") {
			debugHandler.EventStream(w, r)
		} else {
			respondDebugNotFoundStructured(w, r)
		}
	})

	// Mount /api/v1 sub-mux behind request ID middleware
	mux.Handle("/api/v1", reqCtx.Middleware(v1Mux))

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// infoHandler returns basic server information
func infoHandler(version string, cfg *types.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[INFO] GET /api/v1/info - Returning server info (version: %s, storage: %s)", version, cfg.Storage.Adapter)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"version":"%s","storage_adapter":"%s"}`, version, cfg.Storage.Adapter)
	}
}

// providersHandler returns information about registered providers
func providersHandler(registry *provider.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		llm := registry.ListLLM()
		tts := registry.ListTTS()
		ocr := registry.ListOCR()
		log.Printf("[PROVIDERS] GET /api/v1/providers - LLM: %v, TTS: %v, OCR: %v", llm, tts, ocr)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"llm":%v,"tts":%v,"ocr":%v}`,
			toJSON(llm),
			toJSON(tts),
			toJSON(ocr))
	}
}

// respondDebugNotFound writes the legacy debug not-found response.
func respondDebugNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `{"error":"Debug endpoint not found"}`)
}

// respondDebugNotFoundStructured writes a structured 404 error response with request ID.
func respondDebugNotFoundStructured(w http.ResponseWriter, r *http.Request) {
	api.WriteNotFoundError(w, r, "Debug endpoint")
}

func toJSON(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	result := "["
	for i, item := range items {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s"`, item)
	}
	result += "]"
	return result
}

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

	log.Printf("Starting TwelveReader Server v%s", version)
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

	// Set up HTTP server and routes
	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("/health/live", healthHandler.LivenessHandler())
	mux.HandleFunc("/health/ready", healthHandler.ReadinessHandler())
	mux.HandleFunc("/health", healthHandler.HealthHandler())

	// API endpoints (stubs for now)
	mux.HandleFunc("/api/v1/info", infoHandler(version, cfg))
	mux.HandleFunc("/api/v1/providers", providersHandler(providerRegistry))

	// Book API endpoints (Milestone 3)
	bookHandler := api.NewBookHandler(bookRepo, parserFactory, providerRegistry, storageAdapter)
	mux.HandleFunc("/api/v1/books", bookHandler.UploadBook)
	mux.HandleFunc("/api/v1/books/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/status") {
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
		} else if strings.Contains(path, "/audio/") {
			bookHandler.GetAudio(w, r)
		} else {
			bookHandler.GetBook(w, r)
		}
	})

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
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"version":"%s","storage_adapter":"%s"}`, version, cfg.Storage.Adapter)
	}
}

// providersHandler returns information about registered providers
func providersHandler(registry *provider.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"llm":%v,"tts":%v,"ocr":%v}`,
			toJSON(registry.ListLLM()),
			toJSON(registry.ListTTS()),
			toJSON(registry.ListOCR()))
	}
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

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
	"github.com/unalkalkan/TwelveReader/internal/identity"
	"github.com/unalkalkan/TwelveReader/internal/parser"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"

	_ "modernc.org/sqlite" // Register SQLite driver
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

	// Initialize feature flags from config (Milestone 0)
	// Config loader applies: environment defaults -> YAML overrides -> env var overrides
	featureStore := features.NewStore(cfg.FeatureFlags)
	log.Printf("Feature flags initialized: %+v", cfg.FeatureFlags)

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

	// Create readiness smoke visibility handler (Milestone 0, Work 0.5)
	readinessHandler := api.NewReadinessHandler(
		v1System,
		healthHandler,
		featureStore,
		version,
		cfg.Environment,
	)

	// Initialize identity/auth (Milestone 1: Identity, Sessions, and Ownership)
	identityDBPath := cfg.Auth.IdentityDBPath
	if identityDBPath == "" {
		identityDBPath = "data/identity.db"
	}
	identityPool, err := identity.NewDBPool(identityDBPath)
	if err != nil {
		log.Fatalf("Failed to initialize identity DB: %v", err)
	}
	defer identityPool.Close()
	log.Printf("Identity DB initialized: %s", identityDBPath)

	// Parse auth durations with defaults
	sessionTTL, _ := time.ParseDuration(cfg.Auth.SessionTTL)
	if sessionTTL == 0 {
		sessionTTL = 24 * time.Hour
	}
	refreshTTL, _ := time.ParseDuration(cfg.Auth.RefreshTokenTTL)
	if refreshTTL == 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	linkExpiry, _ := time.ParseDuration(cfg.Auth.MagicLinkExpiry)
	if linkExpiry == 0 {
		linkExpiry = 15 * time.Minute
	}

	authService := identity.NewAuthService(
		identityPool,
		&identity.LogEmailSender{},
		cfg.Auth.BaseURL,
		cfg.Auth.SenderFrom,
		sessionTTL,
		refreshTTL,
		linkExpiry,
	)
	log.Printf("Auth service initialized (session_ttl=%s, refresh_ttl=%s, link_expiry=%s)", sessionTTL, refreshTTL, linkExpiry)

	authHandler := api.NewAuthHandler(authService)

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

	// Milestone 1: Auth endpoints (public - no auth required)
	v1Mux.HandleFunc("/api/v1/auth/request", authHandler.RequestMagicLink)
	v1Mux.HandleFunc("/api/v1/auth/verify", authHandler.VerifyMagicLink)

	// Milestone 1: Auth endpoints requiring session authentication (wrapped in middleware)
	wrapAuth := func(h http.HandlerFunc) http.Handler {
		return api.SessionAuthMiddleware(authService)(h)
	}
	v1Mux.Handle("/api/v1/auth/refresh", wrapAuth(authHandler.RefreshSession))
	v1Mux.Handle("/api/v1/auth/logout", wrapAuth(authHandler.Logout))
	v1Mux.Handle("/api/v1/auth/me", wrapAuth(authHandler.Me))

	// Milestone 1: Session management endpoints (list active sessions, revoke specific session)
	v1Mux.Handle("/api/v1/auth/sessions", wrapAuth(authHandler.ListSessions))
	// Sessions path with ID: /api/v1/auth/sessions/{id}
	v1Mux.HandleFunc("/api/v1/auth/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			authHandler.RevokeSession(w, r)
			return
		}
		api.WriteMethodNotAllowedError(w, r)
	})

	// Run startup cleanup of expired sessions/tokens/links (fire and forget)
	go func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := authService.CleanupExpiredSessionsAndTokens(cleanupCtx)
		if err != nil {
			log.Printf("Startup cleanup error: %v", err)
		} else if result.SessionsDeleted > 0 || result.RefreshTokensDeleted > 0 || result.MagicLinksDeleted > 0 {
			log.Printf("Startup cleanup removed stale data (sessions=%d, refresh_tokens=%d, magic_links=%d)",
				result.SessionsDeleted, result.RefreshTokensDeleted, result.MagicLinksDeleted)
		}
	}()

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
	v1Mux.HandleFunc("/api/v1/debug/readiness/smoke", readinessHandler.Smoke)
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

	// Mount /api/v1 sub-mux behind request ID + access log middleware
	// Trailing slash ensures prefix matching for all /api/v1/... routes
	mux.Handle("/api/v1/", api.AccessLogMiddleware(reqCtx.Middleware(v1Mux)))

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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unalkalkan/TwelveReader/pkg/types"
	"gopkg.in/yaml.v3"
)

// Load reads and parses the configuration file
// It also supports environment variable overrides with TR_ prefix
func Load(configPath string) (*types.Config, error) {
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func Validate(cfg *types.Config) error {
	// Validate server config
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	// Validate storage adapter
	if cfg.Storage.Adapter != "local" && cfg.Storage.Adapter != "s3" {
		return fmt.Errorf("invalid storage adapter: %s (must be 'local' or 's3')", cfg.Storage.Adapter)
	}

	if cfg.Storage.Adapter == "local" {
		if cfg.Storage.Local.BasePath == "" {
			return fmt.Errorf("local storage base_path is required")
		}
		// Ensure base path is absolute
		if !filepath.IsAbs(cfg.Storage.Local.BasePath) {
			return fmt.Errorf("local storage base_path must be absolute: %s", cfg.Storage.Local.BasePath)
		}
	}

	if cfg.Storage.Adapter == "s3" {
		if cfg.Storage.S3.Bucket == "" {
			return fmt.Errorf("s3 bucket is required")
		}
		if cfg.Storage.S3.Region == "" {
			return fmt.Errorf("s3 region is required")
		}
	}

	// Validate pipeline config
	if cfg.Pipeline.WorkerPoolSize <= 0 {
		cfg.Pipeline.WorkerPoolSize = 4 // default
	}
	if cfg.Pipeline.MaxRetries < 0 {
		cfg.Pipeline.MaxRetries = 3 // default
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides
// Environment variables should be prefixed with TR_ (TwelveReader)
func applyEnvOverrides(cfg *types.Config) {
	// Server overrides
	if val := os.Getenv("TR_SERVER_HOST"); val != "" {
		cfg.Server.Host = val
	}
	if val := os.Getenv("TR_SERVER_PORT"); val != "" {
		fmt.Sscanf(val, "%d", &cfg.Server.Port)
	}

	// Storage overrides
	if val := os.Getenv("TR_STORAGE_ADAPTER"); val != "" {
		cfg.Storage.Adapter = val
	}
	if val := os.Getenv("TR_STORAGE_LOCAL_BASE_PATH"); val != "" {
		cfg.Storage.Local.BasePath = val
	}
	if val := os.Getenv("TR_STORAGE_S3_BUCKET"); val != "" {
		cfg.Storage.S3.Bucket = val
	}
	if val := os.Getenv("TR_STORAGE_S3_REGION"); val != "" {
		cfg.Storage.S3.Region = val
	}
	if val := os.Getenv("TR_STORAGE_S3_ENDPOINT"); val != "" {
		cfg.Storage.S3.Endpoint = val
	}
	if val := os.Getenv("TR_STORAGE_S3_ACCESS_KEY_ID"); val != "" {
		cfg.Storage.S3.AccessKeyID = val
	}
	if val := os.Getenv("TR_STORAGE_S3_SECRET_ACCESS_KEY"); val != "" {
		cfg.Storage.S3.SecretAccessKey = val
	}

	// Apply provider API key overrides
	applyProviderEnvOverrides(cfg)
}

// applyProviderEnvOverrides applies provider-specific env vars
func applyProviderEnvOverrides(cfg *types.Config) {
	// LLM providers
	for i := range cfg.Providers.LLM {
		prefix := fmt.Sprintf("TR_LLM_%s_", strings.ToUpper(cfg.Providers.LLM[i].Name))
		if val := os.Getenv(prefix + "API_KEY"); val != "" {
			cfg.Providers.LLM[i].APIKey = val
		}
		if val := os.Getenv(prefix + "ENDPOINT"); val != "" {
			cfg.Providers.LLM[i].Endpoint = val
		}
	}

	// TTS providers
	for i := range cfg.Providers.TTS {
		prefix := fmt.Sprintf("TR_TTS_%s_", strings.ToUpper(cfg.Providers.TTS[i].Name))
		if val := os.Getenv(prefix + "API_KEY"); val != "" {
			cfg.Providers.TTS[i].APIKey = val
		}
		if val := os.Getenv(prefix + "ENDPOINT"); val != "" {
			cfg.Providers.TTS[i].Endpoint = val
		}
	}

	// OCR providers
	for i := range cfg.Providers.OCR {
		prefix := fmt.Sprintf("TR_OCR_%s_", strings.ToUpper(cfg.Providers.OCR[i].Name))
		if val := os.Getenv(prefix + "API_KEY"); val != "" {
			cfg.Providers.OCR[i].APIKey = val
		}
		if val := os.Getenv(prefix + "ENDPOINT"); val != "" {
			cfg.Providers.OCR[i].Endpoint = val
		}
	}
}

// GetDefault returns a default configuration
func GetDefault() *types.Config {
	return &types.Config{
		Server: types.ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  15,
			WriteTimeout: 15,
		},
		Storage: types.StorageConfig{
			Adapter: "local",
			Local: types.LocalStorageOpts{
				BasePath: "/var/lib/twelvereader/storage",
			},
		},
		Pipeline: types.PipelineConfig{
			WorkerPoolSize: 4,
			MaxRetries:     3,
			RetryBackoffMs: 1000,
			TempDir:        "/tmp/twelvereader",
		},
	}
}

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

	// Ensure FeatureFlags map exists
	if cfg.FeatureFlags == nil {
		cfg.FeatureFlags = make(map[string]bool)
	}

	// Save YAML-specified flags so we can merge them on top of defaults later
	yamlFlags := make(map[string]bool)
	for k, v := range cfg.FeatureFlags {
		yamlFlags[k] = v
	}

	// Resolve environment early (env var overrides YAML) so defaults pick the right profile
	if val := os.Getenv("TR_ENVIRONMENT"); val != "" {
		cfg.Environment = val
	}
	if cfg.Environment == "" {
		cfg.Environment = "local"
	}

	// Apply environment-specific default feature flags (lowest priority)
	applyDefaultFeatureFlags(&cfg)

	// Merge YAML-specified flags on top of defaults (YAML overrides defaults)
	for k, v := range yamlFlags {
		cfg.FeatureFlags[k] = v
	}

	// Apply environment variable overrides (highest priority)
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

	// Validate environment mode
	validEnvs := map[string]bool{"local": true, "dev": true, "staging": true, "production": true}
	if !validEnvs[cfg.Environment] {
		return fmt.Errorf("invalid environment: %s (must be one of: local, dev, staging, production)", cfg.Environment)
	}

	// Apply auth config defaults
	applyAuthDefaults(cfg)

	return nil
}

// applyAuthDefaults sets default values for auth configuration.
func applyAuthDefaults(cfg *types.Config) {
	if cfg.Auth.IdentityDBPath == "" {
		cfg.Auth.IdentityDBPath = "data/identity.db"
	}
	if cfg.Auth.MagicLinkExpiry == "" {
		cfg.Auth.MagicLinkExpiry = "15m"
	}
	if cfg.Auth.SessionTTL == "" {
		cfg.Auth.SessionTTL = "24h"
	}
	if cfg.Auth.RefreshTokenTTL == "" {
		cfg.Auth.RefreshTokenTTL = "168h" // 7 days
	}
	if cfg.Auth.SenderFrom == "" {
		cfg.Auth.SenderFrom = "noreply@twelvereader.local"
	}
	// Default sender_mode based on environment (if not explicitly set in YAML).
	if cfg.Auth.SenderMode == "" {
		switch cfg.Environment {
		case "local", "dev":
			cfg.Auth.SenderMode = "log"
		case "staging", "production":
			cfg.Auth.SenderMode = "none"
		default:
			cfg.Auth.SenderMode = "log"
		}
	}
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

	// Apply feature flag overrides: TR_FEATURE_<FLAG_NAME>
	applyFeatureFlagEnvOverrides(cfg)

	// Apply auth overrides: TR_AUTH_*
	applyAuthEnvOverrides(cfg)
}

// applyProviderEnvOverrides applies provider-specific env vars
func applyProviderEnvOverrides(cfg *types.Config) {
	// LLM providers
	for i := range cfg.Providers.LLM {
		if val := providerEnvValue("LLM", cfg.Providers.LLM[i].Name, "API_KEY"); val != "" {
			cfg.Providers.LLM[i].APIKey = val
		}
		if val := providerEnvValue("LLM", cfg.Providers.LLM[i].Name, "ENDPOINT"); val != "" {
			cfg.Providers.LLM[i].Endpoint = val
		}
	}

	// TTS providers
	for i := range cfg.Providers.TTS {
		if val := providerEnvValue("TTS", cfg.Providers.TTS[i].Name, "API_KEY"); val != "" {
			cfg.Providers.TTS[i].APIKey = val
		}
		if val := providerEnvValue("TTS", cfg.Providers.TTS[i].Name, "ENDPOINT"); val != "" {
			cfg.Providers.TTS[i].Endpoint = val
		}
	}

	// OCR providers
	for i := range cfg.Providers.OCR {
		if val := providerEnvValue("OCR", cfg.Providers.OCR[i].Name, "API_KEY"); val != "" {
			cfg.Providers.OCR[i].APIKey = val
		}
		if val := providerEnvValue("OCR", cfg.Providers.OCR[i].Name, "ENDPOINT"); val != "" {
			cfg.Providers.OCR[i].Endpoint = val
		}
		if val := providerEnvValue("OCR", cfg.Providers.OCR[i].Name, "MODEL"); val != "" {
			if cfg.Providers.OCR[i].Options == nil {
				cfg.Providers.OCR[i].Options = make(map[string]string)
			}
			cfg.Providers.OCR[i].Options["model"] = val
		}
	}
}

func providerEnvValue(kind, name, suffix string) string {
	for _, prefix := range providerEnvPrefixes(kind, name) {
		if val := os.Getenv(prefix + suffix); val != "" {
			return val
		}
	}
	return ""
}

func providerEnvPrefixes(kind, name string) []string {
	safe := fmt.Sprintf("TR_%s_%s_", kind, envSafeProviderName(name))
	legacy := fmt.Sprintf("TR_%s_%s_", kind, strings.ToUpper(name))
	if legacy == safe {
		return []string{safe}
	}
	return []string{safe, legacy}
}

func envSafeProviderName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

// applyDefaultFeatureFlags sets environment-specific default flags.
// These are the lowest-priority values — YAML config and TR_FEATURE_* env vars override them.
func applyDefaultFeatureFlags(cfg *types.Config) {
	if cfg.FeatureFlags == nil {
		cfg.FeatureFlags = make(map[string]bool)
	}

	// Known SaaS feature flags (milestone-driven)
	defaults := map[string]map[string]bool{
		"local": {
			"saas_auth":      false,
			"usage_metering": false,
			"quota_engine":   false,
			"repository_pub": false,
			"user_accounts":  false,
			"billing":        false,
		},
		"dev": {
			"saas_auth":      false,
			"usage_metering": true,
			"quota_engine":   false,
			"repository_pub": false,
			"user_accounts":  false,
			"billing":        false,
		},
		"staging": {
			"saas_auth":      true,
			"usage_metering": true,
			"quota_engine":   true,
			"repository_pub": false,
			"user_accounts":  true,
			"billing":        false,
		},
		"production": {
			"saas_auth":      true,
			"usage_metering": true,
			"quota_engine":   true,
			"repository_pub": false,
			"user_accounts":  true,
			"billing":        false,
		},
	}

	envDefaults, ok := defaults[cfg.Environment]
	if !ok {
		// Fallback to "local" defaults for unknown environments
		envDefaults = defaults["local"]
	}

	for name, enabled := range envDefaults {
		cfg.FeatureFlags[name] = enabled
	}

	// Merge YAML-specified flags on top of defaults (YAML takes precedence)
	// This is handled in Load() — we only set defaults here.
}

// applyFeatureFlagEnvOverrides applies TR_FEATURE_<FLAG_NAME> env vars to feature flags.
func applyFeatureFlagEnvOverrides(cfg *types.Config) {
	prefix := "TR_FEATURE_"
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, prefix) {
			continue
		}
		// Extract KEY=VALUE
		idx := strings.IndexByte(e, '=')
		if idx < 0 {
			continue
		}
		key := strings.ToLower(e[len(prefix):idx])
		val := e[idx+1:]
		// Convert to lowercase bool parsing
		cfg.FeatureFlags[key] = (val == "true" || val == "True" || val == "TRUE" || val == "1")
	}
}

// applyAuthEnvOverrides applies TR_AUTH_* env vars to auth configuration.
func applyAuthEnvOverrides(cfg *types.Config) {
	if val := os.Getenv("TR_AUTH_IDENTITY_DB_PATH"); val != "" {
		cfg.Auth.IdentityDBPath = val
	}
	if val := os.Getenv("TR_AUTH_MAGIC_LINK_EXPIRY"); val != "" {
		cfg.Auth.MagicLinkExpiry = val
	}
	if val := os.Getenv("TR_AUTH_SESSION_TTL"); val != "" {
		cfg.Auth.SessionTTL = val
	}
	if val := os.Getenv("TR_AUTH_REFRESH_TOKEN_TTL"); val != "" {
		cfg.Auth.RefreshTokenTTL = val
	}
	if val := os.Getenv("TR_AUTH_SENDER_FROM"); val != "" {
		cfg.Auth.SenderFrom = val
	}
	if val := os.Getenv("TR_AUTH_BASE_URL"); val != "" {
		cfg.Auth.BaseURL = val
	}
	if val := os.Getenv("TR_AUTH_BOOTSTRAP_ADMIN_EMAIL"); val != "" {
		cfg.Auth.BootstrapAdminEmail = val
	}

	// Sender mode override (highest priority)
	if val := os.Getenv("TR_AUTH_SENDER_MODE"); val != "" {
		cfg.Auth.SenderMode = val
	}

	// SMTP configuration overrides
	if val := os.Getenv("TR_AUTH_SMTP_HOST"); val != "" {
		cfg.Auth.SMTP.Host = val
	}
	if val := os.Getenv("TR_AUTH_SMTP_PORT"); val != "" {
		fmt.Sscanf(val, "%d", &cfg.Auth.SMTP.Port)
	}
	if val := os.Getenv("TR_AUTH_SMTP_USERNAME"); val != "" {
		cfg.Auth.SMTP.Username = val
	}
	if val := os.Getenv("TR_AUTH_SMTP_PASSWORD"); val != "" {
		cfg.Auth.SMTP.Password = val
	}
	if val := os.Getenv("TR_AUTH_SMTP_USE_TLS"); val != "" {
		cfg.Auth.SMTP.UseTLS = (val == "true" || val == "True" || val == "TRUE" || val == "1")
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
		Environment:  "local",
		FeatureFlags: nil,
	}
}

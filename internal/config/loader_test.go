package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 9090
  read_timeout: 10
  write_timeout: 10

storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"

pipeline:
  worker_pool_size: 2
  max_retries: 3
  retry_backoff_ms: 500
  temp_dir: "/tmp"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load configuration
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Storage.Adapter != "local" {
		t.Errorf("Expected adapter 'local', got '%s'", cfg.Storage.Adapter)
	}
	if cfg.Storage.Local.BasePath != "/tmp/test" {
		t.Errorf("Expected base_path '/tmp/test', got '%s'", cfg.Storage.Local.BasePath)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*types.Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *types.Config) {},
			wantErr: false,
		},
		{
			name: "invalid port",
			modify: func(c *types.Config) {
				c.Server.Port = 0
			},
			wantErr: true,
		},
		{
			name: "invalid storage adapter",
			modify: func(c *types.Config) {
				c.Storage.Adapter = "invalid"
			},
			wantErr: true,
		},
		{
			name: "missing local base path",
			modify: func(c *types.Config) {
				c.Storage.Adapter = "local"
				c.Storage.Local.BasePath = ""
			},
			wantErr: true,
		},
		{
			name: "missing s3 bucket",
			modify: func(c *types.Config) {
				c.Storage.Adapter = "s3"
				c.Storage.S3.Bucket = ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := GetDefault()
			tt.modify(cfg)
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvOverrides(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variables
	os.Setenv("TR_SERVER_PORT", "9999")
	os.Setenv("TR_STORAGE_LOCAL_BASE_PATH", "/tmp/override")
	defer func() {
		os.Unsetenv("TR_SERVER_PORT")
		os.Unsetenv("TR_STORAGE_LOCAL_BASE_PATH")
	}()

	// Load configuration
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment overrides were applied
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999 from env override, got %d", cfg.Server.Port)
	}
	if cfg.Storage.Local.BasePath != "/tmp/override" {
		t.Errorf("Expected base_path '/tmp/override' from env override, got '%s'", cfg.Storage.Local.BasePath)
	}
}

func TestEnvOverrides_OCRModel(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
providers:
  ocr:
    - name: "vision"
      enabled: true
      endpoint: "https://api.openai.com/v1"
      options:
        language: "eng"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	os.Setenv("TR_OCR_VISION_MODEL", "gpt-4o")
	defer os.Unsetenv("TR_OCR_VISION_MODEL")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Providers.OCR) != 1 {
		t.Fatalf("Expected 1 OCR provider, got %d", len(cfg.Providers.OCR))
	}

	ocr := cfg.Providers.OCR[0]
	if ocr.Options["model"] != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o' from env override, got '%s'", ocr.Options["model"])
	}
}

func TestEnvOverrides_OCRModelCreatesOptions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
providers:
  ocr:
    - name: "mynv"
      enabled: true
      endpoint: "https://api.openai.com/v1"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	os.Setenv("TR_OCR_MYNV_MODEL", "gpt-4o-mini")
	defer os.Unsetenv("TR_OCR_MYNV_MODEL")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	ocr := cfg.Providers.OCR[0]
	if ocr.Options == nil {
		t.Fatal("Expected Options map to be created by env override")
	}
	if ocr.Options["model"] != "gpt-4o-mini" {
		t.Errorf("Expected model 'gpt-4o-mini', got '%s'", ocr.Options["model"])
	}
}

func TestEnvOverrides_ProviderNamesKeepLegacyHyphenFallback(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
providers:
  llm:
    - name: "local-llm"
      enabled: true
  tts:
    - name: "qwen3-tts"
      enabled: true
  ocr:
    - name: "openai-ocr"
      enabled: true
      endpoint: "https://api.openai.com/v1"
      options:
        language: "eng"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	os.Setenv("TR_LLM_LOCAL-LLM_ENDPOINT", "http://legacy-llm.local/v1")
	os.Setenv("TR_TTS_QWEN3-TTS_ENDPOINT", "http://legacy-tts.local/v1")
	os.Setenv("TR_OCR_OPENAI-OCR_MODEL", "legacy-vision")
	defer func() {
		os.Unsetenv("TR_LLM_LOCAL-LLM_ENDPOINT")
		os.Unsetenv("TR_TTS_QWEN3-TTS_ENDPOINT")
		os.Unsetenv("TR_OCR_OPENAI-OCR_MODEL")
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Providers.LLM[0].Endpoint != "http://legacy-llm.local/v1" {
		t.Errorf("Expected legacy hyphenated LLM env override to apply, got '%s'", cfg.Providers.LLM[0].Endpoint)
	}
	if cfg.Providers.TTS[0].Endpoint != "http://legacy-tts.local/v1" {
		t.Errorf("Expected legacy hyphenated TTS env override to apply, got '%s'", cfg.Providers.TTS[0].Endpoint)
	}
	if cfg.Providers.OCR[0].Options["model"] != "legacy-vision" {
		t.Errorf("Expected legacy hyphenated OCR model env override to apply, got '%s'", cfg.Providers.OCR[0].Options["model"])
	}
}

func TestEnvOverrides_ProviderNamesUseSafeUnderscores(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
providers:
  llm:
    - name: "local-llm"
      enabled: true
  tts:
    - name: "qwen3-tts"
      enabled: true
  ocr:
    - name: "openai-ocr"
      enabled: true
      endpoint: "https://api.openai.com/v1"
      options:
        language: "eng"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	os.Setenv("TR_LLM_LOCAL_LLM_ENDPOINT", "http://localhost:11434/v1")
	os.Setenv("TR_TTS_QWEN3_TTS_ENDPOINT", "http://tts.local/v1")
	os.Setenv("TR_OCR_OPENAI_OCR_MODEL", "gpt-4o")
	defer func() {
		os.Unsetenv("TR_LLM_LOCAL_LLM_ENDPOINT")
		os.Unsetenv("TR_TTS_QWEN3_TTS_ENDPOINT")
		os.Unsetenv("TR_OCR_OPENAI_OCR_MODEL")
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Providers.LLM[0].Endpoint != "http://localhost:11434/v1" {
		t.Errorf("Expected hyphenated LLM env override to apply, got '%s'", cfg.Providers.LLM[0].Endpoint)
	}
	if cfg.Providers.TTS[0].Endpoint != "http://tts.local/v1" {
		t.Errorf("Expected hyphenated TTS env override to apply, got '%s'", cfg.Providers.TTS[0].Endpoint)
	}
	if cfg.Providers.OCR[0].Options["model"] != "gpt-4o" {
		t.Errorf("Expected hyphenated OCR model env override to apply, got '%s'", cfg.Providers.OCR[0].Options["model"])
	}
}

func TestGetDefault(t *testing.T) {
	cfg := GetDefault()
	if cfg == nil {
		t.Fatal("GetDefault() returned nil")
	}
	if cfg.Server.Port <= 0 {
		t.Error("Default config has invalid port")
	}
	if cfg.Storage.Adapter == "" {
		t.Error("Default config has empty storage adapter")
	}
}

func TestEnvironmentSpecificDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	baseConfig := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
`

	tests := []struct {
		name        string
		environment string
		expected    map[string]bool
	}{
		{
			name:        "local defaults",
			environment: "local",
			expected: map[string]bool{
				"saas_auth":      false,
				"usage_metering": false,
				"quota_engine":   false,
				"repository_pub": false,
				"user_accounts":  false,
				"billing":        false,
			},
		},
		{
			name:        "dev defaults - usage_metering on",
			environment: "dev",
			expected: map[string]bool{
				"saas_auth":      false,
				"usage_metering": true,
				"quota_engine":   false,
				"repository_pub": false,
				"user_accounts":  false,
				"billing":        false,
			},
		},
		{
			name:        "staging defaults - auth + metering + quota on",
			environment: "staging",
			expected: map[string]bool{
				"saas_auth":      true,
				"usage_metering": true,
				"quota_engine":   true,
				"repository_pub": false,
				"user_accounts":  true,
				"billing":        false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, "test_"+tt.environment+".yaml")
			content := baseConfig + "\nenvironment: \"" + tt.environment + "\"\n"
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}

			for flag, expectedVal := range tt.expected {
				actual, ok := cfg.FeatureFlags[flag]
				if !ok {
					t.Errorf("Flag %q missing from FeatureFlags", flag)
					continue
				}
				if actual != expectedVal {
					t.Errorf("Flag %q: got %v, want %v (environment=%s)", flag, actual, expectedVal, tt.environment)
				}
			}
		})
	}
}

func TestYamlOverridesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Local defaults: all off. Override billing=true in YAML.
	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
environment: "local"
feature_flags:
  billing: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Default for local environment should apply for saas_auth
	if !ok(cfg.FeatureFlags, "saas_auth") || cfg.FeatureFlags["saas_auth"] != false {
		t.Error("Expected saas_auth=false (local default)")
	}
	// YAML override should win
	if cfg.FeatureFlags["billing"] != true {
		t.Error("Expected billing=true (YAML override of local default)")
	}
}

func ok(m map[string]bool, key string) bool {
	_, exists := m[key]
	return exists
}

func TestFeatureFlagEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
environment: "local"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Local default for billing is false; override via env var
	os.Setenv("TR_FEATURE_BILLING", "true")
	defer os.Unsetenv("TR_FEATURE_BILLING")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.FeatureFlags["billing"] != true {
		t.Errorf("Expected billing=true from TR_FEATURE_BILLING env var, got %v", cfg.FeatureFlags["billing"])
	}
	// Other local defaults should be intact
	if cfg.FeatureFlags["saas_auth"] != false {
		t.Errorf("Expected saas_auth=false (local default), got %v", cfg.FeatureFlags["saas_auth"])
	}
}

func TestTrEnvironmentVarAffectsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Config specifies local, but env var says production → should get prod defaults
	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
environment: "local"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	os.Setenv("TR_ENVIRONMENT", "production")
	defer os.Unsetenv("TR_ENVIRONMENT")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// TR_ENVIRONMENT=production should make env defaults production-based
	if cfg.Environment != "production" {
		t.Errorf("Expected environment='production', got '%s'", cfg.Environment)
	}
	// Production default for saas_auth is true
	if cfg.FeatureFlags["saas_auth"] != true {
		t.Errorf("Expected saas_auth=true (production default), got %v", cfg.FeatureFlags["saas_auth"])
	}
	// Usage metering should also be true in production defaults
	if cfg.FeatureFlags["usage_metering"] != true {
		t.Errorf("Expected usage_metering=true (production default), got %v", cfg.FeatureFlags["usage_metering"])
	}
}

func TestFeatureFlagEnvVarBoolParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  host: "localhost"
  port: 8080
storage:
  adapter: "local"
  local:
    base_path: "/tmp/test"
pipeline:
  worker_pool_size: 2
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	tests := []struct {
		val    string
		expect bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"False", false},
		{"0", false},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			os.Setenv("TR_FEATURE_TEST_FLAG", tt.val)
			defer os.Unsetenv("TR_FEATURE_TEST_FLAG")

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}

			if cfg.FeatureFlags["test_flag"] != tt.expect {
				t.Errorf("TR_FEATURE_TEST_FLAG=%s → got %v, want %v", tt.val, cfg.FeatureFlags["test_flag"], tt.expect)
			}
		})
	}
}

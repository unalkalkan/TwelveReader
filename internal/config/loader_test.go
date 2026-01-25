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

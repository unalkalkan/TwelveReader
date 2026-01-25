package types

// Config represents the overall application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server" json:"server"`
	Storage   StorageConfig   `yaml:"storage" json:"storage"`
	Providers ProvidersConfig `yaml:"providers" json:"providers"`
	Pipeline  PipelineConfig  `yaml:"pipeline" json:"pipeline"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host         string `yaml:"host" json:"host"`
	Port         int    `yaml:"port" json:"port"`
	ReadTimeout  int    `yaml:"read_timeout" json:"read_timeout"`   // seconds
	WriteTimeout int    `yaml:"write_timeout" json:"write_timeout"` // seconds
}

// StorageConfig defines storage adapter settings
type StorageConfig struct {
	Adapter string            `yaml:"adapter" json:"adapter"` // "local" or "s3"
	Local   LocalStorageOpts  `yaml:"local" json:"local"`
	S3      S3StorageOpts     `yaml:"s3" json:"s3"`
	Options map[string]string `yaml:"options" json:"options"` // Additional adapter-specific options
}

// LocalStorageOpts configures the local filesystem adapter
type LocalStorageOpts struct {
	BasePath string `yaml:"base_path" json:"base_path"`
}

// S3StorageOpts configures the S3-compatible adapter
type S3StorageOpts struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint"`
	Region          string `yaml:"region" json:"region"`
	Bucket          string `yaml:"bucket" json:"bucket"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"`
	UseSSL          bool   `yaml:"use_ssl" json:"use_ssl"`
}

// ProvidersConfig holds all provider configurations
type ProvidersConfig struct {
	LLM []LLMProviderConfig `yaml:"llm" json:"llm"`
	TTS []TTSProviderConfig `yaml:"tts" json:"tts"`
	OCR []OCRProviderConfig `yaml:"ocr" json:"ocr"`
}

// LLMProviderConfig configures an LLM provider
type LLMProviderConfig struct {
	Name          string            `yaml:"name" json:"name"`
	Enabled       bool              `yaml:"enabled" json:"enabled"`
	Endpoint      string            `yaml:"endpoint" json:"endpoint"`
	APIKey        string            `yaml:"api_key" json:"api_key"`
	Model         string            `yaml:"model" json:"model"`
	ContextWindow int               `yaml:"context_window" json:"context_window"`
	Concurrency   int               `yaml:"concurrency" json:"concurrency"`
	RateLimitQPS  float64           `yaml:"rate_limit_qps" json:"rate_limit_qps"`
	Options       map[string]string `yaml:"options" json:"options"`
}

// TTSProviderConfig configures a TTS provider
type TTSProviderConfig struct {
	Name           string            `yaml:"name" json:"name"`
	Enabled        bool              `yaml:"enabled" json:"enabled"`
	Endpoint       string            `yaml:"endpoint" json:"endpoint"`
	APIKey         string            `yaml:"api_key" json:"api_key"`
	MaxSegmentSize int               `yaml:"max_segment_size" json:"max_segment_size"` // characters
	Concurrency    int               `yaml:"concurrency" json:"concurrency"`
	RateLimitQPS   float64           `yaml:"rate_limit_qps" json:"rate_limit_qps"`
	TimestampPrec  string            `yaml:"timestamp_precision" json:"timestamp_precision"` // "word" or "sentence"
	Options        map[string]string `yaml:"options" json:"options"`
}

// OCRProviderConfig configures an OCR provider
type OCRProviderConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Endpoint    string            `yaml:"endpoint" json:"endpoint"`
	APIKey      string            `yaml:"api_key" json:"api_key"`
	Concurrency int               `yaml:"concurrency" json:"concurrency"`
	Options     map[string]string `yaml:"options" json:"options"`
}

// PipelineConfig holds pipeline-level settings
type PipelineConfig struct {
	WorkerPoolSize int    `yaml:"worker_pool_size" json:"worker_pool_size"`
	MaxRetries     int    `yaml:"max_retries" json:"max_retries"`
	RetryBackoffMs int    `yaml:"retry_backoff_ms" json:"retry_backoff_ms"`
	TempDir        string `yaml:"temp_dir" json:"temp_dir"`
}

# TwelveReader Server Implementation

## Overview

This directory contains the Golang server implementation for TwelveReader. The server orchestrates book ingestion, LLM-driven segmentation, voice assignment, and TTS synthesis.

## Project Structure

```
.
├── cmd/
│   └── server/          # Main server entry point
├── internal/
│   ├── config/          # Configuration loading and validation
│   ├── health/          # Health check handlers
│   ├── provider/        # Provider interfaces and registry
│   └── storage/         # Storage adapter implementations
├── pkg/
│   └── types/           # Shared type definitions
├── config/              # Configuration files
│   └── dev.example.yaml # Example development configuration
├── Makefile             # Build and development tasks
└── go.mod               # Go module definition
```

## Quick Start

### Prerequisites
- Go 1.24 or later
- Make (optional, but recommended)

### Build
```bash
make build
```

### Run Tests
```bash
make test
```

### Run Server
```bash
# Using make
make run

# Or directly with custom config
./bin/twelvereader -config config/dev.example.yaml
```

## Configuration

The server uses YAML configuration files. Copy `config/dev.example.yaml` and customize:

```bash
cp config/dev.example.yaml config/dev.yaml
# Edit config/dev.yaml with your settings
```

### Key Configuration Sections

1. **Server**: HTTP server settings (host, port, timeouts)
2. **Storage**: Storage adapter configuration (local filesystem or S3)
3. **Providers**: LLM, TTS, and OCR provider configurations
4. **Pipeline**: Pipeline settings (worker pools, retries)

See [API.md](API.md) for detailed configuration options and environment variables.

## Architecture

### Storage Adapters

The storage layer is abstracted through the `Adapter` interface, allowing different backends:

- **Local**: Filesystem-based storage for development and small deployments
- **S3**: S3-compatible storage for production (supports AWS S3, MinIO, etc.)

### Provider Registry

Providers are registered dynamically from configuration:

- **LLM Providers**: OpenAI-compatible endpoints for segmentation
  - Supports OpenAI, Azure OpenAI, local LLMs (Ollama, LM Studio, etc.)
  - Requires `endpoint` and `model` configuration
  - Optional `api_key` for authenticated endpoints
  - Falls back to stub provider if endpoint/model not configured
- **TTS Providers**: Text-to-speech synthesis (e.g., Qwen3-TTS, OpenAI TTS)
- **OCR Providers**: Optical character recognition for scanned PDFs

All providers implement standard interfaces and can be swapped without code changes.

### Health Checks

Three health endpoints are available:

- `/health/live` - Basic liveness check
- `/health/ready` - Readiness check with component validation
- `/health` - Comprehensive health report

## Development

### Available Make Targets

```bash
make help          # Show all available targets
make build         # Build the server binary
make test          # Run all tests with race detector
make test-coverage # Generate HTML coverage report
make lint          # Run linter (go vet + formatting check)
make fmt           # Format code
make clean         # Remove build artifacts
make run           # Build and run server
make dev           # Run in development mode
make deps          # Download dependencies
```

### Running Tests

```bash
# Run all tests
make test

# Run tests for a specific package
go test ./internal/config -v

# Run tests with coverage
make test-coverage
# Open coverage.html in browser
```

### Code Quality

Before committing:

```bash
make fmt   # Format code
make lint  # Check for issues
make test  # Ensure tests pass
```

## Testing

### Unit Tests

- `internal/config/loader_test.go` - Configuration loading and validation
- `internal/storage/local_test.go` - Local storage adapter
- `internal/provider/registry_test.go` - Provider registry and stubs

### Integration Tests

Storage adapters include integration tests that verify:
- File creation and retrieval
- Directory listing
- Concurrent access
- Error handling

## Security

### API Keys and Secrets

- Never commit API keys or secrets to the repository
- Use environment variables for sensitive values
- See `config/dev.example.yaml` for placeholder patterns

### Storage

- Local storage paths must be absolute
- S3 credentials should be provided via environment variables
- All storage operations include context for timeout/cancellation

## Monitoring

### Health Checks

Kubernetes-style health checks are provided:

- **Liveness**: Is the process running?
- **Readiness**: Can the service handle requests?

### Metrics (Future)

Planned for future milestones:
- Request counts and latencies
- Provider API usage
- Storage operations
- Pipeline throughput

## Troubleshooting

### Server won't start

1. Check configuration file exists: `config/dev.example.yaml`
2. Verify storage path is accessible
3. Check port availability: `lsof -i :8080`

### Tests failing

1. Ensure dependencies are up to date: `make deps`
2. Check Go version: `go version` (requires 1.24+)
3. Run with verbose output: `go test -v ./...`

### Storage errors

1. Verify storage adapter configuration
2. For local adapter: ensure base path exists and is writable
3. For S3: check credentials and bucket access

## Contributing

When adding new features:

1. Follow existing code structure
2. Add tests for new functionality
3. Update documentation
4. Run `make all` before committing

## License

See LICENSE file in repository root.

## Next Steps (Milestone 5)

The next milestone will implement:

- Web Client MVP (React + TypeScript)
- Tamagui UI components
- TanStack Query for API integration
- Zod for schema validation
- Book upload and management interface
- Playback UI with text highlighting
- Voice mapping interface

See `Milestones.md` for the complete roadmap.

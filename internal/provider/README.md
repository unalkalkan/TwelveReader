# Provider Package

This package contains the provider interfaces and implementations for LLM, TTS, and OCR services.

## LLM Providers

### OpenAI-Compatible Provider

The `OpenAILLMProvider` implements the `LLMProvider` interface and supports any OpenAI-compatible API endpoint. This includes:

- OpenAI API (https://api.openai.com/v1)
- Azure OpenAI
- Local LLM servers (Ollama, LM Studio, LocalAI, etc.)
- Any other service implementing the OpenAI chat completion API

#### Configuration

To use the OpenAI-compatible provider, configure your LLM provider with:

```yaml
providers:
  llm:
    - name: "my-llm"
      enabled: true
      endpoint: "https://api.openai.com/v1"  # Required: API endpoint
      api_key: "sk-..."                       # Optional: API key (can use env var)
      model: "gpt-4"                          # Required: Model name
      context_window: 8192                    # Optional: Context window size
      concurrency: 5                          # Optional: Concurrent requests
      rate_limit_qps: 10.0                    # Optional: Rate limit
      options:
        temperature: "0.7"                    # Optional: Temperature (0.0-2.0)
        timeout: "60"                         # Optional: HTTP timeout in seconds (default: 60)
```

#### Examples

**Using OpenAI:**
```yaml
- name: "openai"
  enabled: true
  endpoint: "https://api.openai.com/v1"
  api_key: ""  # Set via TR_LLM_OPENAI_API_KEY env var
  model: "gpt-4"
```

**Using a local Ollama instance:**
```yaml
- name: "local-llm"
  enabled: true
  endpoint: "http://localhost:11434/v1"
  api_key: ""  # Not needed for Ollama
  model: "llama2"
```

**Using Azure OpenAI:**
```yaml
- name: "azure-openai"
  enabled: true
  endpoint: "https://your-resource.openai.azure.com/openai/deployments/your-deployment"
  api_key: ""  # Set via env var
  model: "gpt-4"
```

#### Fallback Behavior

If a provider configuration doesn't include both `endpoint` and `model`, the system will automatically use the `StubLLMProvider` for backward compatibility and testing. The stub provider returns the input text as a single segment with default values.

#### API Details

The OpenAI provider:
1. Constructs a detailed prompt for text segmentation
2. Injects the known-people list (when provided) to enforce consistent speaker identifiers
3. Calls the `/chat/completions` endpoint with the prompt
4. Parses the JSON response containing segment information
5. Returns structured segments with speaker, language, and voice description

Each segment includes:
- `text`: The text content of the segment
- `person`: Speaker identifier (e.g., "narrator", "character_name")
- `language`: ISO-639-1 language code (e.g., "en", "es")
- `voice_description`: Voice/tone description (e.g., "neutral", "excited")

#### Error Handling

The provider includes robust error handling:
- HTTP errors are reported with status codes and messages
- Malformed JSON responses fallback to treating the full response as a single segment
- Network timeouts use a 60-second default
- Context cancellation is supported for request cancellation

## Testing

The provider includes comprehensive tests with mock HTTP servers:
- `openai_llm_test.go`: Unit tests for the OpenAI provider
- `registry_test.go`: Integration tests for provider registration

Run tests with:
```bash
go test ./internal/provider/... -v
```

## Extending

To add a new LLM provider:

1. Implement the `LLMProvider` interface from `interfaces.go`
2. Add a factory function (e.g., `NewMyLLMProvider`)
3. Update `registry.go`'s `InitializeProviders` to instantiate your provider
4. Add tests

The interface requires three methods:
- `Name() string`: Return the provider name
- `Segment(ctx context.Context, req SegmentRequest) (*SegmentResponse, error)`: Perform segmentation
- `Close() error`: Clean up resources

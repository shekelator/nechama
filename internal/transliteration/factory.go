package transliteration

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const ProviderOllama = "ollama"

type FactoryConfig struct {
	Provider string
	Ollama   OllamaFactoryConfig
}

type OllamaFactoryConfig struct {
	BaseURL string
	Model   string
	APIKey  string
	Timeout time.Duration
}

// NewHybridFromConfig builds a HybridService. The LLM provider is constructed
// only when both base_url and model are configured; otherwise the hybrid runs
// as a pure deterministic engine with no network calls.
func NewHybridFromConfig(cfg FactoryConfig, rules string, logger *slog.Logger) (*HybridService, error) {
	provider, err := optionalProvider(cfg)
	if err != nil {
		return nil, err
	}
	return NewHybridService(provider, rules, logger)
}

// optionalProvider returns a configured provider when base_url and model are
// both set, and nil otherwise.
func optionalProvider(cfg FactoryConfig) (Provider, error) {
	if strings.TrimSpace(cfg.Ollama.BaseURL) == "" || strings.TrimSpace(cfg.Ollama.Model) == "" {
		return nil, nil
	}
	return NewProviderFromConfig(cfg)
}

func NewProviderFromConfig(config FactoryConfig) (Provider, error) {
	provider := strings.ToLower(strings.TrimSpace(config.Provider))
	if provider == "" {
		provider = ProviderOllama
	}

	switch provider {
	case ProviderOllama:
		timeout := config.Ollama.Timeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}

		httpClient := &http.Client{Timeout: timeout}
		return NewOllamaClient(
			WithOllamaBaseURL(config.Ollama.BaseURL),
			WithOllamaModel(config.Ollama.Model),
			WithOllamaAPIKey(config.Ollama.APIKey),
			WithOllamaHTTPClient(httpClient),
		)
	default:
		return nil, fmt.Errorf("transliteration provider %q is not implemented yet", provider)
	}
}

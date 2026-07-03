package transliteration

import (
	"fmt"
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

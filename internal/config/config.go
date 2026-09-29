package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultConfigFileName = "config.json"

// Transliteration modes.
const (
	// ModeHybrid uses a deterministic engine for the bulk of the work and
	// consults the configured LLM provider only for words the engine flags as
	// ambiguous. The provider is optional in this mode.
	ModeHybrid = "hybrid"
	// ModeModel sends the entire text to the LLM provider, which must be
	// configured.
	ModeModel = "model"
)

const (
	envTransliterationProvider = "NECHAMA_TRANSLITERATION_PROVIDER"
	envTransliterationBaseURL   = "NECHAMA_TRANSLITERATION_BASE_URL"
	envTransliterationModel     = "NECHAMA_TRANSLITERATION_MODEL"
	envTransliterationAPIKey    = "NECHAMA_TRANSLITERATION_API_KEY"
	envTransliterationTimeout   = "NECHAMA_TRANSLITERATION_TIMEOUT_SECONDS"
	envTransliterationMode      = "NECHAMA_TRANSLITERATION_MODE"
)

type AppConfig struct {
	Transliteration TransliterationConfig `json:"transliteration"`
}

type TransliterationConfig struct {
	Provider string       `json:"provider"`
	Mode     string       `json:"mode"`
	Ollama   OllamaConfig `json:"ollama"`
}

type OllamaConfig struct {
	BaseURL        string `json:"base_url"`
	Model          string `json:"model"`
	APIKey         string `json:"api_key"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func Default() AppConfig {
	return AppConfig{
		Transliteration: TransliterationConfig{
			Provider: "ollama",
			Mode:     ModeHybrid,
			Ollama: OllamaConfig{
				BaseURL:        "http://host.docker.internal:11434",
				Model:          "gemma4:cloud",
				APIKey:         "dummy-api-key",
				TimeoutSeconds: 60,
			},
		},
	}
}

func Load() (AppConfig, error) {
	config := Default()

	path, err := configPath()
	if err != nil {
		return AppConfig{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			applyEnvOverrides(&config)
			if err := config.Validate(); err != nil {
				return AppConfig{}, err
			}
			return config, nil
		}
		return AppConfig{}, err
	}

	if err := json.Unmarshal(content, &config); err != nil {
		return AppConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	applyEnvOverrides(&config)

	if err := config.Validate(); err != nil {
		return AppConfig{}, err
	}

	return config, nil
}

func (c AppConfig) Validate() error {
	provider := strings.ToLower(strings.TrimSpace(c.Transliteration.Provider))
	if provider == "" {
		provider = "ollama"
	}

	switch provider {
	case "ollama":
	default:
		return fmt.Errorf("unsupported transliteration provider %q", c.Transliteration.Provider)
	}

	mode := c.TransliterationMode()
	switch mode {
	case ModeHybrid, ModeModel:
	default:
		return fmt.Errorf("unsupported transliteration mode %q (want %q or %q)", c.Transliteration.Mode, ModeHybrid, ModeModel)
	}

	// The LLM provider is only required in model mode. In hybrid mode the
	// deterministic engine does the work and the provider is optional (used
	// only for the few words the engine flags as ambiguous).
	if mode == ModeModel {
		if strings.TrimSpace(c.Transliteration.Ollama.BaseURL) == "" {
			return errors.New("transliteration.ollama.base_url is required for model mode")
		}
		if strings.TrimSpace(c.Transliteration.Ollama.Model) == "" {
			return errors.New("transliteration.ollama.model is required for model mode")
		}
		if c.Transliteration.Ollama.TimeoutSeconds <= 0 {
			return errors.New("transliteration.ollama.timeout_seconds must be greater than zero")
		}
	}

	return nil
}

func (c AppConfig) TransliterationProvider() string {
	provider := strings.ToLower(strings.TrimSpace(c.Transliteration.Provider))
	if provider == "" {
		return "ollama"
	}
	return provider
}

// TransliterationMode returns the configured transliteration mode, defaulting
// to hybrid when unset.
func (c AppConfig) TransliterationMode() string {
	mode := strings.ToLower(strings.TrimSpace(c.Transliteration.Mode))
	if mode == "" {
		return ModeHybrid
	}
	return mode
}

func (c AppConfig) OllamaTimeout() time.Duration {
	seconds := c.Transliteration.Ollama.TimeoutSeconds
	if seconds <= 0 {
		seconds = 30
	}
	return time.Duration(seconds) * time.Second
}

func configPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("NECHAMA_CONFIG")); path != "" {
		return path, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, "nechama", defaultConfigFileName), nil
}

func applyEnvOverrides(config *AppConfig) {
	if config == nil {
		return
	}

	if value := strings.TrimSpace(os.Getenv(envTransliterationProvider)); value != "" {
		config.Transliteration.Provider = value
	}
	if value := strings.TrimSpace(os.Getenv(envTransliterationMode)); value != "" {
		config.Transliteration.Mode = value
	}
	if value := strings.TrimSpace(os.Getenv(envTransliterationBaseURL)); value != "" {
		config.Transliteration.Ollama.BaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv(envTransliterationModel)); value != "" {
		config.Transliteration.Ollama.Model = value
	}
	if value := os.Getenv(envTransliterationAPIKey); value != "" {
		config.Transliteration.Ollama.APIKey = value
	}
	if value := strings.TrimSpace(os.Getenv(envTransliterationTimeout)); value != "" {
		seconds, err := strconv.Atoi(value)
		if err == nil {
			config.Transliteration.Ollama.TimeoutSeconds = seconds
		}
	}
}

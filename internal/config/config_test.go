package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
	t.Setenv("NECHAMA_CONFIG", filepath.Join(t.TempDir(), "missing.json"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.TransliterationProvider() != "ollama" {
		t.Fatalf("unexpected provider: %q", cfg.TransliterationProvider())
	}
}

func TestLoadReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "transliteration": {
    "provider": "ollama",
    "ollama": {
      "base_url": "http://localhost:11434",
      "model": "qwen2.5:7b",
      "timeout_seconds": 10
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("NECHAMA_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.Transliteration.Ollama.Model, "qwen2.5:7b"; got != want {
		t.Fatalf("unexpected model: got %q want %q", got, want)
	}
}

func TestValidateRejectsUnknownProvider(t *testing.T) {
	cfg := Default()
	cfg.Transliteration.Provider = "unknown"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestDefaultModeIsHybrid(t *testing.T) {
	cfg := Default()
	if got, want := cfg.TransliterationMode(), ModeHybrid; got != want {
		t.Fatalf("default mode: got %q want %q", got, want)
	}
}

func TestModeEnvOverride(t *testing.T) {
	t.Setenv("NECHAMA_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("NECHAMA_TRANSLITERATION_MODE", "model")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := cfg.TransliterationMode(), ModeModel; got != want {
		t.Fatalf("mode from env: got %q want %q", got, want)
	}
}

func TestValidateRejectsUnknownMode(t *testing.T) {
	cfg := Default()
	cfg.Transliteration.Mode = "magic"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestHybridModeDoesNotRequireOllama(t *testing.T) {
	cfg := Default()
	cfg.Transliteration.Mode = ModeHybrid
	cfg.Transliteration.Ollama.BaseURL = ""
	cfg.Transliteration.Ollama.Model = ""
	cfg.Transliteration.Ollama.TimeoutSeconds = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("hybrid mode should not require ollama config: %v", err)
	}
}

func TestModelModeRequiresOllama(t *testing.T) {
	cfg := Default()
	cfg.Transliteration.Mode = ModeModel
	cfg.Transliteration.Ollama.BaseURL = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("model mode should require base_url")
	}
	cfg.Transliteration.Ollama.BaseURL = "http://localhost:11434"
	cfg.Transliteration.Ollama.Model = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("model mode should require model")
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("NECHAMA_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("NECHAMA_TRANSLITERATION_PROVIDER", "ollama")
	t.Setenv("NECHAMA_TRANSLITERATION_BASE_URL", "https://ollama.example.com")
	t.Setenv("NECHAMA_TRANSLITERATION_MODEL", "gemma4:31b-cloud")
	t.Setenv("NECHAMA_TRANSLITERATION_API_KEY", "secret-test-key")
	t.Setenv("NECHAMA_TRANSLITERATION_TIMEOUT_SECONDS", "75")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.TransliterationProvider(), "ollama"; got != want {
		t.Fatalf("unexpected provider: got %q want %q", got, want)
	}
	if got, want := cfg.Transliteration.Ollama.BaseURL, "https://ollama.example.com"; got != want {
		t.Fatalf("unexpected base_url: got %q want %q", got, want)
	}
	if got, want := cfg.Transliteration.Ollama.Model, "gemma4:31b-cloud"; got != want {
		t.Fatalf("unexpected model: got %q want %q", got, want)
	}
	if got, want := cfg.Transliteration.Ollama.APIKey, "secret-test-key"; got != want {
		t.Fatalf("unexpected api_key: got %q want %q", got, want)
	}
	if got, want := cfg.Transliteration.Ollama.TimeoutSeconds, 75; got != want {
		t.Fatalf("unexpected timeout_seconds: got %d want %d", got, want)
	}
}

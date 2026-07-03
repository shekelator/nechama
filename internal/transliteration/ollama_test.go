package transliteration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaClientGenerate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/chat"; got != want {
			t.Fatalf("unexpected path: got %q want %q", got, want)
		}
		if got := r.Method; got != http.MethodPost {
			t.Fatalf("unexpected method: %s", got)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer test-api-key"; got != want {
			t.Fatalf("unexpected auth header: got %q want %q", got, want)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if payload["model"] != "llama3.2:3b" {
			t.Fatalf("unexpected model: %v", payload["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"shalom"}}`))
	}))
	defer server.Close()

	client, err := NewOllamaClient(
		WithOllamaBaseURL(server.URL),
		WithOllamaModel("llama3.2:3b"),
		WithOllamaAPIKey("test-api-key"),
		WithOllamaHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewOllamaClient() error = %v", err)
	}

	got, err := client.Generate(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got != "shalom" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestOllamaClientGenerateReturnsHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := NewOllamaClient(
		WithOllamaBaseURL(server.URL),
		WithOllamaHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewOllamaClient() error = %v", err)
	}

	_, err = client.Generate(context.Background(), "sys", "user")
	if err == nil || !strings.Contains(err.Error(), "ollama API returned") {
		t.Fatalf("expected ollama API error, got %v", err)
	}
}

func TestFactoryBuildsOllamaProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"shalom"}}`))
	}))
	defer server.Close()

	provider, err := NewProviderFromConfig(FactoryConfig{
		Provider: ProviderOllama,
		Ollama: OllamaFactoryConfig{
			BaseURL: server.URL,
			Model:   "llama3.2:3b",
			APIKey:  "factory-key",
		},
	})
	if err != nil {
		t.Fatalf("NewProviderFromConfig() error = %v", err)
	}

	got, err := provider.Generate(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got != "shalom" {
		t.Fatalf("unexpected output: %q", got)
	}
}

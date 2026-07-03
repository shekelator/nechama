package transliteration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OllamaClient struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

type OllamaOption func(*OllamaClient)

func WithOllamaBaseURL(baseURL string) OllamaOption {
	return func(client *OllamaClient) {
		if strings.TrimSpace(baseURL) != "" {
			client.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

func WithOllamaModel(model string) OllamaOption {
	return func(client *OllamaClient) {
		if strings.TrimSpace(model) != "" {
			client.model = strings.TrimSpace(model)
		}
	}
}

func WithOllamaAPIKey(apiKey string) OllamaOption {
	return func(client *OllamaClient) {
		if apiKey != "" {
			client.apiKey = strings.TrimSpace(apiKey)
		}
	}
}

func WithOllamaHTTPClient(httpClient *http.Client) OllamaOption {
	return func(client *OllamaClient) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func NewOllamaClient(options ...OllamaOption) (*OllamaClient, error) {
	client := &OllamaClient{
		baseURL:    "http://127.0.0.1:11434",
		model:      "gemma4:e4b",
		apiKey:     "dummy-api-key",
		httpClient: http.DefaultClient,
	}

	for _, option := range options {
		option(client)
	}

	if client.httpClient == nil {
		return nil, errors.New("http client is required")
	}
	if strings.TrimSpace(client.model) == "" {
		return nil, errors.New("ollama model is required")
	}

	return client, nil
}

func (c *OllamaClient) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	body := ollamaChatRequest{
		Model:  c.model,
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	endpoint := c.baseURL + "/api/chat"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if strings.TrimSpace(c.apiKey) != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return "", fmt.Errorf("ollama API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var parsed ollamaChatResponse
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return "", err
	}

	return parsed.Message.Content, nil
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

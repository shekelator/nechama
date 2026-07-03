package transliteration

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrEmptyOutput = errors.New("transliteration provider returned empty output")

type Request struct {
	Text           string
	LanguageFamily string
	ActualLanguage string
}

type Provider interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type Service struct {
	provider Provider
	rules    string
}

func NewService(provider Provider, rules string) (*Service, error) {
	if provider == nil {
		return nil, errors.New("provider is required")
	}
	if strings.TrimSpace(rules) == "" {
		return nil, errors.New("transliteration rules are required")
	}

	return &Service{provider: provider, rules: rules}, nil
}

func (s *Service) Transliterate(ctx context.Context, req Request) (string, error) {
	input := strings.TrimSpace(req.Text)
	if input == "" {
		return "", errors.New("text is required")
	}

	systemPrompt := buildSystemPrompt(s.rules)
	userPrompt := buildUserPrompt(req, input)

	output, err := s.provider.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return "", ErrEmptyOutput
	}

	return output, nil
}

func buildSystemPrompt(rules string) string {
	return strings.TrimSpace(fmt.Sprintf(`You are a precise Jewish text transliteration engine.
Return only transliterated text in Latin letters and preserve line breaks exactly.
Do not add explanations, notes, brackets, numbering, or metadata.

Transliteration rules:
%s`, strings.TrimSpace(rules)))
}

func buildUserPrompt(req Request, input string) string {
	return strings.TrimSpace(fmt.Sprintf(`Language family: %s
Actual language: %s

Transliterate this text:
%s`, req.LanguageFamily, req.ActualLanguage, input))
}

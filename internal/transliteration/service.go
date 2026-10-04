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

	return normalizeTransliterationOutput(output), nil
}

func buildSystemPrompt(rules string) string {
	return strings.TrimSpace(fmt.Sprintf(`You are a precise Jewish text transliteration engine.
Return only transliterated text in Latin letters and preserve line breaks exactly.
Do not add explanations, notes, brackets, numbering, or metadata.
Do not drop vowels or collapse them into bare consonant clusters.
Do not output bare clusters such as kl or chk when the source contains a visible vowel mark.
When the source has a visible vowel mark, the output must keep a vowel letter.
This is especially important for cholam and qamatz-katan: prefer chol/khol and chok/kol-style forms when the rules call for them.
Do not use hyphens unless they come from Hebrew maqaf (־).
Do not preserve Hebrew verse punctuation such as sof pasuq (׃) in the output. Instead, add periods where appropriate.
If the engine output suggests an apostrophe or hyphen, preserve it as a plain ASCII apostrophe or hyphen.

Transliteration rules:
%s`, strings.TrimSpace(rules)))
}

func buildUserPrompt(req Request, input string) string {
	return strings.TrimSpace(fmt.Sprintf(`Language family: %s
Actual language: %s

Transliterate this text:
%s`, req.LanguageFamily, req.ActualLanguage, input))
}

func normalizeTransliterationOutput(text string) string {
	text = strings.NewReplacer(
		"־", "-",
		"‐", "-",
		"‑", "-",
		"‒", "-",
		"–", "-",
		"—", "-",
		"−", "-",
		"׃", ".",
		"‘", "'",
		"’", "'",
		"ʼ", "'",
		"ʾ", "'",
		"ʿ", "'",
	).Replace(text)
	return text
}

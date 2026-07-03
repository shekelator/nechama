package transliteration

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	generate func(context.Context, string, string) (string, error)
}

func (f fakeProvider) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return f.generate(ctx, systemPrompt, userPrompt)
}

func TestServiceTransliterateBuildsPrompts(t *testing.T) {
	t.Parallel()

	service, err := NewService(fakeProvider{
		generate: func(_ context.Context, systemPrompt, userPrompt string) (string, error) {
			if systemPrompt == "" {
				t.Fatal("expected non-empty system prompt")
			}
			if userPrompt == "" {
				t.Fatal("expected non-empty user prompt")
			}
			return " bereshit ", nil
		},
	}, DefaultRules)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	got, err := service.Transliterate(context.Background(), Request{
		Text:           "בְּרֵאשִׁית",
		LanguageFamily: "hebrew",
		ActualLanguage: "he",
	})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}

	if got != "bereshit" {
		t.Fatalf("unexpected transliteration: %q", got)
	}
}

func TestServiceTransliterateReturnsProviderError(t *testing.T) {
	t.Parallel()

	want := errors.New("boom")
	service, err := NewService(fakeProvider{
		generate: func(context.Context, string, string) (string, error) {
			return "", want
		},
	}, DefaultRules)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Transliterate(context.Background(), Request{Text: "שלום"})
	if !errors.Is(err, want) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestServiceTransliterateRejectsEmptyOutput(t *testing.T) {
	t.Parallel()

	service, err := NewService(fakeProvider{
		generate: func(context.Context, string, string) (string, error) {
			return "   ", nil
		},
	}, DefaultRules)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Transliterate(context.Background(), Request{Text: "שלום"})
	if !errors.Is(err, ErrEmptyOutput) {
		t.Fatalf("expected ErrEmptyOutput, got %v", err)
	}
}

package cmd

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shekelator/nechama/internal/sefaria"
	"github.com/shekelator/nechama/internal/transliteration"
)

type stubTextService struct {
	fetch func(context.Context, sefaria.FetchRequest) (sefaria.Text, error)
	list  func(context.Context, string) ([]sefaria.VersionChoice, error)
}

type stubTransliterator struct {
	transliterate func(context.Context, string) (string, error)
}

func (s stubTextService) FetchText(ctx context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
	return s.fetch(ctx, req)
}

func (s stubTextService) ListEnglishVersions(ctx context.Context, ref string) ([]sefaria.VersionChoice, error) {
	return s.list(ctx, ref)
}

func (s stubTransliterator) Transliterate(ctx context.Context, req transliteration.Request) (string, error) {
	return s.transliterate(ctx, req.Text)
}

func TestRootCommandFetchesSourceTextByDefault(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest
	stdout := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "בראשית"}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called")
				return nil, nil
			},
		},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Language != sefaria.LanguageSource {
		t.Fatalf("expected source language, got %q", captured.Language)
	}

	if got := stdout.String(); got != "בראשית\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestRootCommandReadsReferenceFromStdinWhenNoArgument(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest
	stdout := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "ok"}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called")
				return nil, nil
			},
		},
		stdin:      strings.NewReader("Psalm 51:4\n"),
		stdout:     stdout,
		stderr:     &bytes.Buffer{},
		isTTY:      func() bool { return false },
		isInputTTY: func() bool { return false },
	})

	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Ref != "Psalm 51:4" {
		t.Fatalf("expected ref from stdin, got %q", captured.Ref)
	}
	if got := stdout.String(); got != "ok\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestFetchCommandSelectsRequestedTranslation(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list: func(_ context.Context, ref string) ([]sefaria.VersionChoice, error) {
				if ref != "Genesis 1:1" {
					t.Fatalf("unexpected ref: %q", ref)
				}
				return []sefaria.VersionChoice{
					{VersionTitle: "THE JPS TANAKH: Gender-Sensitive Edition", ShortVersionTitle: "Revised JPS, 2023"},
					{VersionTitle: "The Holy Scriptures: A New Translation (JPS 1917)", ShortVersionTitle: "JPS 1917"},
				}, nil
			},
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				if req.Language != sefaria.LanguageEnglish {
					t.Fatalf("expected english request, got %q", req.Language)
				}
				if req.TranslationTitle != "The Holy Scriptures: A New Translation (JPS 1917)" {
					t.Fatalf("unexpected translation title: %q", req.TranslationTitle)
				}
				return sefaria.Text{Text: "In the beginning"}, nil
			},
		},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"fetch", "--translation", "jps 1917", "Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := stdout.String(); got != "In the beginning\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestFetchCommandReadsReferenceFromStdinWhenNoArgument(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest
	stdout := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "ok"}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called")
				return nil, nil
			},
		},
		stdin:      strings.NewReader("Psalm 51:4"),
		stdout:     stdout,
		stderr:     &bytes.Buffer{},
		isTTY:      func() bool { return false },
		isInputTTY: func() bool { return false },
	})

	cmd.SetArgs([]string{"fetch"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Ref != "Psalm 51:4" {
		t.Fatalf("expected ref from stdin, got %q", captured.Ref)
	}
	if got := stdout.String(); got != "ok\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestFetchCommandPromptsForTranslationChoice(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				return []sefaria.VersionChoice{
					{VersionTitle: "THE JPS TANAKH: Gender-Sensitive Edition", ShortVersionTitle: "Revised JPS, 2023"},
					{VersionTitle: "The Holy Scriptures: A New Translation (JPS 1917)", ShortVersionTitle: "JPS 1917"},
				}, nil
			},
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				if req.TranslationTitle != "The Holy Scriptures: A New Translation (JPS 1917)" {
					t.Fatalf("unexpected translation title: %q", req.TranslationTitle)
				}
				return sefaria.Text{Text: "In the beginning"}, nil
			},
		},
		stdin:  strings.NewReader("2\n"),
		stdout: stdout,
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return true },
	})

	cmd.SetArgs([]string{"fetch", "--choose-translation", "Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Available English translations:") {
		t.Fatalf("expected prompt output, got %q", stdout.String())
	}
}

func TestFetchCommandRejectsInteractiveChoiceWithoutTTY(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list:  func(context.Context, string) ([]sefaria.VersionChoice, error) { return nil, nil },
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) { return sefaria.Text{}, nil },
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"fetch", "--choose-translation", "Genesis 1:1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("expected interactive terminal error, got %v", err)
	}
}

func TestFetchCommandRejectsConflictingTranslationFlags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called")
				return nil, nil
			},
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) {
				t.Fatal("fetch should not be called")
				return sefaria.Text{}, nil
			},
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return true },
	})

	cmd.SetArgs([]string{"fetch", "--translation", "JPS 1917", "--choose-translation", "Genesis 1:1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("expected conflicting flags error, got %v", err)
	}
}

func TestFetchCommandWritesToFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	output := filepath.Join(dir, "genesis.txt")

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				return sefaria.Text{Text: "When God began"}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) { return nil, nil },
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"--output", output, "Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if got := string(contents); got != "When God began\n" {
		t.Fatalf("unexpected file contents: %q", got)
	}
}

func TestFetchCommandTransliteratesSourceText(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				if req.Language != sefaria.LanguageSource {
					t.Fatalf("expected source request, got %q", req.Language)
				}
				return sefaria.Text{
					Text:           "מִזְמ֥וֹר",
					LanguageFamily: "hebrew",
					ActualLanguage: "he",
					IsSource:       true,
				}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called")
				return nil, nil
			},
		},
		newTransliterator: func(_ *slog.Logger) (transliterator, error) {
			return stubTransliterator{
				transliterate: func(_ context.Context, text string) (string, error) {
					if text != "מִזְמ֥וֹר" {
						t.Fatalf("unexpected text: %q", text)
					}
					return "mizmor", nil
				},
			}, nil
		},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"fetch", "--transliteration", "Psalm 132:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := stdout.String(); got != "mizmor\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestFetchCommandRejectsTransliterationWithEnglishFlags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list:  func(context.Context, string) ([]sefaria.VersionChoice, error) { return nil, nil },
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) { return sefaria.Text{}, nil },
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"fetch", "--transliteration", "--english", "Genesis 1:1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "can only be used with source text") {
		t.Fatalf("expected transliteration conflict error, got %v", err)
	}
}

func TestFetchCommandFailsWhenTransliterationIsUnavailable(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) {
				return sefaria.Text{Text: "דָּבָר", LanguageFamily: "hebrew", ActualLanguage: "he", IsSource: true}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) { return nil, nil },
		},
		newTransliterator: nil,
		stdin:             strings.NewReader(""),
		stdout:            &bytes.Buffer{},
		stderr:            &bytes.Buffer{},
		isTTY:             func() bool { return false },
	})

	cmd.SetArgs([]string{"fetch", "--transliteration", "Genesis 1:1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected transliteration unavailable error, got %v", err)
	}
}

func TestFetchCommandPrintsSourceDetailsToStderr(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest
	stderr := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "בראשית"}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called")
				return nil, nil
			},
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: stderr,
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Language != sefaria.LanguageSource {
		t.Fatalf("expected source language, got %q", captured.Language)
	}
	if got := stderr.String(); !strings.Contains(got, "Genesis 1:1 (source)") {
		t.Fatalf("expected source details on stderr, got %q", got)
	}
}

func TestFetchCommandEnglishUsesDefaultTranslationWhenAvailable(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest
	stderr := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list: func(_ context.Context, ref string) ([]sefaria.VersionChoice, error) {
				if ref != "Genesis 1:1" {
					t.Fatalf("unexpected ref: %q", ref)
				}
				return []sefaria.VersionChoice{
					{VersionTitle: "THE JPS TANAKH: Gender-Sensitive Edition", ShortVersionTitle: "Revised JPS, 2023"},
					{VersionTitle: "The Holy Scriptures: A New Translation (JPS 1917)", ShortVersionTitle: "JPS 1917"},
				}, nil
			},
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "In the beginning"}, nil
			},
		},
		stdin:                     strings.NewReader(""),
		stdout:                    &bytes.Buffer{},
		stderr:                    stderr,
		isTTY:                     func() bool { return false },
		defaultEnglishTranslation: "Revised JPS, 2023",
	})

	cmd.SetArgs([]string{"fetch", "--english", "Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Language != sefaria.LanguageEnglish {
		t.Fatalf("expected english language, got %q", captured.Language)
	}
	if captured.TranslationTitle != "THE JPS TANAKH: Gender-Sensitive Edition" {
		t.Fatalf("expected default translation title, got %q", captured.TranslationTitle)
	}
	if got := stderr.String(); !strings.Contains(got, "English, THE JPS TANAKH: Gender-Sensitive Edition") {
		t.Fatalf("expected default translation in stderr details, got %q", got)
	}
	if strings.Contains(stderr.String(), "not available") {
		t.Fatalf("did not expect fallback warning, got %q", stderr.String())
	}
}

func TestFetchCommandEnglishFallsBackWhenDefaultTranslationUnavailable(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest
	stderr := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				return []sefaria.VersionChoice{
					{VersionTitle: "The Holy Scriptures: A New Translation (JPS 1917)", ShortVersionTitle: "JPS 1917"},
				}, nil
			},
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "In the beginning"}, nil
			},
		},
		stdin:                     strings.NewReader(""),
		stdout:                    &bytes.Buffer{},
		stderr:                    stderr,
		isTTY:                     func() bool { return false },
		defaultEnglishTranslation: "Revised JPS, 2023",
	})

	cmd.SetArgs([]string{"fetch", "--english", "Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Language != sefaria.LanguageEnglish {
		t.Fatalf("expected english language, got %q", captured.Language)
	}
	if captured.TranslationTitle != "" {
		t.Fatalf("expected highest-priority fallback (empty title), got %q", captured.TranslationTitle)
	}
	if got := stderr.String(); !strings.Contains(got, "not available") || !strings.Contains(got, "highest-priority") {
		t.Fatalf("expected fallback warning + highest-priority details on stderr, got %q", got)
	}
}

func TestFetchCommandTranslationFlagOverridesDefault(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				return []sefaria.VersionChoice{
					{VersionTitle: "THE JPS TANAKH: Gender-Sensitive Edition", ShortVersionTitle: "Revised JPS, 2023"},
					{VersionTitle: "The Holy Scriptures: A New Translation (JPS 1917)", ShortVersionTitle: "JPS 1917"},
				}, nil
			},
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "In the beginning"}, nil
			},
		},
		stdin:                     strings.NewReader(""),
		stdout:                    &bytes.Buffer{},
		stderr:                    &bytes.Buffer{},
		isTTY:                     func() bool { return false },
		defaultEnglishTranslation: "Revised JPS, 2023",
	})

	cmd.SetArgs([]string{"fetch", "--translation", "jps 1917", "Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.TranslationTitle != "The Holy Scriptures: A New Translation (JPS 1917)" {
		t.Fatalf("expected --translation flag to win, got %q", captured.TranslationTitle)
	}
}

func TestFetchCommandDefaultTranslationDoesNotToggleEnglish(t *testing.T) {
	t.Parallel()

	var captured sefaria.FetchRequest

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(_ context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
				captured = req
				return sefaria.Text{Text: "בראשית"}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called without --english")
				return nil, nil
			},
		},
		stdin:                     strings.NewReader(""),
		stdout:                    &bytes.Buffer{},
		stderr:                    &bytes.Buffer{},
		isTTY:                     func() bool { return false },
		defaultEnglishTranslation: "Revised JPS, 2023",
	})

	cmd.SetArgs([]string{"Genesis 1:1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Language != sefaria.LanguageSource {
		t.Fatalf("default translation env var must not toggle english, got %q", captured.Language)
	}
	if captured.TranslationTitle != "" {
		t.Fatalf("expected empty translation title for source fetch, got %q", captured.TranslationTitle)
	}
}

func TestRootCommandTransliteratesRawHebrewArgument(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) {
				t.Fatal("fetch should not be called for raw Hebrew input")
				return sefaria.Text{}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called for raw Hebrew input")
				return nil, nil
			},
		},
		newTransliterator: func(_ *slog.Logger) (transliterator, error) {
			return stubTransliterator{
				transliterate: func(_ context.Context, text string) (string, error) {
					if text != "שָׁלוֹם עָלֵיכֶם" {
						t.Fatalf("unexpected text passed to transliterator: %q", text)
					}
					return "shalom aleichem", nil
				},
			}, nil
		},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: stderr,
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"שָׁלוֹם עָלֵיכֶם"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := stdout.String(); got != "shalom aleichem\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "(transliterate)") {
		t.Fatalf("expected transliterate details on stderr, got %q", got)
	}
}

func TestFetchCommandTransliteratesRawHebrewFromStdin(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) {
				t.Fatal("fetch should not be called for raw Hebrew input")
				return sefaria.Text{}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called for raw Hebrew input")
				return nil, nil
			},
		},
		newTransliterator: func(_ *slog.Logger) (transliterator, error) {
			return stubTransliterator{
				transliterate: func(_ context.Context, text string) (string, error) {
					if text != "בְּרֵאשִׁית" {
						t.Fatalf("unexpected text passed to transliterator: %q", text)
					}
					return "bereshit", nil
				},
			}, nil
		},
		stdin:      strings.NewReader("בְּרֵאשִׁית\n"),
		stdout:     stdout,
		stderr:     &bytes.Buffer{},
		isTTY:      func() bool { return false },
		isInputTTY: func() bool { return false },
	})

	cmd.SetArgs([]string{"fetch"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := stdout.String(); got != "bereshit\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestFetchCommandRawHebrewRejectsEnglishFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) {
				t.Fatal("fetch should not be called")
				return sefaria.Text{}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) {
				t.Fatal("list should not be called")
				return nil, nil
			},
		},
		newTransliterator: func(_ *slog.Logger) (transliterator, error) {
			t.Fatal("transliterator should not be constructed on flag conflict")
			return nil, nil
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"--english", "שָׁלוֹם"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be combined with --english") {
		t.Fatalf("expected hebrew+english conflict error, got %v", err)
	}
}

func TestFetchCommandRawHebrewRespectsOutputFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	output := filepath.Join(dir, "translit.txt")

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) {
				t.Fatal("fetch should not be called for raw Hebrew input")
				return sefaria.Text{}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) { return nil, nil },
		},
		newTransliterator: func(_ *slog.Logger) (transliterator, error) {
			return stubTransliterator{
				transliterate: func(context.Context, string) (string, error) {
					return "shalom", nil
				},
			}, nil
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		isTTY:  func() bool { return false },
	})

	cmd.SetArgs([]string{"--output", output, "שָׁלוֹם"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := string(contents); got != "shalom\n" {
		t.Fatalf("unexpected file contents: %q", got)
	}
}

func TestFetchCommandRawHebrewFailsWhenTransliterationUnavailable(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand(commandDependencies{
		service: stubTextService{
			fetch: func(context.Context, sefaria.FetchRequest) (sefaria.Text, error) {
				t.Fatal("fetch should not be called for raw Hebrew input")
				return sefaria.Text{}, nil
			},
			list: func(context.Context, string) ([]sefaria.VersionChoice, error) { return nil, nil },
		},
		newTransliterator: nil,
		stdin:             strings.NewReader(""),
		stdout:            &bytes.Buffer{},
		stderr:            &bytes.Buffer{},
		isTTY:             func() bool { return false },
	})

	cmd.SetArgs([]string{"שָׁלוֹם"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected transliteration unavailable error, got %v", err)
	}
}

// writeHybridConfig writes a config that selects hybrid mode with no LLM
// provider configured, so the command runs the deterministic engine only.
func writeHybridConfig(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	content := `{"transliteration":{"provider":"ollama","mode":"hybrid","ollama":{"base_url":"","model":"","api_key":"","timeout_seconds":0}}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

// TestCommandUsesHybridEngineWithoutProvider verifies that with hybrid mode
// and no LLM configured, the command transliterates raw Hebrew using only the
// deterministic engine (no network).
func TestCommandUsesHybridEngineWithoutProvider(t *testing.T) {
	t.Setenv("NECHAMA_CONFIG", writeHybridConfig(t, t.TempDir()))

	stdout := &bytes.Buffer{}
	deps := defaultDependencies()
	deps.stdin = strings.NewReader("")
	deps.stdout = stdout
	deps.stderr = &bytes.Buffer{}
	deps.isTTY = func() bool { return false }
	deps.isInputTTY = func() bool { return false }

	cmd := newRootCommand(deps)
	cmd.SetArgs([]string{"שְׁמַע מִדְבָּר"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := stdout.String(), "Shema midbar\n"; got != want {
		t.Fatalf("unexpected stdout: got %q want %q", got, want)
	}
}

// TestCommandDebugLoggingEmitted verifies --debug routes engine/hybrid debug
// logs to stderr.
func TestCommandDebugLoggingEmitted(t *testing.T) {
	t.Setenv("NECHAMA_CONFIG", writeHybridConfig(t, t.TempDir()))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := defaultDependencies()
	deps.stdin = strings.NewReader("")
	deps.stdout = stdout
	deps.stderr = stderr
	deps.isTTY = func() bool { return false }
	deps.isInputTTY = func() bool { return false }

	cmd := newRootCommand(deps)
	cmd.SetArgs([]string{"--debug", "שְׁמַע מִדְבָּר"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "level=DEBUG") {
		t.Fatalf("expected debug log lines on stderr, got %q", stderr.String())
	}
}

// TestCommandNoDebugLoggingByDefault verifies that without --debug, no
// debug-level log lines are written to stderr.
func TestCommandNoDebugLoggingByDefault(t *testing.T) {
	t.Setenv("NECHAMA_CONFIG", writeHybridConfig(t, t.TempDir()))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := defaultDependencies()
	deps.stdin = strings.NewReader("")
	deps.stdout = stdout
	deps.stderr = stderr
	deps.isTTY = func() bool { return false }
	deps.isInputTTY = func() bool { return false }

	cmd := newRootCommand(deps)
	cmd.SetArgs([]string{"שְׁמַע מִדְבָּר"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.Contains(stderr.String(), "level=DEBUG") {
		t.Fatalf("expected no debug log lines without --debug, got %q", stderr.String())
	}
}

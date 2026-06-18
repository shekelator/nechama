package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shekelator/nechama/internal/sefaria"
)

type stubTextService struct {
	fetch func(context.Context, sefaria.FetchRequest) (sefaria.Text, error)
	list  func(context.Context, string) ([]sefaria.VersionChoice, error)
}

func (s stubTextService) FetchText(ctx context.Context, req sefaria.FetchRequest) (sefaria.Text, error) {
	return s.fetch(ctx, req)
}

func (s stubTextService) ListEnglishVersions(ctx context.Context, ref string) ([]sefaria.VersionChoice, error) {
	return s.list(ctx, ref)
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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode"

	"github.com/shekelator/nechama/internal/config"
	"github.com/shekelator/nechama/internal/sefaria"
	"github.com/shekelator/nechama/internal/transliteration"
	"golang.org/x/term"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var Version = "dev"

type textService interface {
	FetchText(ctx context.Context, req sefaria.FetchRequest) (sefaria.Text, error)
	ListEnglishVersions(ctx context.Context, ref string) ([]sefaria.VersionChoice, error)
}

type fetchOptions struct {
	english           bool
	translation       string
	chooseTranslation bool
	transliteration   bool
	debug             bool
	outputPath        string
}

type transliterator interface {
	Transliterate(ctx context.Context, req transliteration.Request) (string, error)
}

type commandDependencies struct {
	service                   textService
	newTransliterator         func(logger *slog.Logger) (transliterator, error)
	stdin                     io.Reader
	stdout                    io.Writer
	stderr                    io.Writer
	isTTY                     func() bool
	isInputTTY                func() bool
	defaultEnglishTranslation string
}

func Execute() error {
	return newRootCommand(defaultDependencies()).Execute()
}

func defaultDependencies() commandDependencies {
	newTransliterator := func(logger *slog.Logger) (transliterator, error) {
		appConfig, err := config.Load()
		if err != nil {
			return nil, err
		}

		factoryCfg := transliteration.FactoryConfig{
			Provider: appConfig.TransliterationProvider(),
			Ollama: transliteration.OllamaFactoryConfig{
				BaseURL: appConfig.Transliteration.Ollama.BaseURL,
				Model:   appConfig.Transliteration.Ollama.Model,
				APIKey:  appConfig.Transliteration.Ollama.APIKey,
				Timeout: appConfig.OllamaTimeout(),
			},
		}

		switch appConfig.TransliterationMode() {
		case config.ModeModel:
			// Model mode sends the whole text to the LLM, so a configured
			// provider is required.
			provider, err := transliteration.NewProviderFromConfig(factoryCfg)
			if err != nil {
				return nil, err
			}
			return transliteration.NewService(provider, transliteration.DefaultRules)
		default:
			// Hybrid mode (default): a deterministic engine does the work and
			// the LLM is consulted only for ambiguous words. The provider is
			// optional; if unconfigured, the hybrid runs network-free.
			return transliteration.NewHybridFromConfig(factoryCfg, transliteration.DefaultRules, logger)
		}
	}

	return commandDependencies{
		service: sefaria.NewClient(
			sefaria.WithBaseURL(os.Getenv("NECHAMA_SEFARIA_BASE_URL")),
			sefaria.WithUserAgent(fmt.Sprintf("nechama/%s", Version)),
		),
		newTransliterator:         newTransliterator,
		stdin:                     os.Stdin,
		stdout:                    os.Stdout,
		stderr:                    os.Stderr,
		defaultEnglishTranslation: os.Getenv("NECHAMA_DEFAULT_ENGLISH_TRANSLATION"),
		isTTY: func() bool {
			return isTerminal(os.Stdin) && isTerminal(os.Stdout)
		},
		isInputTTY: func() bool {
			return isTerminal(os.Stdin)
		},
	}
}

func isTerminal(file *os.File) bool {
	return term.IsTerminal(int(file.Fd()))
}

// debugEnabled reports whether debug logging should be active. The --debug
// flag wins; otherwise NECHAMA_DEBUG is consulted (accepts 1/true/yes/on).
func debugEnabled(flag bool) bool {
	if flag {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NECHAMA_DEBUG"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// newLogger returns a debug-level text logger writing to w, or nil when debug
// is disabled (the transliteration services treat nil as a no-op logger).
func newLogger(debug bool, w io.Writer) *slog.Logger {
	if !debug {
		return nil
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func newRootCommand(deps commandDependencies) *cobra.Command {
	opts := fetchOptions{}

	cmd := &cobra.Command{
		Use:           "nechama [ref]",
		Short:         "Fetch Jewish texts from Sefaria",
		Long:          "Nechama fetches plain-text excerpts from Sefaria and prints them to stdout or saves them to a file.",
		Example:       "  nechama \"Genesis 1:1\"\n  nechama --english \"Genesis 1:1\"\n  nechama --transliteration \"Psalm 132\"\n  nechama fetch --translation \"Revised JPS, 2023\" \"Genesis 1\"",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := resolveReference(args, deps)
			if err != nil {
				if len(args) == 0 && inputIsTTY(deps) {
					return cmd.Help()
				}
				return err
			}
			return runFetch(cmd.Context(), deps, opts, ref)
		},
	}

	cmd.SetOut(deps.stdout)
	cmd.SetErr(deps.stderr)
	bindFetchFlags(cmd.Flags(), &opts)

	cmd.AddCommand(newFetchCommand(deps), newVersionCommand(deps.stdout))

	return cmd
}

func newFetchCommand(deps commandDependencies) *cobra.Command {
	opts := fetchOptions{}

	cmd := &cobra.Command{
		Use:           "fetch [ref]",
		Short:         "Fetch a text from Sefaria",
		Example:       "  nechama fetch \"Berakhot 2a:1\"\n  nechama fetch --english --choose-translation \"Genesis 1:1\"\n  nechama fetch --transliteration \"Psalm 132\"\n  nechama fetch -o genesis.txt \"Genesis 1\"",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := resolveReference(args, deps)
			if err != nil {
				if len(args) == 0 && inputIsTTY(deps) {
					return cmd.Help()
				}
				return err
			}
			return runFetch(cmd.Context(), deps, opts, ref)
		},
	}

	cmd.SetOut(deps.stdout)
	cmd.SetErr(deps.stderr)
	bindFetchFlags(cmd.Flags(), &opts)

	return cmd
}

func bindFetchFlags(flags *pflag.FlagSet, opts *fetchOptions) {
	flags.BoolVarP(&opts.english, "english", "e", false, "Fetch the highest-priority English translation")
	flags.StringVarP(&opts.translation, "translation", "t", "", "Fetch a specific English translation by short or full title")
	flags.BoolVar(&opts.chooseTranslation, "choose-translation", false, "Interactively choose an English translation")
	flags.BoolVar(&opts.transliteration, "transliteration", false, "Transliterate source Hebrew/Aramaic text into Latin letters")
	flags.BoolVar(&opts.debug, "debug", false, "Enable debug logging to stderr (also: NECHAMA_DEBUG)")
	flags.StringVarP(&opts.outputPath, "output", "o", "", "Write the fetched text to a file instead of stdout")
}

func resolveReference(args []string, deps commandDependencies) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if inputIsTTY(deps) {
		return "", errors.New("reference or text is required")
	}

	data, err := io.ReadAll(deps.stdin)
	if err != nil {
		return "", err
	}

	ref := strings.TrimSpace(string(data))
	if ref == "" {
		return "", errors.New("reference or text is required")
	}
	return ref, nil
}

func inputIsTTY(deps commandDependencies) bool {
	if deps.isInputTTY == nil {
		return false
	}
	return deps.isInputTTY()
}

func runFetch(ctx context.Context, deps commandDependencies, opts fetchOptions, ref string) error {
	if opts.translation != "" && opts.chooseTranslation {
		return errors.New("--translation and --choose-translation cannot be used together")
	}
	if opts.transliteration && (opts.english || opts.translation != "" || opts.chooseTranslation) {
		return errors.New("--transliteration can only be used with source text (not --english, --translation, or --choose-translation)")
	}

	// Input that contains Hebrew script is treated as raw text to transliterate
	// directly, bypassing the Sefaria lookup entirely.
	if containsHebrewScript(ref) {
		return runTransliterateText(ctx, deps, opts, ref)
	}

	request := sefaria.FetchRequest{Ref: ref, Language: sefaria.LanguageSource}

	if opts.english || opts.translation != "" || opts.chooseTranslation {
		request.Language = sefaria.LanguageEnglish
	}

	switch {
	case opts.translation != "":
		versions, err := deps.service.ListEnglishVersions(ctx, ref)
		if err != nil {
			return err
		}

		version, err := sefaria.MatchTranslation(versions, opts.translation)
		if err != nil {
			return err
		}

		request.TranslationTitle = version.VersionTitle
	case opts.chooseTranslation:
		if !deps.isTTY() {
			return errors.New("--choose-translation requires an interactive terminal")
		}

		versions, err := deps.service.ListEnglishVersions(ctx, ref)
		if err != nil {
			return err
		}

		version, err := chooseTranslation(deps.stdin, deps.stdout, versions)
		if err != nil {
			return err
		}

		request.TranslationTitle = version.VersionTitle
	case opts.english && deps.defaultEnglishTranslation != "":
		versions, err := deps.service.ListEnglishVersions(ctx, ref)
		if err != nil {
			return err
		}

		version, err := sefaria.MatchTranslation(versions, deps.defaultEnglishTranslation)
		if err != nil {
			fmt.Fprintf(deps.stderr, "default translation %q not available for %s (%v); using highest-priority English\n", deps.defaultEnglishTranslation, ref, err)
		} else {
			request.TranslationTitle = version.VersionTitle
		}
	}

	fmt.Fprintln(deps.stderr, describeFetchRequest(request))

	result, err := deps.service.FetchText(ctx, request)
	if err != nil {
		return err
	}
	if opts.transliteration {
		if !isTransliterableSource(result) {
			return errors.New("--transliteration requires source Hebrew or Aramaic text")
		}
		if deps.newTransliterator == nil {
			return errors.New("transliteration service is not configured")
		}

		service, err := deps.newTransliterator(newLogger(debugEnabled(opts.debug), deps.stderr))
		if err != nil {
			return err
		}

		transliterated, err := service.Transliterate(ctx, transliteration.Request{
			Text:           result.Text,
			LanguageFamily: result.LanguageFamily,
			ActualLanguage: result.ActualLanguage,
		})
		if err != nil {
			return err
		}
		result.Text = transliterated
	}

	content := ensureTrailingNewline(result.Text)
	return writeOutput(deps, opts, content)
}

// runTransliterateText transliterates raw Hebrew/Aramaic text supplied directly
// (as an argument or via stdin) instead of looking it up on Sefaria. It reuses
// the same transliteration provider configuration as source-text transliteration.
func runTransliterateText(ctx context.Context, deps commandDependencies, opts fetchOptions, text string) error {
	if opts.english || opts.translation != "" || opts.chooseTranslation {
		return errors.New("Hebrew text input cannot be combined with --english, --translation, or --choose-translation")
	}
	if deps.newTransliterator == nil {
		return errors.New("transliteration service is not configured")
	}

	fmt.Fprintf(deps.stderr, "%s (transliterate)\n", previewText(text, 40))

	service, err := deps.newTransliterator(newLogger(debugEnabled(opts.debug), deps.stderr))
	if err != nil {
		return err
	}

	transliterated, err := service.Transliterate(ctx, transliteration.Request{
		Text:           text,
		LanguageFamily: "hebrew",
		ActualLanguage: "he",
	})
	if err != nil {
		return err
	}

	return writeOutput(deps, opts, ensureTrailingNewline(transliterated))
}

func writeOutput(deps commandDependencies, opts fetchOptions, content string) error {
	if opts.outputPath != "" {
		return os.WriteFile(opts.outputPath, []byte(content), 0o644)
	}
	_, err := io.WriteString(deps.stdout, content)
	return err
}

// containsHebrewScript reports whether s contains any character in the
// Unicode Hebrew script. Such input is treated as raw text to transliterate
// rather than as a Sefaria reference.
func containsHebrewScript(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hebrew, r) {
			return true
		}
	}
	return false
}

// previewText returns a single-line preview of text truncated to max runes,
// suitable for a stderr status line.
func previewText(text string, max int) string {
	preview := strings.TrimSpace(text)
	if idx := strings.IndexAny(preview, "\n\r"); idx >= 0 {
		preview = preview[:idx]
	}
	if max > 0 && len([]rune(preview)) > max {
		preview = string([]rune(preview)[:max]) + "…"
	}
	return preview
}

func ensureTrailingNewline(text string) string {
	if text == "" {
		return "\n"
	}
	if text[len(text)-1] == '\n' {
		return text
	}
	return text + "\n"
}

// describeFetchRequest renders a short human-readable summary of the request
// that is printed to stderr so stdout stays clean for piping. The translation
// field reflects what was actually selected: a specific title, Sefaria's
// highest-priority English translation, or the source text.
func describeFetchRequest(req sefaria.FetchRequest) string {
	switch req.Language {
	case sefaria.LanguageEnglish:
		if req.TranslationTitle != "" {
			return fmt.Sprintf("%s (English, %s)", req.Ref, req.TranslationTitle)
		}
		return fmt.Sprintf("%s (English, highest-priority)", req.Ref)
	default:
		return fmt.Sprintf("%s (source)", req.Ref)
	}
}

func isTransliterableSource(text sefaria.Text) bool {
	if !text.IsSource {
		return false
	}

	family := strings.ToLower(strings.TrimSpace(text.LanguageFamily))
	actual := strings.ToLower(strings.TrimSpace(text.ActualLanguage))

	if family == "hebrew" || family == "aramaic" {
		return true
	}

	return actual == "he" || actual == "arc"
}

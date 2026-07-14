package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
	outputPath        string
}

type transliterator interface {
	Transliterate(ctx context.Context, req transliteration.Request) (string, error)
}

type commandDependencies struct {
	service           textService
	newTransliterator func() (transliterator, error)
	stdin             io.Reader
	stdout            io.Writer
	stderr            io.Writer
	isTTY             func() bool
	isInputTTY        func() bool
}

func Execute() error {
	return newRootCommand(defaultDependencies()).Execute()
}

func defaultDependencies() commandDependencies {
	newTransliterator := func() (transliterator, error) {
		appConfig, err := config.Load()
		if err != nil {
			return nil, err
		}

		provider, err := transliteration.NewProviderFromConfig(transliteration.FactoryConfig{
			Provider: appConfig.TransliterationProvider(),
			Ollama: transliteration.OllamaFactoryConfig{
				BaseURL: appConfig.Transliteration.Ollama.BaseURL,
				Model:   appConfig.Transliteration.Ollama.Model,
				APIKey:  appConfig.Transliteration.Ollama.APIKey,
				Timeout: appConfig.OllamaTimeout(),
			},
		})
		if err != nil {
			return nil, err
		}

		service, err := transliteration.NewService(provider, transliteration.DefaultRules)
		if err != nil {
			return nil, err
		}

		return service, nil
	}

	return commandDependencies{
		service: sefaria.NewClient(
			sefaria.WithBaseURL(os.Getenv("NECHAMA_SEFARIA_BASE_URL")),
			sefaria.WithUserAgent(fmt.Sprintf("nechama/%s", Version)),
		),
		newTransliterator: newTransliterator,
		stdin:             os.Stdin,
		stdout:            os.Stdout,
		stderr:            os.Stderr,
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

func newRootCommand(deps commandDependencies) *cobra.Command {
	opts := fetchOptions{}

	cmd := &cobra.Command{
		Use:           "nechama <ref>",
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
		Use:           "fetch <ref>",
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
	flags.StringVarP(&opts.outputPath, "output", "o", "", "Write the fetched text to a file instead of stdout")
}

func resolveReference(args []string, deps commandDependencies) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if inputIsTTY(deps) {
		return "", errors.New("reference is required")
	}

	data, err := io.ReadAll(deps.stdin)
	if err != nil {
		return "", err
	}

	ref := strings.TrimSpace(string(data))
	if ref == "" {
		return "", errors.New("reference is required")
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

	request := sefaria.FetchRequest{Ref: ref, Language: sefaria.LanguageSource}

	if opts.english || opts.translation != "" || opts.chooseTranslation {
		request.Language = sefaria.LanguageEnglish
	}

	if opts.translation != "" {
		versions, err := deps.service.ListEnglishVersions(ctx, ref)
		if err != nil {
			return err
		}

		version, err := sefaria.MatchTranslation(versions, opts.translation)
		if err != nil {
			return err
		}

		request.TranslationTitle = version.VersionTitle
	}

	if opts.chooseTranslation {
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
	}

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

		service, err := deps.newTransliterator()
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
	if opts.outputPath != "" {
		return os.WriteFile(opts.outputPath, []byte(content), 0o644)
	}

	_, err = io.WriteString(deps.stdout, content)
	return err
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

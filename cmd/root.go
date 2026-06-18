package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/shekelator/nechama/internal/sefaria"
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
	outputPath        string
}

type commandDependencies struct {
	service textService
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	isTTY   func() bool
}

func Execute() error {
	return newRootCommand(defaultDependencies()).Execute()
}

func defaultDependencies() commandDependencies {
	return commandDependencies{
		service: sefaria.NewClient(
			sefaria.WithBaseURL(os.Getenv("NECHAMA_SEFARIA_BASE_URL")),
			sefaria.WithUserAgent(fmt.Sprintf("nechama/%s", Version)),
		),
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		isTTY: func() bool {
			return isTerminal(os.Stdin) && isTerminal(os.Stdout)
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
		Example:       "  nechama \"Genesis 1:1\"\n  nechama --english \"Genesis 1:1\"\n  nechama fetch --translation \"Revised JPS, 2023\" \"Genesis 1\"",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return runFetch(cmd.Context(), deps, opts, args[0])
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
		Example:       "  nechama fetch \"Berakhot 2a:1\"\n  nechama fetch --english --choose-translation \"Genesis 1:1\"\n  nechama fetch -o genesis.txt \"Genesis 1\"",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(cmd.Context(), deps, opts, args[0])
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
	flags.StringVarP(&opts.outputPath, "output", "o", "", "Write the fetched text to a file instead of stdout")
}

func runFetch(ctx context.Context, deps commandDependencies, opts fetchOptions, ref string) error {
	if opts.translation != "" && opts.chooseTranslation {
		return errors.New("--translation and --choose-translation cannot be used together")
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

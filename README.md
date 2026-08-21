# nechama

`nechama` is a small Go CLI for fetching Jewish texts from [Sefaria](https://www.sefaria.org/) as plain text.

It defaults to the source language of the work, which means Hebrew for Tanakh and the original language for other texts where Sefaria marks a source version. It can also fetch English translations, optionally letting you choose among available versions.

## Features

- fetches text by Sefaria reference
- prints plain text to stdout or writes to a file
- defaults to the source-language version
- supports the highest-priority English translation with `--english`
- supports a specific English translation with `--translation`
- supports interactive translation selection with `--choose-translation`
- supports transliteration of source Hebrew/Aramaic text with `--transliteration`
- supports preserving Hebrew cantillation marks with `--preserve-cantillation`
- transliterates arbitrary Hebrew text passed directly (argument or stdin) without a Sefaria lookup
- uses deterministic, network-free tests for the CLI and Sefaria client logic

## Requirements

- Go 1.26+

## Build

```bash
go build -o nechama .
```

Or run it directly:

```bash
go run . "Genesis 1:1"
```

## Install from GitHub Releases

Prebuilt binaries are published on the [GitHub Releases page](https://github.com/shekelator/nechama/releases).

Each release includes archives for:

- macOS (`amd64`, `arm64`)
- Linux (`amd64`, `arm64`)
- Windows (`amd64`, `arm64`)

Download the archive for your platform, extract it, and place the `nechama` binary in your `PATH`.

macOS binaries are ad-hoc code-signed during the release build, so they run on Apple Silicon (unsigned arm64 binaries are killed by the kernel). Gatekeeper may still prompt about an unidentified developer on first launch — right-click the binary and choose **Open**, or clear the quarantine attribute:

```bash
xattr -d com.apple.quarantine /path/to/nechama
```

## Usage

### Fetch the source-language text

```bash
nechama "Genesis 1:1"
```

You can also pipe the reference through stdin:

```bash
echo "Psalm 51:4" | nechama
echo "Genesis 1:1" | nechama fetch --english
```

### Fetch the highest-priority English translation

```bash
nechama --english "Genesis 1:1"
```

### Fetch a specific English translation

You can use either the short title or the full version title.

```bash
nechama --translation "Revised JPS, 2023" "Genesis 1:1"
nechama fetch --translation "THE JPS TANAKH: Gender-Sensitive Edition" "Genesis 1:1"
```

### Set a default English translation

Set `NECHAMA_DEFAULT_ENGLISH_TRANSLATION` to a short or full version title. When `--english` is passed, nechama resolves that translation by name instead of using Sefaria's highest-priority English translation.

```bash
export NECHAMA_DEFAULT_ENGLISH_TRANSLATION="Revised JPS, 2023"
nechama --english "Genesis 1:1"
```

The env var only takes effect with `--english`. It does not change plain `nechama "Genesis 1:1"`, which still fetches the source text. `--translation` and `--choose-translation` always override it.

If the configured translation is not available for the requested ref (for example, the ref has no matching English version), nechama falls back to the highest-priority English translation and prints a note to stderr.

### Choose a translation interactively

```bash
nechama fetch --choose-translation "Genesis 1:1"
```

### Write output to a file

```bash
nechama -o genesis.txt "Genesis 1"
```

### Text details go to stderr

For every fetch, nechama prints a short details line to stderr — for example `Genesis 1:1 (source)` or `Genesis 1:1 (English, Revised JPS, 2023)` — so you can pipe the text itself to another tool without the metadata mixing in:

```bash
nechama --english "Genesis 1:1" | pbcopy
```

stdout gets just the text; stderr gets the details line (and any fallback notes).

### Transliterate source Hebrew/Aramaic text

Make sure an Ollama model is running locally first:
```bash
ollama run gemma4:e4b
```

```bash
nechama --transliteration "Psalm 132"
```

`--transliteration` only works with source-language fetches. It cannot be combined with `--english`, `--translation`, or `--choose-translation`.

### Preserve cantillation marks in source text

By default, source Hebrew output strips cantillation marks (te'amim), while preserving niqqud and sof pasuq/meteg.

Use `--preserve-cantillation` to keep all cantillation marks in the output:

```bash
nechama --preserve-cantillation "Genesis 1:1"
```

### Transliterate arbitrary Hebrew text directly

If the input contains Hebrew script, `nechama` skips the Sefaria lookup and transliterates the text directly. No flag is needed — pass the text as an argument or pipe it via stdin:

```bash
nechama "שָׁלוֹם עָלֵיכֶם"
echo "בְּרֵאשִׁית" | nechama
```

This uses the same transliteration provider/settings as `--transliteration` (see [Transliteration configuration](#transliteration-configuration)). It cannot be combined with `--english`, `--translation`, or `--choose-translation`. Non-Hebrew input is still treated as a Sefaria reference.

## Command reference

```text
nechama [flags] [ref|text]
nechama fetch [flags] [ref|text]
nechama version
```

### Flags

| Flag | Meaning |
| --- | --- |
| `-e`, `--english` | Fetch the highest-priority English translation |
| `-t`, `--translation <name>` | Fetch a specific English translation by short or full title |
| `--choose-translation` | Prompt for an English translation in an interactive terminal |
| `--transliteration` | Transliterate source Hebrew/Aramaic text into Latin letters |
| `-c`, `--preserve-cantillation` | Keep Hebrew cantillation marks in source text output |
| `-o`, `--output <path>` | Write the fetched text to a file instead of stdout |

## How text selection works

### Default behavior

By default, `nechama` asks Sefaria for the `source` version of the requested ref. That follows Sefaria's own notion of the source text, which is usually Hebrew for Tanakh and the original/default language for other works.

For source Hebrew, `nechama` strips cantillation marks by default for cleaner plain-text output. Add `--preserve-cantillation` to keep them.

### English behavior

- `--english` fetches Sefaria's highest-priority English translation, or the translation named by `NECHAMA_DEFAULT_ENGLISH_TRANSLATION` if set and available for the ref.
- `--translation` resolves a specific English version title before fetching the text.
- `--choose-translation` lists the English versions available for that ref and prompts you to choose one.

### Transliteration behavior

- `--transliteration` transliterates source Hebrew/Aramaic text and outputs transliteration only.
- Input containing Hebrew script is transliterated directly without a Sefaria lookup.
- Transliteration errors fail the command.
- The transliteration rules are built into source at `internal/transliteration/rules.go`.

## Transliteration configuration

Transliteration provider configuration is read from JSON:

- `NECHAMA_CONFIG` if set
- otherwise `$XDG_CONFIG_HOME/nechama/config.json` (or the OS-equivalent user config directory)

Example config:

```json
{
	"transliteration": {
		"provider": "ollama",
		"ollama": {
			"base_url": "http://host.docker.internal:11434",
			"model": "gemma4:e4b",
			"api_key": "dummy-api-key",
			"timeout_seconds": 60
		}
	}
}
```

Environment variable overrides (take precedence over config file):

- `NECHAMA_TRANSLITERATION_PROVIDER`
- `NECHAMA_TRANSLITERATION_BASE_URL`
- `NECHAMA_TRANSLITERATION_MODEL`
- `NECHAMA_TRANSLITERATION_API_KEY`
- `NECHAMA_TRANSLITERATION_TIMEOUT_SECONDS`

### Defaults and precedence

- The app loads defaults from code first.
- Then it reads config from `NECHAMA_CONFIG` (if set) or the default user config path.
- Then environment variables override config values.

Current in-code defaults are:

- provider: `ollama`
- base_url: `http://host.docker.internal:11434`
- model: `gemma4:cloud`
- api_key: `dummy-api-key`
- timeout_seconds: `60`

The Ollama HTTP client sends `Authorization: Bearer <api_key>` when `api_key` is non-empty. This supports Ollama Cloud and other API-key-based gateways.

### Devcontainer note

In a devcontainer, `127.0.0.1` points to the container itself, not your host machine. If Ollama is running on your host, set `base_url` to `http://host.docker.internal:11434` (or set `NECHAMA_TRANSLITERATION_BASE_URL` to that value).

### Environment-only example

```bash
# NECHAMA_TRANSLITERATION_BASE_URL should be http://localhost:11434 if running outside of docker
NECHAMA_TRANSLITERATION_PROVIDER=ollama \
NECHAMA_TRANSLITERATION_BASE_URL=http://host.docker.internal:11434 \
NECHAMA_TRANSLITERATION_MODEL=gemma4:e4b \
NECHAMA_TRANSLITERATION_API_KEY=dummy-api-key \
NECHAMA_TRANSLITERATION_TIMEOUT_SECONDS=60 \
go run . "Psalm 27" --transliteration
```

Currently implemented provider:

- `ollama`

The transliteration module already uses a provider factory pattern in `internal/transliteration/factory.go`, so additional providers can be added without changing command flow.

## Development overview

The codebase is intentionally small:

- `cmd/` contains the Cobra CLI commands
- `internal/sefaria/` contains the Sefaria API client and text-formatting logic
- `main.go` wires the CLI entrypoint

The CLI uses Sefaria's v3 texts API with `return_format=text_only`, so output is plain text rather than HTML-rich markup.

## Testing

Run the full test suite with:

```bash
go test ./...
```

The tests cover:

- CLI command behavior
- translation selection
- text flattening for strings, sections, and nested commentary responses
- Sefaria request/query construction
- error handling for unresolved refs and missing translations

## Releasing

Releases are automated with GitHub Actions + GoReleaser.

1. Create and push a semantic version tag from `main`:

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. The `Release` workflow runs tests, builds cross-platform binaries, and publishes a GitHub Release with archives plus `checksums.txt`.

For local verification of the release config without publishing:

```bash
go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish
```

## Sefaria references

- API getting started: <https://developers.sefaria.org/reference/getting-started>
- v3 texts API: <https://developers.sefaria.org/reference/get-v3-texts>
- Sefaria MCP docs: <https://developers.sefaria.org/docs/the-sefaria-mcp>

## TODOs
- Incorporate [tanach.us](https://tanach.us) for better text reliability
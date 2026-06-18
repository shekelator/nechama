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
- uses deterministic, network-free tests for the CLI and Sefaria client logic

## Requirements

- Go 1.26+

## Build

```bash
go build ./...
```

Or run it directly:

```bash
go run . "Genesis 1:1"
```

## Usage

### Fetch the source-language text

```bash
nechama "Genesis 1:1"
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

### Choose a translation interactively

```bash
nechama fetch --choose-translation "Genesis 1:1"
```

### Write output to a file

```bash
nechama -o genesis.txt "Genesis 1"
```

## Command reference

```text
nechama [flags] <ref>
nechama fetch [flags] <ref>
nechama version
```

### Flags

| Flag | Meaning |
| --- | --- |
| `-e`, `--english` | Fetch the highest-priority English translation |
| `-t`, `--translation <name>` | Fetch a specific English translation by short or full title |
| `--choose-translation` | Prompt for an English translation in an interactive terminal |
| `-o`, `--output <path>` | Write the fetched text to a file instead of stdout |

## How text selection works

### Default behavior

By default, `nechama` asks Sefaria for the `source` version of the requested ref. That follows Sefaria's own notion of the source text, which is usually Hebrew for Tanakh and the original/default language for other works.

### English behavior

- `--english` fetches Sefaria's highest-priority English translation.
- `--translation` resolves a specific English version title before fetching the text.
- `--choose-translation` lists the English versions available for that ref and prompts you to choose one.

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

## Sefaria references

- API getting started: <https://developers.sefaria.org/reference/getting-started>
- v3 texts API: <https://developers.sefaria.org/reference/get-v3-texts>
- Sefaria MCP docs: <https://developers.sefaria.org/docs/the-sefaria-mcp>

# AGENTS.md

## Project purpose

This repo provides a maintainable Go CLI for fetching plain-text passages from Sefaria.

## Architecture

- `main.go`: process entrypoint
- `cmd/`: Cobra command tree and CLI UX
- `internal/sefaria/`: Sefaria HTTP client, translation lookup, and text flattening

Keep the CLI surface simple and script-friendly. Default behavior should stay non-interactive; interactive prompts should only be used for explicit user choice flows such as `--choose-translation`.

## Sefaria API conventions

- Primary text endpoint: `GET https://www.sefaria.org/api/v3/texts/{ref}`
- Default output format in this repo: `return_format=text_only`
- Source-language fetch: `version=source`
- Highest-priority English fetch: `version=english`
- Specific English fetch: `version=english|<full version title>`
- English version listing for a ref: `version=english|all`

When changing the client:

1. Prefer Sefaria v3 responses over older endpoint shapes.
2. Treat `actualLanguage` and `languageFamilyName` as more trustworthy than legacy `language`.
3. Preserve readable line breaks when flattening section or commentary text.
4. Keep HTTP behavior explicit and testable with injected clients or `httptest`.

## Testing expectations

- Prefer deterministic tests with fixtures or `httptest`.
- Do not add default tests that depend on live network access.
- Cover both CLI behavior and client behavior when changing fetching logic.

## MCP guidance

This repo intentionally keeps MCP support as documentation rather than committed client-specific config.

Relevant Sefaria MCP endpoints:

- Texts MCP: `https://mcp.sefaria.org/sse`
- Developers MCP: `https://developers.sefaria.org/mcp`

Relevant docs:

- <https://developers.sefaria.org/docs/the-sefaria-mcp>
- <https://github.com/Sefaria/sefaria-mcp>

Use those MCPs for future agent-assisted exploration of texts, versions, and API behavior, but keep repo changes portable and tool-agnostic unless a future task explicitly asks for client-specific config.

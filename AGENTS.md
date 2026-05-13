# AGENTS.md/CLAUDE.md

This file provides guidance to autonomous AI agents when working with code in
this repository.

## Project

trello-mcp is a Go MCP (Model Context Protocol) server that exposes Trello
board/card/list operations as tools for LLM agents. It communicates over stdio
using the MCP protocol. See SPEC.md for the full API specification.

## Build and Test

First-time setup installs project-local Go tools into `./bin` (golangci-lint,
goimports, gocyclo):

```
make setup
```

Build the binary to `./bin/trello-mcp`:

```
make build
```

Run tests (`-race` is on by default; set `RACE=0` to disable):

```
make test
```

Optional coverage report: `make test:cover`. See `make help` for other targets.

The binary is `trello-mcp`. It has two modes: `trello-mcp auth` (credential
validation) and the default MCP server mode (stdio).

## Verification

Before ANY change is considered done, run all five targets. All must
pass. No exceptions.

```
make fmt
make build
make test
make lint
make cyclo
```

## Code Composition

Top-level functions should scan as a DSL of named steps. They express WHAT
happens as a sequence of well-named calls. The HOW lives in the helpers.

When refactoring a function with high cyclomatic complexity, apply this
principle: extract blocks that have their own distinct purpose into named
helpers, so the parent reads as a series of named steps.

**Extract when** a block has a distinct purpose nameable in 2-3 words, operates
at a different abstraction level than its surroundings, and replacing 10-30
inline lines with a named call makes the parent more readable.

**Do NOT extract when** the code is purely sequential setup with no branching,
the helper would need 4+ parameters (the extraction boundary is wrong), the
name would just restate the code in camelCase, or the code is declarative data
(tool definitions, config structs).

**Naming:** Helpers describe what they yield or do -- `drainChat`,
`writeEntityBuckets`, `shortEntityList`. Never name by where it's called --
`thinkStep1`, `handlePart2`.

**Placement:** Helpers stay in the same file as their caller unless they serve
multiple files in the package. Never create a file just for one small helper.

**Anti-patterns:** Helper soup (extracting every 5-line block obscures a
readable 40-line flow). State-smuggling parameters (5+ params means the
extraction boundary is wrong). Naming after the caller. Extracting trivial
`if err != nil` handling.

**After refactoring:** Read the top-level function aloud -- does it scan as
named steps? Run `make cyclo` -- complexity must hold steady or improve.

## Architecture

```
cmd/trello-mcp/   # Main entry point, CLI parsing (auth subcommand)
internal/
  server/
    tools.go      # MCP tool definitions (names, parameters, descriptions)
    handler/
      handler.go  # Shared logic: board ID resolution, allowed_boards checks
      apierr.go   # mapAPIError: HTTP status -> tool error mapping
      <tool>.go   # One file per tool handler
  trello/
    client.go     # Thin HTTP client over net/http, appends key+token
    types.go      # Typed structs for Trello API responses
```

Config lives at `~/.config/trello-mcp/config.json` (`api_key`, `token`, optional
`default_board`, optional `allowed_boards`).

## Conventions

**ASCII only.** Use only characters typeable on a US English keyboard
in Markdown files and Go/TypeScript source comments. No Unicode arrows
(use `->`), no em-dashes (use `--` or rephrase), no multiplication signs
(use `x`), no special math symbols (use `>=` not the glyph). TUI box-drawing
characters for terminal rendering are exempt.

**Markdown line wrapping.** Wrap regular prose in Markdown files at 80
characters. Exceptions: code blocks, inline code spans, CLI usage
examples, Markdown tables, URLs, and cases where wrapping would reduce
readability.

**Logging.** `internal/logging/` follows the mab-go pattern. DO NOT MODIFY this
package.

## Key design decisions

- **No third-party Trello client.** Thin wrapper in `internal/trello/` over
  `net/http`. Every request gets `key` and `token` query params appended
  automatically.
- **Two-tier error model.** Tool errors (`*mcp.CallToolResult, nil`) for
  user-recoverable problems (bad input, 400/401/403/404/429). Protocol errors
  (`nil, error`) for unexpected failures. Error messages must suggest the fix.
- **Board ID resolution** is shared logic in `handler.go`: explicit arg ->
  `default_board` -> error. `trello_search` has a soft fallback (no error when
  both are empty).
- **`allowed_boards` enforcement** is cross-cutting: all tools accepting
  `card_id` must check the card's `idBoard` against the list. `trello_move_card`
  also checks the target board.
- **List name resolution** is case-insensitive exact match against the board's
  lists. On zero matches, the tool error lists available names.
- **Label operations** are add/remove only (not replace). Label names are
  resolved case-insensitively against board labels; unrecognized names produce
  a tool error listing available labels.
- **All responses are JSON objects.** No custom text formats.
- **Dates**: due dates -> `YYYY-MM-DD`, activity timestamps ->
  `YYYY-MM-DD HH:MM` (UTC).

# AGENTS.md/CLAUDE.md

This file provides guidance to autonomous AI agents when working with code in
this repository.

## Project

trello-mcp is a Go MCP (Model Context Protocol) server that exposes Trello
board/card/list operations as tools for LLM agents. It communicates over stdio
using the MCP protocol. See DESIGN.md for the full API specification.

## Build and Test

(TODO: Add relevant `make` targets here)

The binary is `trello-mcp`. It has two modes: `trello-mcp auth` (credential
validation) and the default MCP server mode (stdio).

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

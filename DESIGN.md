# DESIGN: trello-mcp

API specification for [trello-mcp](https://github.com/mab-go/trello-mcp). Tool
definitions live in `internal/server/tools.go`; behavior is implemented under
`internal/server/handler/`. For setup instructions, see [SETUP.md](SETUP.md).

---

## Configuration

Config file: `~/.config/trello-mcp/config.json` (see [SETUP.md](SETUP.md) for
creation).

| Field            | Required | Role                                                      |
|------------------|----------|-----------------------------------------------------------|
| `api_key`        | Yes      | Trello API key (from Power-Up admin page)                 |
| `token`          | Yes      | Trello API token (generated alongside the API key)        |
| `default_board`  | No       | Used when a tool omits `board_id` (see below)             |
| `allowed_boards` | No       | When non-empty, restricts which board IDs may be accessed |

No separate token file; Trello's API key and token are static credentials stored
directly in `config.json`. There is no token refresh flow.

---

## Authentication

Trello uses API key + token authentication. Every HTTP request to the Trello API
includes `key` and `token` as query parameters. There is no OAuth2 flow, no
callback server, and no token expiry/refresh cycle.

### `auth` command

The `auth` subcommand validates the configured credentials:

```
trello-mcp auth           # Validate key+token against GET /1/members/me
trello-mcp auth --status  # Check config file state (key+token present, non-empty)
```

`auth` (no flags) makes a real API call to `GET /1/members/me?key=...&token=...`
and reports the authenticated member's full name and username, or the error.
This is the only network call the `auth` command makes.

`auth --status` only checks that `config.json` exists and contains non-empty
`api_key` and `token` fields. It does not call the Trello API.

There is no `--revoke` flag. Trello tokens can be revoked manually via
`https://trello.com/1/tokens/{token}?key={key}` (DELETE) or through the Trello
UI under account settings. The SETUP.md documents this.

---

## Board ID resolution

Tools that accept `board_id` use shared resolution logic
(`internal/server/handler/handler.go`):

1. Use the `board_id` argument if non-empty.
2. Else use `default_board` from config if set.
3. If still empty, return a tool error: missing `board_id` (no default
   configured).
4. If `allowed_boards` is non-empty, the resolved ID must appear in that list;
   otherwise a tool error is returned.

`trello_boards` does **not** take `board_id`. `trello_search` accepts an
optional `board_id` that falls back to `default_board` if set; when neither is
provided, the search spans all boards (no error). If `allowed_boards` is set,
both tools **filter** results to only boards in the list.

---

## Trello HTTP client (`internal/trello/`)

A thin wrapper over `net/http`. No third-party Trello client library.

### Request construction

All requests go to `https://api.trello.com/1/...`. The client appends `key` and
`token` query parameters to every request automatically.

### Timeouts

All HTTP requests use a 30-second timeout. This is set on the `http.Client`,
not per-request.

### Rate limiting

Trello enforces two rate limits (per the Trello API docs):

- **Per API key**: 300 requests per 10 seconds
- **Per token**: 100 requests per 10 seconds

The client reads rate limit response headers
(`x-rate-limit-api-token-remaining`, etc.) for logging but does **not**
implement client-side throttling in v1. If a 429 is received, it is surfaced as
a tool error with a message to wait and retry.

### Error mapping

HTTP status codes map to tool errors via `mapAPIError`
(`internal/server/handler/apierr.go`):

| HTTP status | Tool error message                                                                          |
|-------------|---------------------------------------------------------------------------------------------|
| 400         | Include Trello's error message: "Trello rejected the request (400): {message}"              |
| 401         | "Authentication failed. Verify your API key and token in ~/.config/trello-mcp/config.json." |
| 403         | "Access denied (403). Verify you have access to this board/card."                           |
| 404         | "Resource not found (404). Verify the board/card/list ID."                                  |
| 429         | "Trello API rate limit exceeded (429). Wait a few seconds and retry."                       |

Status codes outside this set are protocol errors (returned as `(nil, err)`).

### JSON handling

All Trello API responses are JSON. The client deserializes into typed structs
defined in `internal/trello/types.go`. The `fields` query parameter is used on
GET requests to limit response payloads to only the fields the handler needs.

---

## Response format

All tool responses are JSON objects. Tool inputs are also JSON (MCP default).

Response shapes:

- **List tools** (`trello_boards`, `trello_lists`, `trello_cards`,
  `trello_checklists`, `trello_labels`): metadata fields plus an array of items.
- **Detail tools** (`trello_get_card`, `trello_board_summary`): flat metadata
  fields with nested arrays for sub-resources (checklists, comments, etc.).
- **Mutation tools** (`trello_create_card`, `trello_update_card`,
  `trello_archive_card`, `trello_unarchive_card`, `trello_add_comment`,
  `trello_check_item`, `trello_add_checklist`, `trello_add_check_item`,
  `trello_add_label`, `trello_remove_label`, `trello_add_attachment`,
  `trello_create_list`, `trello_move_card`): confirmation object with the
  created/updated resource's key fields.
- **Search** (`trello_search`): metadata plus separate `cards` and `boards`
  arrays.

---

## Tool reference

All tool names are prefixed with `trello_`. Parameters below match
`internal/server/tools.go`.

---

### Tier 1 -- Core CRUD

---

### `trello_boards`

List boards for the authenticated member, optionally filtered by name.

| Parameter | Type   | Required | Notes                                                                                           |
|-----------|--------|----------|-------------------------------------------------------------------------------------------------|
| `query`   | string | No       | Filter boards whose name contains this string (case-insensitive). Omit to list all open boards. |

**Behavior:** Calls `GET /1/members/me/boards` with `filter=open` and
`fields=name,url,shortUrl,dateLastActivity`. If `query` is provided, filters
results client-side by case-insensitive substring match on the board name.
Results respect `allowed_boards` filtering when that list is non-empty.

**Response:** `count`, `boards` array (each with: `board_id`, `name`, `url`,
optional `last_activity` as `YYYY-MM-DD`).

---

### `trello_lists`

Get all open lists on a board.

| Parameter  | Type   | Required | Notes                                          |
|------------|--------|----------|------------------------------------------------|
| `board_id` | string | No*      | *Required unless `default_board` is configured |

**Behavior:** Calls `GET /1/boards/{id}/lists` with `filter=open` and
`fields=name,pos`.

**Response:** `board_id`, `count`, `lists` array (each with: `list_id`, `name`,
`position`).

---

### `trello_cards`

Get cards on a board or in a specific list, with optional filtering.

| Parameter  | Type   | Required | Notes                                                     |
|------------|--------|----------|-----------------------------------------------------------|
| `board_id` | string | No*      | *Required unless `default_board` is configured            |
| `list_id`  | string | No       | Scope to a specific list; omit for all cards on the board |
| `filter`   | string | No       | `open` (default), `closed`, `all`                         |
| `due`      | string | No       | Filter by due date: `overdue`, `day`, `week`, `month`     |
| `label`    | string | No       | Filter by label name (case-insensitive substring match)   |
| `limit`    | number | No       | Max cards to return; default 100, max 1000                |

**Behavior:** When `list_id` is provided, calls
`GET /1/lists/{listId}/cards`. Otherwise, calls
`GET /1/boards/{boardId}/cards`. Always requests
`fields=name,due,dueComplete,idList,labels,shortUrl,dateLastActivity,closed`.
Also requests `members=true&member_fields=fullName,username` to include assigned
members.

The `due` filter is applied client-side after fetching:

- `overdue`: due date is in the past and `dueComplete` is false
- `day`: due within the next 24 hours (inclusive of overdue)
- `week`: due within the next 7 days (inclusive of overdue)
- `month`: due within the next 30 days (inclusive of overdue)

The `label` filter is applied client-side by case-insensitive substring match on
label names.

**Response:** `board_id`, optional `list_id`, `filter`, `count`, `cards` array
(each with: `card_id`, `name`, `list_name` (resolved from list data), `due`
(ISO 8601 date or empty), `due_complete`, `labels` (array of label name
strings), `members` (array of username strings), `url`, optional
`last_activity`).

**List name resolution:** To include human-readable list names in card results,
the handler fetches the board's lists (cached for the duration of the request)
and maps `idList` → list name.

---

### `trello_get_card`

Get full detail for a single card.

| Parameter | Type   | Required | Notes                |
|-----------|--------|----------|----------------------|
| `card_id` | string | Yes      | Card ID or shortLink |

**Behavior:** Calls `GET /1/cards/{id}` with
`checklists=all&checklist_fields=name&members=true&member_fields=fullName,username&attachments=true&attachment_fields=name,url,date`
and `actions=commentCard&actions_limit=10&action_fields=data,date,memberCreator&action_memberCreator_fields=fullName,username`
to get checklists, members, attachments, and recent comments in a single call.

Does **not** use board ID resolution -- `card_id` is globally unique in Trello.
However, if `allowed_boards` is non-empty, the handler checks that the card's
`idBoard` is in the allowed list before returning results.

**Response:**

```json
{
  "card_id": "abc123",
  "name": "Q3 Website Redesign",
  "list": "In Progress",
  "board": "Acme Project",
  "url": "https://trello.com/c/...",
  "description": "Redesign the public-facing marketing site...",
  "due": "2026-06-15",
  "due_complete": false,
  "labels": ["Urgent", "Blocked"],
  "members": ["jsmith"],
  "checklists": [
    {"checklist_name": "...", "items": [{"item_name": "...", "complete": false}]}
  ],
  "comments": [
    {"author": "...", "text": "...", "date": "2026-05-10 14:30"}
  ],
  "attachments": [
    {"name": "...", "url": "https://...", "date": "2026-05-08 09:15"}
  ]
}
```

Checklists, comments (most recent 10), and attachments are included as nested
arrays. Empty arrays are omitted entirely.

---

### `trello_create_card`

Create a new card on a list.

| Parameter     | Type            | Required | Notes                                                                                                                       |
|---------------|-----------------|----------|-----------------------------------------------------------------------------------------------------------------------------|
| `board_id`    | string          | No*      | *Required unless `default_board` is configured. Used to validate `list_id` belongs to the board and to resolve `list_name`. |
| `list_id`     | string          | See note | Required unless `list_name` is provided                                                                                     |
| `list_name`   | string          | See note | Alternative to `list_id`; resolved to an ID via the board's lists. Case-insensitive exact match.                            |
| `name`        | string          | Yes      | Card title                                                                                                                  |
| `description` | string          | No       | Card description (Markdown supported by Trello)                                                                             |
| `due`         | string          | No       | Due date (ISO 8601 date or datetime string)                                                                                 |
| `labels`      | array of string | No       | Label names to apply (matched case-insensitively against existing board labels)                                             |
| `position`    | string          | No       | `top` or `bottom` (default: `bottom`)                                                                                       |

**List resolution:** Exactly one of `list_id` or `list_name` must be provided.
If `list_name` is given, the handler fetches the board's lists and matches by
case-insensitive exact comparison. If zero matches, the tool error lists
available list names. If multiple matches (shouldn't happen, but defensive),
the first match is used.

**Label resolution:** Label names are matched case-insensitively against
existing labels on the board (`GET /1/boards/{id}/labels`). Unrecognized label
names are reported as a tool error listing available labels.

**Behavior:** Calls `POST /1/cards` with the resolved parameters.

**Response:** `card_id`, `name`, `list` (resolved list name), `url`.

---

### `trello_update_card`

Update one or more fields on an existing card.

| Parameter      | Type            | Required | Notes                                                                                                        |
|----------------|-----------------|----------|--------------------------------------------------------------------------------------------------------------|
| `card_id`      | string          | Yes      | Card ID or shortLink                                                                                         |
| `name`         | string          | No       | New card title                                                                                               |
| `description`  | string          | No       | New card description                                                                                         |
| `due`          | string          | No       | New due date (ISO 8601), or empty string to remove                                                           |
| `due_complete` | boolean         | No       | Mark due date as complete or incomplete                                                                      |
| `list_id`      | string          | No       | Move card to this list                                                                                       |
| `list_name`    | string          | No       | Move card to this list (resolved by name; requires `board_id` or default board)                              |
| `board_id`     | string          | No       | Required only when using `list_name` for list resolution                                                     |
| `position`     | string          | No       | `top` or `bottom` within the current or target list                                                          |

At least one field besides `card_id` must be provided; otherwise return a tool
error.

**Moving cards:** If `list_id` or `list_name` is provided, the card is moved.
`list_name` requires board context (from `board_id` argument, card's current
board, or `default_board`).

**Behavior:** Calls `PUT /1/cards/{id}` with only the fields that were provided.

**Response:** `card_id`, `name`, `list` (current list name after update),
`url`, plus each updated field echoed back for confirmation.

---

### `trello_archive_card`

Archive (close) a card.

| Parameter | Type   | Required | Notes                |
|-----------|--------|----------|----------------------|
| `card_id` | string | Yes      | Card ID or shortLink |

**Behavior:** Calls `PUT /1/cards/{id}` with `closed=true`.

**Response:** `card_id`, `name`, `archived` (always `true`).

---

### `trello_unarchive_card`

Unarchive (reopen) a card.

| Parameter | Type   | Required | Notes                |
|-----------|--------|----------|----------------------|
| `card_id` | string | Yes      | Card ID or shortLink |

**Behavior:** Calls `PUT /1/cards/{id}` with `closed=false`.

**Response:** `card_id`, `name`, `archived` (always `false`).

---

### `trello_add_comment`

Add a comment to a card.

| Parameter | Type   | Required | Notes                |
|-----------|--------|----------|----------------------|
| `card_id` | string | Yes      | Card ID or shortLink |
| `text`    | string | Yes      | Comment text         |

**Behavior:** Calls `POST /1/cards/{id}/actions/comments` with `text`.

**Response:** `card_id`, `comment_id`, `text` (echoed back, truncated to
200 chars in the response for confirmation).

---

### `trello_search`

Search for cards and boards by keyword.

| Parameter  | Type   | Required | Notes                                                       |
|------------|--------|----------|-------------------------------------------------------------|
| `query`    | string | Yes      | Search query (Trello search syntax supported)               |
| `board_id` | string | No       | Scope to a board. Falls back to `default_board`; omit both to search all |
| `limit`    | number | No       | Max results per type; default 10, max 20                    |

**Behavior:** Resolves `board_id` using the standard fallback (explicit →
`default_board`); if neither is set, searches all boards. Calls `GET /1/search`
with `query`, `modelTypes=cards,boards`, and optionally `idBoards` for scoping.
Requests `card_fields=name,idBoard,idList,due,labels,shortUrl,closed`,
`card_board=true&card_board_fields=name`,
`card_list=true&card_list_fields=name`, and `board_fields=name,url,closed`.
Board and list names for each card are extracted from the inlined board/list
objects.

Results respect `allowed_boards` filtering. Cards on disallowed boards are
excluded; disallowed boards are excluded from the boards list.

**Response:** `query`, `card_count`, `board_count`, `cards` array (each with:
`card_id`, `name`, `board_name`, `list_name`, `due`, `labels`, `url`,
`closed`), `boards` array (each with: `board_id`, `name`, `url`, `closed`).

---

### Tier 2 -- Checklists & Labels

---

### `trello_checklists`

Get all checklists on a card with their items.

| Parameter | Type   | Required | Notes                |
|-----------|--------|----------|----------------------|
| `card_id` | string | Yes      | Card ID or shortLink |

**Behavior:** Calls `GET /1/cards/{id}/checklists` with
`checkItem_fields=name,state,pos`.

**Response:** `card_id`, `count`, `checklists` array (each with: `checklist_id`,
`name`, `items` array of objects with `item_id`, `name`, `complete`).

---

### `trello_check_item`

Check or uncheck a checklist item.

| Parameter  | Type    | Required | Notes                               |
|------------|---------|----------|-------------------------------------|
| `card_id`  | string  | Yes      | Card ID or shortLink                |
| `item_id`  | string  | Yes      | Checklist item ID                   |
| `complete` | boolean | Yes      | `true` to check, `false` to uncheck |

**Behavior:** Calls
`PUT /1/cards/{cardId}/checkItem/{itemId}` with
`state=complete` or `state=incomplete`.

**Response:** `card_id`, `item_id`, `name`, `complete`.

---

### `trello_add_checklist`

Create a new checklist on a card.

| Parameter | Type            | Required | Notes                             |
|-----------|-----------------|----------|-----------------------------------|
| `card_id` | string          | Yes      | Card ID or shortLink              |
| `name`    | string          | Yes      | Checklist name                    |
| `items`   | array of string | No       | Initial checklist items to create |

**Behavior:** Calls `POST /1/cards/{id}/checklists` with `name`. If `items` is
provided, iterates and calls `POST /1/checklists/{id}/checkItems` for each item
(Trello's API does not support bulk checklist item creation).

**Response:** `card_id`, `checklist_id`, `name`, `item_count`.

---

### `trello_add_check_item`

Add an item to an existing checklist.

| Parameter      | Type    | Required | Notes                                      |
|----------------|---------|----------|--------------------------------------------|
| `checklist_id` | string  | Yes      | Checklist ID                               |
| `name`         | string  | Yes      | Item text                                  |
| `checked`      | boolean | No       | Create as already checked; default `false` |

**Behavior:** Calls `POST /1/checklists/{id}/checkItems` with `name` and
optionally `checked`.

**Response:** `checklist_id`, `item_id`, `name`, `complete`.

---

### `trello_labels`

Get all labels on a board.

| Parameter  | Type   | Required | Notes                                          |
|------------|--------|----------|------------------------------------------------|
| `board_id` | string | No*      | *Required unless `default_board` is configured |

**Behavior:** Calls `GET /1/boards/{id}/labels` with `fields=name,color`.
Filters out labels with empty names (Trello creates unnamed placeholder labels
for each color).

**Response:** `board_id`, `count`, `labels` array (each with: `label_id`, `name`,
`color`).

---

### `trello_add_label`

Add an existing label to a card.

| Parameter | Type   | Required | Notes                                                        |
|-----------|--------|----------|--------------------------------------------------------------|
| `card_id` | string | Yes      | Card ID or shortLink                                         |
| `label`   | string | Yes      | Label name (matched case-insensitively against board labels) |

**Behavior:** Fetches the card to determine its board, then resolves the label
name against the board's labels (`GET /1/boards/{id}/labels`). If no match is
found, the tool error lists available label names. Calls
`POST /1/cards/{id}/idLabels` with the resolved label ID.

**Response:** `card_id`, `label_id`, `name`, `color`.

---

### `trello_remove_label`

Remove a label from a card.

| Parameter | Type   | Required | Notes                                                         |
|-----------|--------|----------|---------------------------------------------------------------|
| `card_id` | string | Yes      | Card ID or shortLink                                          |
| `label`   | string | Yes      | Label name (matched case-insensitively against card's labels) |

**Behavior:** Fetches the card (including its current labels). Matches the label
name case-insensitively against the card's labels. If the label is not on the
card, returns a tool error listing the card's current labels. Calls
`DELETE /1/cards/{id}/idLabels/{idLabel}`.

**Response:** `card_id`, `label_id`, `name`, `removed` (always `true`).

---

### Tier 3 -- Workflow
 
---
 
### `trello_add_attachment`
 
Attach a URL to a card.
 
| Parameter | Type   | Required | Notes                                                      |
|-----------|--------|----------|------------------------------------------------------------|
| `card_id` | string | Yes      | Card ID or shortLink                                       |
| `url`     | string | Yes      | URL to attach (must be a valid HTTP/HTTPS URL)             |
| `name`    | string | No       | Display name for the attachment; Trello auto-generates one from the URL if omitted |
 
**Behavior:** Calls `POST /1/cards/{id}/attachments` with `url` and optionally
`name`. Only URL attachments are supported — file uploads are a non-goal (see
[Explicit non-goals](#explicit-non-goals)).
 
The handler validates that `url` begins with `http://` or `https://` before
calling the API. Other schemes are rejected with a tool error.
 
**Response:** `card_id`, `attachment_id`, `name`, `url`.
 
---
 
### `trello_create_list`
 
Create a new list on a board.
 
| Parameter  | Type   | Required | Notes                                          |
|------------|--------|----------|-------------------------------------------------|
| `board_id` | string | No*      | *Required unless `default_board` is configured  |
| `name`     | string | Yes      | List name                                       |
| `position` | string | No       | `top` or `bottom` (default: `bottom`)           |
 
**Behavior:** Calls `POST /1/boards/{id}/lists` with `name` and `pos`. The
`position` value is mapped: `top` → `"top"`, `bottom` → `"bottom"` (Trello's
API accepts these string values directly for the `pos` field).
 
**Response:** `board_id`, `list_id`, `name`, `position`.
 
---
 
### `trello_board_summary`
 
Get a high-level status overview of a board: card counts per list, overdue
items, and upcoming due dates.
 
| Parameter  | Type   | Required | Notes                                          |
|------------|--------|----------|-------------------------------------------------|
| `board_id` | string | No*      | *Required unless `default_board` is configured  |
 
**Behavior:** Fetches all open lists (`GET /1/boards/{id}/lists?filter=open`)
and all open cards
(`GET /1/boards/{id}/cards?filter=open&fields=name,idList,due,dueComplete,labels`)
in two API calls. Then computes:
 
- **Per-list card counts**: Number of open cards in each list.
- **Overdue cards**: Cards with a due date in the past where `dueComplete` is
  false.
- **Due soon**: Cards due within the next 7 days (not yet overdue) where
  `dueComplete` is false.
- **Total open cards**: Sum across all lists.

**Response:**

```json
{
  "board_id": "abc123",
  "board_name": "Acme Project",
  "total_cards": 24,
  "overdue_count": 2,
  "due_soon_count": 5,
  "lists": [
    {"list_name": "Backlog", "card_count": 8},
    {"list_name": "In Progress", "card_count": 12}
  ],
  "overdue": [
    {"card_id": "...", "name": "...", "list_name": "...", "due": "2026-05-01", "labels": ["Urgent"]}
  ],
  "due_soon": [
    {"card_id": "...", "name": "...", "list_name": "...", "due": "2026-05-15", "labels": []}
  ]
}
```

Empty `overdue` and `due_soon` arrays are omitted entirely.
 
This is the "morning check-in" tool — a single call gives the agent everything
it needs to answer "What's going on with Acme Project today?"
 
---
 
### `trello_move_card`
 
Move a card to a different board and/or list.
 
| Parameter       | Type   | Required | Notes                                                     |
|-----------------|--------|----------|-----------------------------------------------------------|
| `card_id`       | string | Yes      | Card ID or shortLink                                      |
| `target_board`  | string | No       | Destination board ID. Omit to stay on the same board.     |
| `target_list_id`| string | See note | Destination list ID. Required unless `target_list_name` is provided. |
| `target_list_name`| string | See note | Destination list name (resolved against the target board's lists). Case-insensitive exact match. |
| `position`      | string | No       | `top` or `bottom` within the target list (default: `bottom`) |
 
**Why a separate tool from `trello_update_card`?** Cross-board moves require
both `idBoard` and `idList` to be set atomically, and the list resolution
context changes (you're resolving against the *target* board, not the card's
current board). Keeping this separate avoids overloading `trello_update_card`
with conditional board-resolution logic.
 
**List resolution:** Same pattern as `trello_create_card` — exactly one of
`target_list_id` or `target_list_name` must be provided. Name resolution uses
the target board (explicit or the card's current board if `target_board` is
omitted).
 
**Behavior:** First fetches the card
(`GET /1/cards/{id}?fields=idBoard,idList`) to capture the current board and
list for the response's `from_board`/`from_list` fields. Then calls
`PUT /1/cards/{id}` with `idBoard` (if cross-board), `idList`, and optionally
`pos`. Board and list IDs (both source and target) are resolved to names for the
response.
 
If `allowed_boards` is non-empty, both the card's current board and the target
board must be in the allowed list.
 
**Response:** `card_id`, `name`, `from_board`, `from_list`, `to_board`,
`to_list`, `url`.
 
---

## Cross-cutting rules

### Empty results

Empty result sets (no cards, no checklists, no search results) are **not**
errors. Return a valid JSON response with an empty array and zero count.

### Error handling philosophy

Same two-tier model as `sheets-mcp`:

- **Tool errors** (`*mcp.CallToolResult, nil`): Bad arguments, resolution
  failures, board not allowed, user-recoverable API conditions (400, 401, 403,
  404, 429).
- **Protocol errors** (`nil, error`): Unexpected failures (I/O, unhandled HTTP
  status codes, JSON deserialization failures).

Error messages **must suggest the fix.** Not "board not found" but "Board
'abc123' not found. Use trello_boards to list your boards."

### Auth errors

Auth failures (401 from Trello) are tool errors, not protocol errors:

```go
return mcp.NewToolResultError(
    "authentication failed. Verify your API key and token in " +
    "~/.config/trello-mcp/config.json, or generate new credentials " +
    "at https://trello.com/power-ups/admin.",
), nil
```

### Card ID flexibility

Trello supports both full card IDs and shortLinks (the short alphanumeric string
from the card URL). All tools that accept `card_id` should work with either
form -- the Trello API handles this transparently.

### Date formatting

Trello returns dates in ISO 8601 format (`2026-06-15T12:00:00.000Z`). In
responses:

- Due dates are formatted as `YYYY-MM-DD` (date only, no time) unless the time
  component is meaningful (not midnight UTC).
- Activity timestamps are formatted as `YYYY-MM-DD HH:MM` (UTC).

### Allowed boards enforcement

When `allowed_boards` is non-empty:

- `trello_boards` filters results to only boards in the list.
- `trello_search` filters both card and board results.
- All tools accepting `card_id` check the card's `idBoard` against the list
  before proceeding. `trello_move_card` additionally checks the target board.
- `trello_add_check_item` resolves the checklist's board (via
  `GET /1/checklists/{id}/board`) and checks it against the list.
- All other tools that resolve a `board_id` check it against the list.

---

## Explicit non-goals
 
Do not implement the following in v1. If asked, decline and point here.
 
- Board creation or deletion
- List archiving or reordering
- Label creation or deletion (labels are managed in the Trello UI; the server
  can only apply or remove existing labels via `trello_add_label` /
  `trello_remove_label`)
- Card copy/template operations (duplicating a card with its checklists)
- Custom fields (reading or writing)
- Power-Up management
- Webhook registration or management
- Member management (inviting/removing board members)
- Organization/Workspace management
- Attachment file uploads (URL attachments via `trello_add_attachment` are
  supported; binary file uploads are not)
- Batch API operations (`/1/batch`)
- Client-side rate limiting or request queuing
 
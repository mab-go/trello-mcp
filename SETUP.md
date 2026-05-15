# Setup: trello-mcp

Step-by-step guide to configuring [trello-mcp](https://github.com/mab-go/trello-mcp)
with your Trello credentials.

---

## 1. Create a Trello Power-Up

1. Go to the [Trello Power-Up admin page](https://trello.com/power-ups/admin).
2. Click **New** to create a new integration.
3. Fill in the required fields (name, workspace, etc.) and submit.
4. On the integration's page, click **Generate a new API key**.
5. Copy the API key; you will need it in the next step and in your config file.

---

## 2. Generate a token

On the same page where you generated your API key, click the **Token** link next
to it. This opens an authorization page where you grant read/write access to
your Trello account.

Click **Allow**. The page displays your token. Copy it -- you will need it in
your config file.

The token does not expire unless you revoke it.

---

## 3. Create the config file

```bash
mkdir -p ~/.config/trello-mcp
```

Create `~/.config/trello-mcp/config.json` with your credentials:

```json
{
  "api_key": "YOUR_API_KEY",
  "token": "YOUR_TOKEN"
}
```

### Optional fields

| Field            | Type       | Purpose                                     |
|------------------|------------|---------------------------------------------|
| `default_board`  | `string`   | Board ID used when `board_id` is omitted    |
| `allowed_boards` | `string[]` | Restricts all operations to these board IDs |

Example with all fields:

```json
{
  "api_key": "YOUR_API_KEY",
  "token": "YOUR_TOKEN",
  "default_board": "BOARD_ID",
  "allowed_boards": ["BOARD_ID_1", "BOARD_ID_2"]
}
```

---

## 4. Validate credentials

```bash
trello-mcp auth
```

On success, this prints your Trello display name and username:

```
Authenticated as Jane Doe (@janedoe)
```

To check config file state without making an API call:

```bash
trello-mcp auth --status
```

---

## 5. Find your board ID (optional)

If you want to set `default_board` so you can omit `board_id` from tool
calls, you need the board's ID. The easiest way:

1. Register trello-mcp with your MCP client (see
   [README.md](README.md#4-register-with-your-mcp-client)).
2. Ask your AI client to call `trello_boards` -- it returns all your boards with
   their IDs.
3. Copy the desired board ID into `default_board` in your config file.

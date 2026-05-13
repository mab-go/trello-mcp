package server

import "github.com/mark3labs/mcp-go/mcp"

var toolTrelloBoards = mcp.NewTool(
	"trello_boards",
	mcp.WithDescription("List Trello boards for the authenticated member. Returns board IDs, names, and URLs."),
	mcp.WithString("query", mcp.Description("Filter boards whose name contains this string (case-insensitive). Omit to list all open boards.")),
)

var toolTrelloLists = mcp.NewTool(
	"trello_lists",
	mcp.WithDescription("Get all open lists on a Trello board. Returns list IDs, names, and positions."),
	mcp.WithString("board_id", mcp.Description("Board ID; uses default_board if omitted and configured")),
)

var toolTrelloCards = mcp.NewTool(
	"trello_cards",
	mcp.WithDescription("Get cards on a Trello board or in a specific list, with optional filtering by status, due date, or label."),
	mcp.WithString("board_id", mcp.Description("Board ID; uses default_board if omitted and configured")),
	mcp.WithString("list_id", mcp.Description("Scope to a specific list; omit for all cards on the board")),
	mcp.WithString("filter", mcp.Description("Card status filter"), mcp.Enum("open", "closed", "all")),
	mcp.WithString("due", mcp.Description("Filter by due date"), mcp.Enum("overdue", "day", "week", "month")),
	mcp.WithString("label", mcp.Description("Filter by label name (case-insensitive substring match)")),
	mcp.WithNumber("limit", mcp.Description("Max cards to return; default 100, max 1000")),
)

var toolTrelloGetCard = mcp.NewTool(
	"trello_get_card",
	mcp.WithDescription("Get full detail for a single Trello card, including checklists, comments, attachments, and members."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
)

var toolTrelloCreateCard = mcp.NewTool(
	"trello_create_card",
	mcp.WithDescription("Create a new card on a Trello list. Specify the target list by ID or name."),
	mcp.WithString("board_id", mcp.Description("Board ID; uses default_board if omitted and configured. Required for list_name resolution.")),
	mcp.WithString("list_id", mcp.Description("Target list ID; required unless list_name is provided")),
	mcp.WithString("list_name", mcp.Description("Target list name (case-insensitive exact match); alternative to list_id")),
	mcp.WithString("name", mcp.Required(), mcp.Description("Card title")),
	mcp.WithString("description", mcp.Description("Card description (Markdown supported)")),
	mcp.WithString("due", mcp.Description("Due date (ISO 8601 date or datetime)")),
	mcp.WithArray("labels", mcp.Description("Label names to apply (matched case-insensitively against board labels)")),
	mcp.WithString("position", mcp.Description("Position in the list: top or bottom (default: bottom)"), mcp.Enum("top", "bottom")),
)

var toolTrelloUpdateCard = mcp.NewTool(
	"trello_update_card",
	mcp.WithDescription("Update one or more fields on an existing Trello card. At least one field besides card_id must be provided."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
	mcp.WithString("name", mcp.Description("New card title")),
	mcp.WithString("description", mcp.Description("New card description")),
	mcp.WithString("due", mcp.Description("New due date (ISO 8601), or empty string to remove")),
	mcp.WithBoolean("due_complete", mcp.Description("Mark due date as complete or incomplete")),
	mcp.WithString("list_id", mcp.Description("Move card to this list by ID")),
	mcp.WithString("list_name", mcp.Description("Move card to this list by name (requires board context)")),
	mcp.WithString("board_id", mcp.Description("Board ID for list_name resolution; uses card's current board if omitted")),
	mcp.WithString("position", mcp.Description("Position in the list: top or bottom"), mcp.Enum("top", "bottom")),
)

var toolTrelloArchiveCard = mcp.NewTool(
	"trello_archive_card",
	mcp.WithDescription("Archive (close) a Trello card."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
)

var toolTrelloUnarchiveCard = mcp.NewTool(
	"trello_unarchive_card",
	mcp.WithDescription("Unarchive (reopen) a Trello card."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
)

// Tier 2 -- Checklists & Labels

var toolTrelloChecklists = mcp.NewTool(
	"trello_checklists",
	mcp.WithDescription("Get all checklists on a Trello card with their items."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
)

var toolTrelloCheckItem = mcp.NewTool(
	"trello_check_item",
	mcp.WithDescription("Check or uncheck a checklist item on a Trello card."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
	mcp.WithString("item_id", mcp.Required(), mcp.Description("Checklist item ID")),
	mcp.WithBoolean("complete", mcp.Required(), mcp.Description("true to check, false to uncheck")),
)

var toolTrelloAddChecklist = mcp.NewTool(
	"trello_add_checklist",
	mcp.WithDescription("Create a new checklist on a Trello card, optionally with initial items."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
	mcp.WithString("name", mcp.Required(), mcp.Description("Checklist name")),
	mcp.WithArray("items", mcp.Description("Initial checklist items to create")),
)

var toolTrelloAddCheckItem = mcp.NewTool(
	"trello_add_check_item",
	mcp.WithDescription("Add an item to an existing Trello checklist."),
	mcp.WithString("checklist_id", mcp.Required(), mcp.Description("Checklist ID")),
	mcp.WithString("name", mcp.Required(), mcp.Description("Item text")),
	mcp.WithBoolean("checked", mcp.Description("Create as already checked; default false")),
)

var toolTrelloLabels = mcp.NewTool(
	"trello_labels",
	mcp.WithDescription("Get all labels on a Trello board."),
	mcp.WithString("board_id", mcp.Description("Board ID; uses default_board if omitted and configured")),
)

var toolTrelloAddLabel = mcp.NewTool(
	"trello_add_label",
	mcp.WithDescription("Add an existing label to a Trello card."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
	mcp.WithString("label", mcp.Required(), mcp.Description("Label name (matched case-insensitively against board labels)")),
)

var toolTrelloRemoveLabel = mcp.NewTool(
	"trello_remove_label",
	mcp.WithDescription("Remove a label from a Trello card."),
	mcp.WithString("card_id", mcp.Required(), mcp.Description("Card ID or shortLink")),
	mcp.WithString("label", mcp.Required(), mcp.Description("Label name (matched case-insensitively against card's labels)")),
)

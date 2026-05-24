package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

// TrelloClient defines the Trello API operations used by handlers.
type TrelloClient interface {
	GetBoards(ctx context.Context) ([]trello.Board, error)
	GetBoardLists(ctx context.Context, boardID string) ([]trello.List, error)
	GetBoardCards(ctx context.Context, boardID, filter string) ([]trello.Card, error)
	GetListCards(ctx context.Context, listID, filter string) ([]trello.Card, error)
	GetCard(ctx context.Context, cardID string) (*trello.Card, error)
	GetCardBasic(ctx context.Context, cardID string) (*trello.Card, error)
	CreateCard(ctx context.Context, params url.Values) (*trello.Card, error)
	UpdateCard(ctx context.Context, cardID string, params url.Values) (*trello.Card, error)
	GetBoardLabels(ctx context.Context, boardID string) ([]trello.Label, error)
	GetCardChecklists(ctx context.Context, cardID string) ([]trello.Checklist, error)
	UpdateCheckItem(ctx context.Context, cardID, itemID string, params url.Values) (*trello.CheckItem, error)
	CreateChecklist(ctx context.Context, cardID, name string) (*trello.Checklist, error)
	CreateCheckItem(ctx context.Context, checklistID string, params url.Values) (*trello.CheckItem, error)
	AddCardLabel(ctx context.Context, cardID, labelID string) error
	RemoveCardLabel(ctx context.Context, cardID, labelID string) error
	GetBoardName(ctx context.Context, boardID string) (string, error)
	GetListName(ctx context.Context, listID string) (string, error)
	Search(ctx context.Context, query string, boardIDs []string, limit int) (*trello.SearchResult, error)
	AddComment(ctx context.Context, cardID, text string) (*trello.Action, error)
	AddAttachment(ctx context.Context, cardID string, params url.Values) (*trello.Attachment, error)
	GetChecklistBoard(ctx context.Context, checklistID string) (*trello.Board, error)
	CreateList(ctx context.Context, boardID string, params url.Values) (*trello.List, error)
}

// TrelloHandler holds shared state for all MCP tool handlers.
type TrelloHandler struct {
	client TrelloClient
	config *config.Config
}

// NewTrelloHandler creates a TrelloHandler with the given client and config.
func NewTrelloHandler(client TrelloClient, cfg *config.Config) *TrelloHandler {
	return &TrelloHandler{client: client, config: cfg}
}

func (h *TrelloHandler) resolveBoardID(args map[string]any) (string, *mcp.CallToolResult) {
	id, _ := args["board_id"].(string)
	if id == "" {
		id = h.config.DefaultBoard
	}
	if id == "" {
		return "", mcp.NewToolResultError(
			"missing board_id argument (no default_board configured). " +
				"Use trello_boards to find your board ID, or set default_board " +
				"in ~/.config/trello-mcp/config.json.",
		)
	}

	if errResult := h.checkAllowedBoard(id); errResult != nil {
		return "", errResult
	}

	return id, nil
}

func (h *TrelloHandler) checkAllowedBoard(boardID string) *mcp.CallToolResult {
	if len(h.config.AllowedBoards) == 0 {
		return nil
	}
	if !slices.Contains(h.config.AllowedBoards, boardID) {
		return mcp.NewToolResultError(
			fmt.Sprintf("board %q is not in the allowed list. Allowed boards: %v",
				boardID, h.config.AllowedBoards),
		)
	}
	return nil
}

func (h *TrelloHandler) resolveListByName(lists []trello.List, listName string) (string, *mcp.CallToolResult) {
	for _, l := range lists {
		if strings.EqualFold(l.Name, listName) {
			return l.ID, nil
		}
	}

	names := make([]string, len(lists))
	for i, l := range lists {
		names[i] = l.Name
	}
	return "", mcp.NewToolResultError(
		fmt.Sprintf("list %q not found. Available lists: %s",
			listName, strings.Join(names, ", ")),
	)
}

func (h *TrelloHandler) listDisplayName(ctx context.Context, listID string) string {
	if listID == "" {
		return ""
	}
	name, err := h.client.GetListName(ctx, listID)
	if err != nil {
		return ""
	}
	return name
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

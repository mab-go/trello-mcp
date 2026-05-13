package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

// TrelloHandler holds shared state for all MCP tool handlers.
type TrelloHandler struct {
	client *trello.Client
	config *config.Config
}

// NewTrelloHandler creates a TrelloHandler with the given client and config.
func NewTrelloHandler(client *trello.Client, cfg *config.Config) *TrelloHandler {
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

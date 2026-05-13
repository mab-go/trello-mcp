package handler

import (
	"context"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
)

type createListResponse struct {
	BoardID  string  `json:"board_id"`
	ListID   string  `json:"list_id"`
	Name     string  `json:"name"`
	Position float64 `json:"position"`
}

// CreateList handles the trello_create_list tool.
func (h *TrelloHandler) CreateList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	boardID, errResult := h.resolveBoardID(args)
	if errResult != nil {
		return errResult, nil
	}

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("missing required argument: name"), nil
	}

	pos, _ := args["position"].(string)
	if pos == "" {
		pos = "bottom"
	}

	params := url.Values{}
	params.Set("name", name)
	params.Set("pos", pos)

	list, err := h.client.CreateList(ctx, boardID, params)
	if err != nil {
		return mapAPIError(err)
	}

	return jsonResult(createListResponse{
		BoardID:  boardID,
		ListID:   list.ID,
		Name:     list.Name,
		Position: list.Pos,
	})
}

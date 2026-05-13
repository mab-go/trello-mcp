package handler

import (
	"context"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
)

type addCheckItemResponse struct {
	ChecklistID string `json:"checklist_id"`
	ItemID      string `json:"item_id"`
	Name        string `json:"name"`
	Complete    bool   `json:"complete"`
}

// AddCheckItem handles the trello_add_check_item tool.
func (h *TrelloHandler) AddCheckItem(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	checklistID, _ := args["checklist_id"].(string)
	if checklistID == "" {
		return mcp.NewToolResultError("missing required argument: checklist_id"), nil
	}

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("missing required argument: name"), nil
	}

	params := url.Values{}
	params.Set("name", name)

	if checked, ok := args["checked"].(bool); ok && checked {
		params.Set("checked", "true")
	}

	item, err := h.client.CreateCheckItem(ctx, checklistID, params)
	if err != nil {
		return mapAPIError(err)
	}

	return jsonResult(addCheckItemResponse{
		ChecklistID: checklistID,
		ItemID:      item.ID,
		Name:        item.Name,
		Complete:    item.State == "complete",
	})
}

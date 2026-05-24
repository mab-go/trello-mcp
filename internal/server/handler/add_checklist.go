package handler

import (
	"context"
	"net/url"

	"github.com/mab-go/logging"

	"github.com/mark3labs/mcp-go/mcp"
)

type addChecklistResponse struct {
	CardID      string `json:"card_id"`
	ChecklistID string `json:"checklist_id"`
	Name        string `json:"name"`
	ItemCount   int    `json:"item_count"`
}

// AddChecklist handles the trello_add_checklist tool.
func (h *TrelloHandler) AddChecklist(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_add_checklist")

	args := req.GetArguments()

	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("missing required argument: name"), nil
	}

	var items []string
	if rawItems, ok := args["items"]; ok {
		items, ok = toStringSlice(rawItems)
		if !ok {
			return mcp.NewToolResultError("items must be an array of strings"), nil
		}
	}

	card, err := h.client.GetCardBasic(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}
	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	checklist, err := h.client.CreateChecklist(ctx, cardID, name)
	if err != nil {
		return mapAPIError(log, err)
	}

	for _, itemName := range items {
		params := url.Values{}
		params.Set("name", itemName)
		if _, err := h.client.CreateCheckItem(ctx, checklist.ID, params); err != nil {
			return mapAPIError(log, err)
		}
	}

	log.WithFields(logging.Fields{"card_id": cardID, "checklist_id": checklist.ID, "name": checklist.Name}).Info(eventChecklistAdd)

	return jsonResult(addChecklistResponse{
		CardID:      cardID,
		ChecklistID: checklist.ID,
		Name:        checklist.Name,
		ItemCount:   len(items),
	})
}

package handler

import (
	"context"
	"net/url"

	"github.com/mab-go/logging"

	"github.com/mark3labs/mcp-go/mcp"
)

type checkItemResponse struct {
	CardID   string `json:"card_id"`
	ItemID   string `json:"item_id"`
	Name     string `json:"name"`
	Complete bool   `json:"complete"`
}

// CheckItem handles the trello_check_item tool.
func (h *TrelloHandler) CheckItem(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_check_item")

	args := req.GetArguments()

	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	itemID, _ := args["item_id"].(string)
	if itemID == "" {
		return mcp.NewToolResultError("missing required argument: item_id"), nil
	}

	complete, ok := args["complete"].(bool)
	if !ok {
		return mcp.NewToolResultError("missing required argument: complete"), nil
	}

	card, err := h.client.GetCardBasic(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}
	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	state := "incomplete"
	if complete {
		state = "complete"
	}

	params := url.Values{}
	params.Set("state", state)

	item, err := h.client.UpdateCheckItem(ctx, cardID, itemID, params)
	if err != nil {
		return mapAPIError(log, err)
	}

	log.WithFields(logging.Fields{"card_id": cardID, "item_id": item.ID, "complete": complete}).Info(eventCheckItemUpdate)

	return jsonResult(checkItemResponse{
		CardID:   cardID,
		ItemID:   item.ID,
		Name:     item.Name,
		Complete: item.State == "complete",
	})
}

package handler

import (
	"context"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
)

type unarchiveCardResponse struct {
	CardID   string `json:"card_id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

// UnarchiveCard handles the trello_unarchive_card tool.
func (h *TrelloHandler) UnarchiveCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	existing, err := h.client.GetCard(ctx, cardID)
	if err != nil {
		return mapAPIError(err)
	}
	if errResult := h.checkAllowedBoard(existing.IDBoard); errResult != nil {
		return errResult, nil
	}

	params := url.Values{}
	params.Set("closed", "false")

	card, err := h.client.UpdateCard(ctx, cardID, params)
	if err != nil {
		return mapAPIError(err)
	}

	return jsonResult(unarchiveCardResponse{
		CardID:   card.ID,
		Name:     card.Name,
		Archived: false,
	})
}

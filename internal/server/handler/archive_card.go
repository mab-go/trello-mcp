package handler

import (
	"context"
	"net/url"

	"github.com/mab-go/logging"

	"github.com/mark3labs/mcp-go/mcp"
)

type archiveCardResponse struct {
	CardID   string `json:"card_id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

// ArchiveCard handles the trello_archive_card tool.
func (h *TrelloHandler) ArchiveCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_archive_card")

	args := req.GetArguments()
	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	existing, err := h.client.GetCard(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}
	if errResult := h.checkAllowedBoard(existing.IDBoard); errResult != nil {
		return errResult, nil
	}

	params := url.Values{}
	params.Set("closed", "true")

	card, err := h.client.UpdateCard(ctx, cardID, params)
	if err != nil {
		return mapAPIError(log, err)
	}

	log.WithFields(logging.Fields{"card_id": card.ID, "name": card.Name}).Info(eventCardArchive)

	return jsonResult(archiveCardResponse{
		CardID:   card.ID,
		Name:     card.Name,
		Archived: true,
	})
}

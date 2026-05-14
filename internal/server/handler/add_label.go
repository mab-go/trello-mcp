package handler

import (
	"context"

	"github.com/mab-go/trello-mcp/internal/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type addLabelResponse struct {
	CardID  string `json:"card_id"`
	LabelID string `json:"label_id"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

// AddLabel handles the trello_add_label tool.
func (h *TrelloHandler) AddLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_add_label")

	args := req.GetArguments()

	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	labelName, _ := args["label"].(string)
	if labelName == "" {
		return mcp.NewToolResultError("missing required argument: label"), nil
	}

	card, err := h.client.GetCardBasic(ctx, cardID)
	if err != nil {
		return mapAPIError(err)
	}
	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	boardLabels, err := h.client.GetBoardLabels(ctx, card.IDBoard)
	if err != nil {
		return mapAPIError(err)
	}

	labelID, errResult := matchLabel(boardLabels, labelName)
	if errResult != nil {
		return errResult, nil
	}

	if err := h.client.AddCardLabel(ctx, cardID, labelID); err != nil {
		return mapAPIError(err)
	}

	matched := findLabelByID(boardLabels, labelID)

	log.WithFields(logging.Fields{"card_id": cardID, "label": matched.Name}).Info("Label added")

	return jsonResult(addLabelResponse{
		CardID:  cardID,
		LabelID: labelID,
		Name:    matched.Name,
		Color:   matched.Color,
	})
}

func findLabelByID(labels []trello.Label, id string) trello.Label {
	for _, l := range labels {
		if l.ID == id {
			return l
		}
	}
	return trello.Label{}
}

package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type removeLabelResponse struct {
	CardID  string `json:"card_id"`
	LabelID string `json:"label_id"`
	Name    string `json:"name"`
	Removed bool   `json:"removed"`
}

// RemoveLabel handles the trello_remove_label tool.
func (h *TrelloHandler) RemoveLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	labelID, matchedName, errResult := matchCardLabel(card.Labels, labelName)
	if errResult != nil {
		return errResult, nil
	}

	if err := h.client.RemoveCardLabel(ctx, cardID, labelID); err != nil {
		return mapAPIError(err)
	}

	return jsonResult(removeLabelResponse{
		CardID:  cardID,
		LabelID: labelID,
		Name:    matchedName,
		Removed: true,
	})
}

func matchCardLabel(cardLabels []trello.Label, name string) (string, string, *mcp.CallToolResult) {
	for _, l := range cardLabels {
		if strings.EqualFold(l.Name, name) {
			return l.ID, l.Name, nil
		}
	}

	available := make([]string, 0, len(cardLabels))
	for _, l := range cardLabels {
		if l.Name != "" {
			available = append(available, l.Name)
		}
	}

	if len(available) == 0 {
		return "", "", mcp.NewToolResultError(
			fmt.Sprintf("label %q is not on this card. The card has no labels.",
				name),
		)
	}
	return "", "", mcp.NewToolResultError(
		fmt.Sprintf("label %q is not on this card. Current labels: %s",
			name, strings.Join(available, ", ")),
	)
}

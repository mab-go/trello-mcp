package handler

import (
	"context"

	"github.com/mab-go/logging"

	"github.com/mark3labs/mcp-go/mcp"
)

type addCommentResponse struct {
	CardID    string `json:"card_id"`
	CommentID string `json:"comment_id"`
	Text      string `json:"text"`
}

// AddComment handles the trello_add_comment tool.
func (h *TrelloHandler) AddComment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_add_comment")

	args := req.GetArguments()

	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	text, _ := args["text"].(string)
	if text == "" {
		return mcp.NewToolResultError("missing required argument: text"), nil
	}

	card, err := h.client.GetCardBasic(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}
	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	action, err := h.client.AddComment(ctx, cardID, text)
	if err != nil {
		return mapAPIError(log, err)
	}

	responseText := action.Data.Text
	if len(responseText) > 200 {
		responseText = responseText[:200]
	}

	log.WithFields(logging.Fields{"card_id": cardID, "comment_id": action.ID}).Info(eventCommentAdd)

	return jsonResult(addCommentResponse{
		CardID:    cardID,
		CommentID: action.ID,
		Text:      responseText,
	})
}

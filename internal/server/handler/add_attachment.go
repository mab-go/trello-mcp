package handler

import (
	"context"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
)

type addAttachmentResponse struct {
	CardID       string `json:"card_id"`
	AttachmentID string `json:"attachment_id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
}

// AddAttachment handles the trello_add_attachment tool.
func (h *TrelloHandler) AddAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		return mcp.NewToolResultError("missing required argument: url"), nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return mcp.NewToolResultError(
			"url must use http:// or https:// scheme",
		), nil
	}

	card, err := h.client.GetCardBasic(ctx, cardID)
	if err != nil {
		return mapAPIError(err)
	}
	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	params := url.Values{}
	params.Set("url", rawURL)
	if name, _ := args["name"].(string); name != "" {
		params.Set("name", name)
	}

	att, err := h.client.AddAttachment(ctx, cardID, params)
	if err != nil {
		return mapAPIError(err)
	}

	return jsonResult(addAttachmentResponse{
		CardID:       cardID,
		AttachmentID: att.ID,
		Name:         att.Name,
		URL:          att.URL,
	})
}

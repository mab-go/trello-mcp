package handler

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

type labelEntry struct {
	LabelID string `json:"label_id"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

type labelsResponse struct {
	BoardID string       `json:"board_id"`
	Count   int          `json:"count"`
	Labels  []labelEntry `json:"labels"`
}

// Labels handles the trello_labels tool.
func (h *TrelloHandler) Labels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	boardID, errResult := h.resolveBoardID(args)
	if errResult != nil {
		return errResult, nil
	}

	labels, err := h.client.GetBoardLabels(ctx, boardID)
	if err != nil {
		return mapAPIError(err)
	}

	entries := make([]labelEntry, 0, len(labels))
	for _, l := range labels {
		if l.Name == "" {
			continue
		}
		entries = append(entries, labelEntry{
			LabelID: l.ID,
			Name:    l.Name,
			Color:   l.Color,
		})
	}

	return jsonResult(labelsResponse{
		BoardID: boardID,
		Count:   len(entries),
		Labels:  entries,
	})
}

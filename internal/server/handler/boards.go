package handler

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/mab-go/trello-mcp/internal/logging"

	"github.com/mark3labs/mcp-go/mcp"
)

type boardEntry struct {
	BoardID      string `json:"board_id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	LastActivity string `json:"last_activity,omitempty"`
}

type boardsResponse struct {
	Count  int          `json:"count"`
	Boards []boardEntry `json:"boards"`
}

// Boards handles the trello_boards tool.
func (h *TrelloHandler) Boards(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_boards")

	args := req.GetArguments()
	query, _ := args["query"].(string)

	boards, err := h.client.GetBoards(ctx)
	if err != nil {
		return mapAPIError(err)
	}

	var entries []boardEntry
	for _, b := range boards {
		if len(h.config.AllowedBoards) > 0 && !slices.Contains(h.config.AllowedBoards, b.ID) {
			continue
		}

		if query != "" && !strings.Contains(strings.ToLower(b.Name), strings.ToLower(query)) {
			continue
		}

		entry := boardEntry{
			BoardID: b.ID,
			Name:    b.Name,
			URL:     b.URL,
		}

		if b.DateLastActivity != "" {
			if t, err := time.Parse(time.RFC3339, b.DateLastActivity); err == nil {
				entry.LastActivity = t.UTC().Format("2006-01-02")
			}
		}

		entries = append(entries, entry)
	}

	log.WithField("count", len(entries)).Debug("Listed boards")

	return jsonResult(boardsResponse{
		Count:  len(entries),
		Boards: entries,
	})
}

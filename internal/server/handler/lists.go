package handler

import (
	"context"

	"github.com/mab-go/trello-mcp/internal/logging"

	"github.com/mark3labs/mcp-go/mcp"
)

type listEntry struct {
	ListID   string  `json:"list_id"`
	Name     string  `json:"name"`
	Position float64 `json:"position"`
}

type listsResponse struct {
	BoardID string      `json:"board_id"`
	Count   int         `json:"count"`
	Lists   []listEntry `json:"lists"`
}

// Lists handles the trello_lists tool.
func (h *TrelloHandler) Lists(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_lists")

	args := req.GetArguments()

	boardID, errResult := h.resolveBoardID(args)
	if errResult != nil {
		return errResult, nil
	}

	lists, err := h.client.GetBoardLists(ctx, boardID)
	if err != nil {
		return mapAPIError(err)
	}

	entries := make([]listEntry, len(lists))
	for i, l := range lists {
		entries[i] = listEntry{
			ListID:   l.ID,
			Name:     l.Name,
			Position: l.Pos,
		}
	}

	log.WithFields(logging.Fields{"board_id": boardID, "count": len(entries)}).Debug("Listed lists")

	return jsonResult(listsResponse{
		BoardID: boardID,
		Count:   len(entries),
		Lists:   entries,
	})
}

package handler

import (
	"context"
	"slices"

	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type searchCardEntry struct {
	CardID    string   `json:"card_id"`
	Name      string   `json:"name"`
	BoardName string   `json:"board_name"`
	ListName  string   `json:"list_name"`
	Due       string   `json:"due,omitempty"`
	Labels    []string `json:"labels"`
	URL       string   `json:"url"`
	Closed    bool     `json:"closed"`
}

type searchBoardEntry struct {
	BoardID string `json:"board_id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Closed  bool   `json:"closed"`
}

type searchResponse struct {
	Query      string             `json:"query"`
	CardCount  int                `json:"card_count"`
	BoardCount int                `json:"board_count"`
	Cards      []searchCardEntry  `json:"cards"`
	Boards     []searchBoardEntry `json:"boards"`
}

// Search handles the trello_search tool.
func (h *TrelloHandler) Search(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	query, _ := args["query"].(string)
	if query == "" {
		return mcp.NewToolResultError("missing required argument: query"), nil
	}

	boardID, _ := args["board_id"].(string)
	if boardID == "" {
		boardID = h.config.DefaultBoard
	}

	if boardID != "" {
		if errResult := h.checkAllowedBoard(boardID); errResult != nil {
			return errResult, nil
		}
	}

	limit := parseSearchLimit(args)

	var boardIDs []string
	if boardID != "" {
		boardIDs = []string{boardID}
	}

	result, err := h.client.Search(ctx, query, boardIDs, limit)
	if err != nil {
		return mapAPIError(err)
	}

	cards := h.filterSearchCards(result.Cards)
	boards := h.filterSearchBoards(result.Boards)

	return jsonResult(searchResponse{
		Query:      query,
		CardCount:  len(cards),
		BoardCount: len(boards),
		Cards:      cards,
		Boards:     boards,
	})
}

func parseSearchLimit(args map[string]any) int {
	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 20 {
			limit = 20
		}
	}
	return limit
}

func (h *TrelloHandler) filterSearchCards(cards []trello.Card) []searchCardEntry {
	hasAllowedFilter := len(h.config.AllowedBoards) > 0
	entries := make([]searchCardEntry, 0, len(cards))
	for _, c := range cards {
		if hasAllowedFilter && !slices.Contains(h.config.AllowedBoards, c.IDBoard) {
			continue
		}
		entries = append(entries, buildSearchCardEntry(c))
	}
	return entries
}

func buildSearchCardEntry(c trello.Card) searchCardEntry {
	entry := searchCardEntry{
		CardID: c.ID,
		Name:   c.Name,
		URL:    c.ShortURL,
		Closed: c.Closed,
	}
	if c.Board != nil {
		entry.BoardName = c.Board.Name
	}
	if c.List != nil {
		entry.ListName = c.List.Name
	}
	if c.Due != nil {
		entry.Due = c.Due.UTC().Format("2006-01-02")
	}
	entry.Labels = make([]string, 0, len(c.Labels))
	for _, l := range c.Labels {
		if l.Name != "" {
			entry.Labels = append(entry.Labels, l.Name)
		}
	}
	return entry
}

func (h *TrelloHandler) filterSearchBoards(boards []trello.Board) []searchBoardEntry {
	hasAllowedFilter := len(h.config.AllowedBoards) > 0
	entries := make([]searchBoardEntry, 0, len(boards))
	for _, b := range boards {
		if hasAllowedFilter && !slices.Contains(h.config.AllowedBoards, b.ID) {
			continue
		}
		entries = append(entries, searchBoardEntry{
			BoardID: b.ID,
			Name:    b.Name,
			URL:     b.URL,
			Closed:  b.Closed,
		})
	}
	return entries
}

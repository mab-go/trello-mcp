package handler

import (
	"context"
	"time"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type boardSummaryListEntry struct {
	ListName  string `json:"list_name"`
	CardCount int    `json:"card_count"`
}

type boardSummaryCardEntry struct {
	CardID   string   `json:"card_id"`
	Name     string   `json:"name"`
	ListName string   `json:"list_name"`
	Due      string   `json:"due"`
	Labels   []string `json:"labels"`
}

type boardSummaryResponse struct {
	BoardID      string                  `json:"board_id"`
	BoardName    string                  `json:"board_name"`
	TotalCards   int                     `json:"total_cards"`
	OverdueCount int                     `json:"overdue_count"`
	DueSoonCount int                     `json:"due_soon_count"`
	Lists        []boardSummaryListEntry `json:"lists"`
	Overdue      []boardSummaryCardEntry `json:"overdue,omitempty"`
	DueSoon      []boardSummaryCardEntry `json:"due_soon,omitempty"`
}

// BoardSummary handles the trello_board_summary tool.
func (h *TrelloHandler) BoardSummary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_board_summary")

	args := req.GetArguments()

	boardID, errResult := h.resolveBoardID(args)
	if errResult != nil {
		return errResult, nil
	}

	boardName, err := h.client.GetBoardName(ctx, boardID)
	if err != nil {
		return mapAPIError(log, err)
	}

	lists, err := h.client.GetBoardLists(ctx, boardID)
	if err != nil {
		return mapAPIError(log, err)
	}

	cards, err := h.client.GetBoardCards(ctx, boardID, "open")
	if err != nil {
		return mapAPIError(log, err)
	}

	listNames := make(map[string]string, len(lists))
	for _, l := range lists {
		listNames[l.ID] = l.Name
	}

	listCounts := make(map[string]int, len(lists))
	for _, c := range cards {
		listCounts[c.IDList]++
	}

	listEntries := make([]boardSummaryListEntry, len(lists))
	for i, l := range lists {
		listEntries[i] = boardSummaryListEntry{
			ListName:  l.Name,
			CardCount: listCounts[l.ID],
		}
	}

	overdue, dueSoon := classifyDueCards(cards, listNames)

	log.WithFields(logging.Fields{"board_id": boardID, "total_cards": len(cards)}).Info(eventBoardSummarize)

	return jsonResult(boardSummaryResponse{
		BoardID:      boardID,
		BoardName:    boardName,
		TotalCards:   len(cards),
		OverdueCount: len(overdue),
		DueSoonCount: len(dueSoon),
		Lists:        listEntries,
		Overdue:      overdue,
		DueSoon:      dueSoon,
	})
}

func classifyDueCards(cards []trello.Card, listNames map[string]string) (overdue, dueSoon []boardSummaryCardEntry) {
	now := time.Now().UTC()
	dueSoonCutoff := now.Add(7 * 24 * time.Hour)

	for _, c := range cards {
		if c.Due == nil || c.DueComplete {
			continue
		}
		due := c.Due.UTC()
		entry := buildSummaryCardEntry(c, listNames)

		if due.Before(now) {
			overdue = append(overdue, entry)
		} else if due.Before(dueSoonCutoff) {
			dueSoon = append(dueSoon, entry)
		}
	}
	return overdue, dueSoon
}

func buildSummaryCardEntry(c trello.Card, listNames map[string]string) boardSummaryCardEntry {
	labels := make([]string, 0, len(c.Labels))
	for _, l := range c.Labels {
		if l.Name != "" {
			labels = append(labels, l.Name)
		}
	}
	return boardSummaryCardEntry{
		CardID:   c.ID,
		Name:     c.Name,
		ListName: listNames[c.IDList],
		Due:      c.Due.UTC().Format("2006-01-02"),
		Labels:   labels,
	}
}

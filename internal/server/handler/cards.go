package handler

import (
	"context"
	"strings"
	"time"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type cardEntry struct {
	CardID       string   `json:"card_id"`
	Name         string   `json:"name"`
	ListName     string   `json:"list_name"`
	Due          string   `json:"due,omitempty"`
	DueComplete  bool     `json:"due_complete"`
	Labels       []string `json:"labels"`
	Members      []string `json:"members"`
	URL          string   `json:"url"`
	LastActivity string   `json:"last_activity,omitempty"`
}

type cardsResponse struct {
	BoardID string      `json:"board_id"`
	ListID  string      `json:"list_id,omitempty"`
	Filter  string      `json:"filter"`
	Count   int         `json:"count"`
	Cards   []cardEntry `json:"cards"`
}

// Cards handles the trello_cards tool.
func (h *TrelloHandler) Cards(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_cards")

	args := req.GetArguments()

	boardID, errResult := h.resolveBoardID(args)
	if errResult != nil {
		return errResult, nil
	}

	listID, _ := args["list_id"].(string)
	filter, _ := args["filter"].(string)
	if filter == "" {
		filter = "open"
	}
	dueFilter, _ := args["due"].(string)
	labelFilter, _ := args["label"].(string)
	limit := parseLimit(args)

	cards, err := h.fetchCards(ctx, boardID, listID, filter)
	if err != nil {
		return mapAPIError(log, err)
	}

	listNames, err := h.buildListNameMap(ctx, boardID)
	if err != nil {
		return mapAPIError(log, err)
	}

	now := time.Now().UTC()
	entries := make([]cardEntry, 0)

	for _, c := range cards {
		if !matchAllFilters(c, filter, dueFilter, labelFilter, now) {
			continue
		}

		entries = append(entries, buildCardEntry(c, listNames))

		if len(entries) >= limit {
			break
		}
	}

	log.WithFields(logging.Fields{"board_id": boardID, "count": len(entries)}).Info(eventCardList)

	return jsonResult(cardsResponse{
		BoardID: boardID,
		ListID:  listID,
		Filter:  filter,
		Count:   len(entries),
		Cards:   entries,
	})
}

func parseLimit(args map[string]any) int {
	limit := 100
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 1000 {
			limit = 1000
		}
	}
	return limit
}

func (h *TrelloHandler) fetchCards(ctx context.Context, boardID, listID, filter string) ([]trello.Card, error) {
	if listID != "" {
		return h.client.GetListCards(ctx, listID, filter)
	}
	return h.client.GetBoardCards(ctx, boardID, filter)
}

func (h *TrelloHandler) buildListNameMap(ctx context.Context, boardID string) (map[string]string, error) {
	lists, err := h.client.GetBoardLists(ctx, boardID)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(lists))
	for _, l := range lists {
		m[l.ID] = l.Name
	}
	return m, nil
}

func buildCardEntry(c trello.Card, listNames map[string]string) cardEntry {
	entry := cardEntry{
		CardID:      c.ID,
		Name:        c.Name,
		ListName:    listNames[c.IDList],
		DueComplete: c.DueComplete,
		URL:         c.ShortURL,
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

	entry.Members = make([]string, 0, len(c.Members))
	for _, m := range c.Members {
		entry.Members = append(entry.Members, m.Username)
	}

	if c.DateLastActivity != "" {
		if t, parseErr := time.Parse(time.RFC3339, c.DateLastActivity); parseErr == nil {
			entry.LastActivity = t.UTC().Format("2006-01-02 15:04")
		}
	}

	return entry
}

func matchAllFilters(c trello.Card, filter, dueFilter, labelFilter string, now time.Time) bool {
	if !matchFilter(c, filter) {
		return false
	}
	if dueFilter != "" && !matchDueFilter(c, dueFilter, now) {
		return false
	}
	if labelFilter != "" && !matchLabelFilter(c, labelFilter) {
		return false
	}
	return true
}

func matchFilter(c trello.Card, filter string) bool {
	switch filter {
	case "open":
		return !c.Closed
	case "closed":
		return c.Closed
	case "all":
		return true
	default:
		return !c.Closed
	}
}

func matchDueFilter(c trello.Card, dueFilter string, now time.Time) bool {
	if c.Due == nil {
		return false
	}

	due := c.Due.UTC()
	isOverdue := due.Before(now) && !c.DueComplete

	switch dueFilter {
	case "overdue":
		return isOverdue
	case "day":
		return isOverdue || due.Before(now.Add(24*time.Hour))
	case "week":
		return isOverdue || due.Before(now.Add(7*24*time.Hour))
	case "month":
		return isOverdue || due.Before(now.Add(30*24*time.Hour))
	default:
		return true
	}
}

func matchLabelFilter(c trello.Card, labelFilter string) bool {
	lower := strings.ToLower(labelFilter)
	for _, l := range c.Labels {
		if strings.Contains(strings.ToLower(l.Name), lower) {
			return true
		}
	}
	return false
}

package handler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestMatchFilter(t *testing.T) {
	tests := []struct {
		name   string
		card   trello.Card
		filter string
		want   bool
	}{
		{"open matches open card", trello.Card{Closed: false}, "open", true},
		{"open rejects closed card", trello.Card{Closed: true}, "open", false},
		{"closed matches closed card", trello.Card{Closed: true}, "closed", true},
		{"closed rejects open card", trello.Card{Closed: false}, "closed", false},
		{"all matches open card", trello.Card{Closed: false}, "all", true},
		{"all matches closed card", trello.Card{Closed: true}, "all", true},
		{"empty string matches open card", trello.Card{Closed: false}, "", true},
		{"empty string rejects closed card", trello.Card{Closed: true}, "", false},
		{"unknown filter defaults to open behavior", trello.Card{Closed: false}, "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchFilter(tt.card, tt.filter)
			if got != tt.want {
				t.Errorf("matchFilter(%v, %q) = %v, want %v", tt.card.Closed, tt.filter, got, tt.want)
			}
		})
	}
}

func TestMatchDueFilter(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	past := now.Add(-2 * time.Hour)
	in12h := now.Add(12 * time.Hour)
	in3d := now.Add(3 * 24 * time.Hour)
	in10d := now.Add(10 * 24 * time.Hour)
	in45d := now.Add(45 * 24 * time.Hour)

	tests := []struct {
		name      string
		card      trello.Card
		dueFilter string
		want      bool
	}{
		// empty filter always matches
		{"empty filter with due", trello.Card{Due: &in12h}, "", true},
		{"empty filter nil due", trello.Card{Due: nil}, "", false},

		// nil due returns false for any named filter
		{"overdue nil due", trello.Card{Due: nil}, "overdue", false},
		{"day nil due", trello.Card{Due: nil}, "day", false},
		{"week nil due", trello.Card{Due: nil}, "week", false},
		{"month nil due", trello.Card{Due: nil}, "month", false},

		// overdue
		{"overdue past due", trello.Card{Due: &past}, "overdue", true},
		{"overdue future due", trello.Card{Due: &in12h}, "overdue", false},
		{"overdue past but complete", trello.Card{Due: &past, DueComplete: true}, "overdue", false},

		// day (within 24h)
		{"day due in 12h", trello.Card{Due: &in12h}, "day", true},
		{"day due in 3d", trello.Card{Due: &in3d}, "day", false},
		{"day overdue card", trello.Card{Due: &past}, "day", true},
		{"day due complete", trello.Card{Due: &in12h, DueComplete: true}, "day", true},

		// week (within 7 days)
		{"week due in 3d", trello.Card{Due: &in3d}, "week", true},
		{"week due in 10d", trello.Card{Due: &in10d}, "week", false},
		{"week overdue card", trello.Card{Due: &past}, "week", true},

		// month (within 30 days)
		{"month due in 10d", trello.Card{Due: &in10d}, "month", true},
		{"month due in 45d", trello.Card{Due: &in45d}, "month", false},
		{"month overdue card", trello.Card{Due: &past}, "month", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchDueFilter(tt.card, tt.dueFilter, now)
			if got != tt.want {
				t.Errorf("matchDueFilter(%q) = %v, want %v", tt.dueFilter, got, tt.want)
			}
		})
	}
}

func TestMatchLabelFilter(t *testing.T) {
	card := trello.Card{
		Labels: []trello.Label{
			{Name: "Bug"},
			{Name: "High Priority"},
		},
	}

	tests := []struct {
		name        string
		card        trello.Card
		labelFilter string
		want        bool
	}{
		{"empty filter always matches", card, "", true},
		{"exact match", card, "Bug", true},
		{"case-insensitive match", card, "bug", true},
		{"substring match", card, "prior", true},
		{"no match", card, "feature", false},
		{"empty labels no match", trello.Card{}, "Bug", false},
		{"mixed case substring", card, "HIGH", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchLabelFilter(tt.card, tt.labelFilter)
			if got != tt.want {
				t.Errorf("matchLabelFilter(%q) = %v, want %v", tt.labelFilter, got, tt.want)
			}
		})
	}
}

func TestMatchAllFilters(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	in12h := now.Add(12 * time.Hour)

	card := trello.Card{
		Closed: false,
		Due:    &in12h,
		Labels: []trello.Label{{Name: "Bug"}},
	}

	tests := []struct {
		name        string
		card        trello.Card
		filter      string
		dueFilter   string
		labelFilter string
		want        bool
	}{
		{"all filters pass", card, "open", "day", "Bug", true},
		{"status filter fails", card, "closed", "day", "Bug", false},
		{"due filter fails", card, "open", "overdue", "Bug", false},
		{"label filter fails", card, "open", "day", "Feature", false},
		{"empty optional filters pass", card, "open", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAllFilters(tt.card, tt.filter, tt.dueFilter, tt.labelFilter, now)
			if got != tt.want {
				t.Errorf("matchAllFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLimit_Cards(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want int
	}{
		{"default when not present", map[string]any{}, 100},
		{"respects float64 value", map[string]any{"limit": float64(50)}, 50},
		{"clamps to minimum 1", map[string]any{"limit": float64(0)}, 100},
		{"negative treated as absent", map[string]any{"limit": float64(-5)}, 100},
		{"clamps to maximum 1000", map[string]any{"limit": float64(2000)}, 1000},
		{"exact minimum", map[string]any{"limit": float64(1)}, 1},
		{"exact maximum", map[string]any{"limit": float64(1000)}, 1000},
		{"wrong type ignored", map[string]any{"limit": "50"}, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLimit(tt.args)
			if got != tt.want {
				t.Errorf("parseLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildCardEntry(t *testing.T) {
	due := time.Date(2025, 7, 4, 18, 30, 0, 0, time.UTC)

	listNames := map[string]string{
		"list-1": "To Do",
		"list-2": "In Progress",
	}

	tests := []struct {
		name string
		card trello.Card
		want cardEntry
	}{
		{
			name: "full card with due date and labels",
			card: trello.Card{
				ID:               "card-1",
				Name:             "Fix login bug",
				IDList:           "list-1",
				ShortURL:         "https://trello.com/c/abc",
				Due:              &due,
				DueComplete:      false,
				Labels:           []trello.Label{{Name: "Bug"}, {Name: "Urgent"}},
				Members:          []trello.CardMember{{Username: "alice"}, {Username: "bob"}},
				DateLastActivity: "2025-06-10T14:30:00.000Z",
			},
			want: cardEntry{
				CardID:       "card-1",
				Name:         "Fix login bug",
				ListName:     "To Do",
				Due:          "2025-07-04",
				DueComplete:  false,
				Labels:       []string{"Bug", "Urgent"},
				Members:      []string{"alice", "bob"},
				URL:          "https://trello.com/c/abc",
				LastActivity: "2025-06-10 14:30",
			},
		},
		{
			name: "card without due date",
			card: trello.Card{
				ID:     "card-2",
				Name:   "Research task",
				IDList: "list-2",
			},
			want: cardEntry{
				CardID:   "card-2",
				Name:     "Research task",
				ListName: "In Progress",
				Due:      "",
				Labels:   []string{},
				Members:  []string{},
			},
		},
		{
			name: "card with unknown list ID",
			card: trello.Card{
				ID:     "card-3",
				Name:   "Orphan card",
				IDList: "list-unknown",
			},
			want: cardEntry{
				CardID:   "card-3",
				Name:     "Orphan card",
				ListName: "",
				Labels:   []string{},
				Members:  []string{},
			},
		},
		{
			name: "labels with empty names are skipped",
			card: trello.Card{
				ID:     "card-4",
				Name:   "Labeled card",
				IDList: "list-1",
				Labels: []trello.Label{{Name: ""}, {Name: "Bug"}, {Name: ""}},
			},
			want: cardEntry{
				CardID:   "card-4",
				Name:     "Labeled card",
				ListName: "To Do",
				Labels:   []string{"Bug"},
				Members:  []string{},
			},
		},
		{
			name: "due complete is preserved",
			card: trello.Card{
				ID:          "card-5",
				Name:        "Done task",
				IDList:      "list-1",
				Due:         &due,
				DueComplete: true,
			},
			want: cardEntry{
				CardID:      "card-5",
				Name:        "Done task",
				ListName:    "To Do",
				Due:         "2025-07-04",
				DueComplete: true,
				Labels:      []string{},
				Members:     []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCardEntry(tt.card, listNames)
			if got.CardID != tt.want.CardID {
				t.Errorf("CardID = %q, want %q", got.CardID, tt.want.CardID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.ListName != tt.want.ListName {
				t.Errorf("ListName = %q, want %q", got.ListName, tt.want.ListName)
			}
			if got.Due != tt.want.Due {
				t.Errorf("Due = %q, want %q", got.Due, tt.want.Due)
			}
			if got.DueComplete != tt.want.DueComplete {
				t.Errorf("DueComplete = %v, want %v", got.DueComplete, tt.want.DueComplete)
			}
			if got.URL != tt.want.URL {
				t.Errorf("URL = %q, want %q", got.URL, tt.want.URL)
			}
			if got.LastActivity != tt.want.LastActivity {
				t.Errorf("LastActivity = %q, want %q", got.LastActivity, tt.want.LastActivity)
			}
			if len(got.Labels) != len(tt.want.Labels) {
				t.Fatalf("Labels len = %d, want %d", len(got.Labels), len(tt.want.Labels))
			}
			for i := range got.Labels {
				if got.Labels[i] != tt.want.Labels[i] {
					t.Errorf("Labels[%d] = %q, want %q", i, got.Labels[i], tt.want.Labels[i])
				}
			}
			if len(got.Members) != len(tt.want.Members) {
				t.Fatalf("Members len = %d, want %d", len(got.Members), len(tt.want.Members))
			}
			for i := range got.Members {
				if got.Members[i] != tt.want.Members[i] {
					t.Errorf("Members[%d] = %q, want %q", i, got.Members[i], tt.want.Members[i])
				}
			}
		})
	}
}

func TestCards_SuccessWithBoardID(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Card One", IDList: "list-1", Closed: false},
		{ID: "c2", Name: "Card Two", IDList: "list-2", Closed: false},
	}
	lists := []trello.List{
		{ID: "list-1", Name: "To Do"},
		{ID: "list-2", Name: "Done"},
	}

	mc := &mockClient{
		getBoardCardsFn: func(_ context.Context, boardID, _ string) ([]trello.Card, error) {
			if boardID != "board-1" {
				t.Errorf("unexpected board ID: %s", boardID)
			}
			return cards, nil
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return lists, nil
		},
	}

	h := testHandler(mc, nil)
	req := testRequest(map[string]any{"board_id": "board-1"})

	result, err := h.Cards(testCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp cardsResponse
	resultJSON(t, result, &resp)

	if resp.BoardID != "board-1" {
		t.Errorf("BoardID = %q, want %q", resp.BoardID, "board-1")
	}
	if resp.Filter != "open" {
		t.Errorf("Filter = %q, want %q", resp.Filter, "open")
	}
	if resp.Count != 2 {
		t.Errorf("Count = %d, want %d", resp.Count, 2)
	}
	if len(resp.Cards) != 2 {
		t.Fatalf("len(Cards) = %d, want 2", len(resp.Cards))
	}
	if resp.Cards[0].CardID != "c1" {
		t.Errorf("Cards[0].CardID = %q, want %q", resp.Cards[0].CardID, "c1")
	}
	if resp.Cards[1].ListName != "Done" {
		t.Errorf("Cards[1].ListName = %q, want %q", resp.Cards[1].ListName, "Done")
	}
}

func TestCards_SuccessWithListID(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Card One", IDList: "list-1", Closed: false},
	}
	lists := []trello.List{
		{ID: "list-1", Name: "To Do"},
	}

	mc := &mockClient{
		getListCardsFn: func(_ context.Context, listID, _ string) ([]trello.Card, error) {
			if listID != "list-1" {
				t.Errorf("unexpected list ID: %s", listID)
			}
			return cards, nil
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return lists, nil
		},
	}

	h := testHandler(mc, nil)
	req := testRequest(map[string]any{
		"board_id": "board-1",
		"list_id":  "list-1",
	})

	result, err := h.Cards(testCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp cardsResponse
	resultJSON(t, result, &resp)

	if resp.ListID != "list-1" {
		t.Errorf("ListID = %q, want %q", resp.ListID, "list-1")
	}
	if resp.Count != 1 {
		t.Errorf("Count = %d, want %d", resp.Count, 1)
	}
}

func TestCards_MissingBoardError(t *testing.T) {
	mc := &mockClient{}
	h := testHandler(mc, &config.Config{})
	req := testRequest(map[string]any{})

	result, err := h.Cards(testCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := resultText(result)
	if !strings.Contains(text, "missing board_id") {
		t.Errorf("expected missing board_id error, got: %s", text)
	}
}

func TestCards_APIError(t *testing.T) {
	mc := &mockClient{
		getBoardCardsFn: func(_ context.Context, _, _ string) ([]trello.Card, error) {
			return nil, &trello.APIError{StatusCode: 404, Body: "board not found"}
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return nil, nil
		},
	}

	h := testHandler(mc, nil)
	req := testRequest(map[string]any{"board_id": "bad-board"})

	result, err := h.Cards(testCtx(), req)
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}

	text := resultText(result)
	if !strings.Contains(text, "not found (404)") {
		t.Errorf("expected 404 error message, got: %s", text)
	}
}

func TestCards_DefaultBoard(t *testing.T) {
	var capturedBoardID string
	mc := &mockClient{
		getBoardCardsFn: func(_ context.Context, boardID, _ string) ([]trello.Card, error) {
			capturedBoardID = boardID
			return nil, nil
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return nil, nil
		},
	}

	cfg := &config.Config{DefaultBoard: "default-board-id"}
	h := testHandler(mc, cfg)
	req := testRequest(map[string]any{})

	_, err := h.Cards(testCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBoardID != "default-board-id" {
		t.Errorf("board ID = %q, want %q", capturedBoardID, "default-board-id")
	}
}

func TestCards_FilterAndLimit(t *testing.T) {
	cards := make([]trello.Card, 10)
	for i := range cards {
		cards[i] = trello.Card{
			ID:     fmt.Sprintf("c%d", i),
			Name:   fmt.Sprintf("Card %d", i),
			IDList: "list-1",
		}
	}

	mc := &mockClient{
		getBoardCardsFn: func(_ context.Context, _, _ string) ([]trello.Card, error) {
			return cards, nil
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return []trello.List{{ID: "list-1", Name: "Backlog"}}, nil
		},
	}

	h := testHandler(mc, nil)
	req := testRequest(map[string]any{
		"board_id": "board-1",
		"limit":    float64(3),
	})

	result, err := h.Cards(testCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp cardsResponse
	resultJSON(t, result, &resp)

	if resp.Count != 3 {
		t.Errorf("Count = %d, want 3", resp.Count)
	}
	if len(resp.Cards) != 3 {
		t.Errorf("len(Cards) = %d, want 3", len(resp.Cards))
	}
}

func TestCardsLogging(t *testing.T) {
	mc := &mockClient{
		getBoardCardsFn: func(_ context.Context, _, _ string) ([]trello.Card, error) {
			return []trello.Card{
				{ID: "c1", Name: "Card One", IDList: "l1"},
				{ID: "c2", Name: "Card Two", IDList: "l1"},
			}, nil
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return []trello.List{{ID: "l1", Name: "Backlog"}}, nil
		},
	}
	h := testHandler(mc, nil)
	ctx, rec := testCtxRecording()

	result, err := h.Cards(ctx, testRequest(map[string]any{"board_id": "board-1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCardList) {
		t.Fatal("expected INFO entry for eventCardList")
	}
	entry := rec.Entry("INFO", eventCardList)
	if !entry.HasField("board_id") {
		t.Error("expected field 'board_id' on eventCardList entry")
	}
	if !entry.HasField("count") {
		t.Error("expected field 'count' on eventCardList entry")
	}
}

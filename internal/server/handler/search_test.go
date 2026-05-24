package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestParseSearchLimit(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want int
	}{
		{"default", map[string]any{}, 10},
		{"explicit", map[string]any{"limit": float64(5)}, 5},
		{"max clamp", map[string]any{"limit": float64(50)}, 20},
		{"negative", map[string]any{"limit": float64(-1)}, 10},
		{"zero", map[string]any{"limit": float64(0)}, 10},
		{"twenty", map[string]any{"limit": float64(20)}, 20},
		{"wrong type", map[string]any{"limit": "10"}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSearchLimit(tt.args)
			if got != tt.want {
				t.Errorf("parseSearchLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildSearchCardEntry(t *testing.T) {
	due := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	card := trello.Card{
		ID:       "c1",
		Name:     "Test Card",
		ShortURL: "https://trello.com/c/c1",
		Closed:   false,
		Board:    &trello.Board{Name: "My Board"},
		List:     &trello.List{Name: "To Do"},
		Due:      &due,
		Labels:   []trello.Label{{Name: "Bug"}, {Name: ""}, {Name: "P1"}},
	}

	entry := buildSearchCardEntry(card)

	if entry.CardID != "c1" {
		t.Errorf("CardID = %q, want %q", entry.CardID, "c1")
	}
	if entry.BoardName != "My Board" {
		t.Errorf("BoardName = %q, want %q", entry.BoardName, "My Board")
	}
	if entry.ListName != "To Do" {
		t.Errorf("ListName = %q, want %q", entry.ListName, "To Do")
	}
	if entry.Due != "2024-06-15" {
		t.Errorf("Due = %q, want %q", entry.Due, "2024-06-15")
	}
	if len(entry.Labels) != 2 {
		t.Errorf("len(Labels) = %d, want 2 (empty names filtered)", len(entry.Labels))
	}
}

func TestBuildSearchCardEntryNilBoardList(t *testing.T) {
	card := trello.Card{ID: "c1", Name: "Orphan"}
	entry := buildSearchCardEntry(card)

	if entry.BoardName != "" {
		t.Errorf("BoardName = %q, want empty", entry.BoardName)
	}
	if entry.ListName != "" {
		t.Errorf("ListName = %q, want empty", entry.ListName)
	}
	if entry.Due != "" {
		t.Errorf("Due = %q, want empty", entry.Due)
	}
}

func TestSearch(t *testing.T) {
	searchResult := &trello.SearchResult{
		Cards: []trello.Card{
			{ID: "c1", Name: "Found Card", IDBoard: "b1", ShortURL: "https://trello.com/c/c1"},
		},
		Boards: []trello.Board{
			{ID: "b1", Name: "Found Board", URL: "https://trello.com/b/b1"},
		},
	}

	tests := []struct {
		name    string
		args    map[string]any
		cfg     *config.Config
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"query": "test"},
			cfg:  &config.Config{},
		},
		{
			name: "with board_id",
			args: map[string]any{"query": "test", "board_id": "b1"},
			cfg:  &config.Config{},
		},
		{
			name: "with default board",
			args: map[string]any{"query": "test"},
			cfg:  &config.Config{DefaultBoard: "b1"},
		},
		{
			name:    "missing query",
			args:    map[string]any{},
			cfg:     &config.Config{},
			wantErr: "missing required argument: query",
		},
		{
			name:    "board not allowed",
			args:    map[string]any{"query": "test", "board_id": "b2"},
			cfg:     &config.Config{AllowedBoards: []string{"b1"}},
			wantErr: "not in the allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{
				searchFn: func(_ context.Context, _ string, _ []string, _ int) (*trello.SearchResult, error) {
					return searchResult, nil
				},
			}
			h := testHandler(mc, tt.cfg)

			result, err := h.Search(testCtx(), testRequest(tt.args))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			text := resultText(result)

			if tt.wantErr != "" {
				if !result.IsError {
					t.Fatal("expected error result")
				}
				if !strings.Contains(text, tt.wantErr) {
					t.Errorf("error %q missing %q", text, tt.wantErr)
				}
				return
			}

			if result.IsError {
				t.Fatalf("unexpected error: %s", text)
			}

			var resp searchResponse
			resultJSON(t, result, &resp)

			if resp.Query == "" {
				t.Error("query should be set")
			}
		})
	}
}

func TestSearchAllowedBoardsFilter(t *testing.T) {
	searchResult := &trello.SearchResult{
		Cards: []trello.Card{
			{ID: "c1", Name: "Allowed", IDBoard: "b1"},
			{ID: "c2", Name: "Not Allowed", IDBoard: "b2"},
		},
		Boards: []trello.Board{
			{ID: "b1", Name: "Allowed Board"},
			{ID: "b2", Name: "Not Allowed Board"},
		},
	}

	mc := &mockClient{
		searchFn: func(_ context.Context, _ string, _ []string, _ int) (*trello.SearchResult, error) {
			return searchResult, nil
		},
	}
	h := testHandler(mc, &config.Config{AllowedBoards: []string{"b1"}})

	result, err := h.Search(testCtx(), testRequest(map[string]any{"query": "test"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp searchResponse
	resultJSON(t, result, &resp)

	if resp.CardCount != 1 {
		t.Errorf("card_count = %d, want 1", resp.CardCount)
	}
	if resp.BoardCount != 1 {
		t.Errorf("board_count = %d, want 1", resp.BoardCount)
	}
}

func TestSearchLogging(t *testing.T) {
	mc := &mockClient{
		searchFn: func(_ context.Context, _ string, _ []string, _ int) (*trello.SearchResult, error) {
			return &trello.SearchResult{
				Cards: []trello.Card{
					{ID: "c1", Name: "Found Card", IDBoard: "b1"},
				},
				Boards: []trello.Board{
					{ID: "b1", Name: "Found Board"},
				},
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.Search(ctx, testRequest(map[string]any{"query": "test"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventSearchRun) {
		t.Fatal("expected INFO entry for eventSearchRun")
	}
	entry := rec.Entry("INFO", eventSearchRun)
	if !entry.HasField("query") {
		t.Error("expected field 'query' on eventSearchRun entry")
	}
	if !entry.HasField("card_count") {
		t.Error("expected field 'card_count' on eventSearchRun entry")
	}
	if !entry.HasField("board_count") {
		t.Error("expected field 'board_count' on eventSearchRun entry")
	}
}

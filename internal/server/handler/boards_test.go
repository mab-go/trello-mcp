package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestBoards(t *testing.T) {
	boards := []trello.Board{
		{ID: "b1", Name: "Project Alpha", URL: "https://trello.com/b/b1", DateLastActivity: "2024-06-15T10:00:00.000Z"},
		{ID: "b2", Name: "Project Beta", URL: "https://trello.com/b/b2", DateLastActivity: "2024-07-01T08:00:00.000Z"},
		{ID: "b3", Name: "Archive Board", URL: "https://trello.com/b/b3"},
	}

	tests := []struct {
		name       string
		args       map[string]any
		cfg        *config.Config
		wantCount  int
		wantSubstr string
		wantErr    string
	}{
		{
			name:      "all boards",
			args:      map[string]any{},
			cfg:       &config.Config{},
			wantCount: 3,
		},
		{
			name:      "filter by query",
			args:      map[string]any{"query": "alpha"},
			cfg:       &config.Config{},
			wantCount: 1,
		},
		{
			name:      "query case insensitive",
			args:      map[string]any{"query": "BETA"},
			cfg:       &config.Config{},
			wantCount: 1,
		},
		{
			name:      "query no match",
			args:      map[string]any{"query": "nonexistent"},
			cfg:       &config.Config{},
			wantCount: 0,
		},
		{
			name:       "allowed boards filter",
			args:       map[string]any{},
			cfg:        &config.Config{AllowedBoards: []string{"b1", "b3"}},
			wantCount:  2,
			wantSubstr: "Alpha",
		},
		{
			name:       "date formatting",
			args:       map[string]any{},
			cfg:        &config.Config{},
			wantCount:  3,
			wantSubstr: "2024-06-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{
				getBoardsFn: func(_ context.Context) ([]trello.Board, error) {
					return boards, nil
				},
			}
			h := testHandler(mc, tt.cfg)
			ctx := testCtx()

			result, err := h.Boards(ctx, testRequest(tt.args))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			text := resultText(result)

			if tt.wantErr != "" {
				if !result.IsError {
					t.Fatalf("expected error result")
				}
				if !strings.Contains(text, tt.wantErr) {
					t.Fatalf("error %q missing expected substring %q", text, tt.wantErr)
				}
				return
			}

			if result.IsError {
				t.Fatalf("unexpected error result: %s", text)
			}

			var resp boardsResponse
			resultJSON(t, result, &resp)

			if resp.Count != tt.wantCount {
				t.Errorf("count = %d, want %d", resp.Count, tt.wantCount)
			}
			if len(resp.Boards) != tt.wantCount {
				t.Errorf("len(boards) = %d, want %d", len(resp.Boards), tt.wantCount)
			}
			if tt.wantSubstr != "" && !strings.Contains(text, tt.wantSubstr) {
				t.Errorf("result missing substring %q", tt.wantSubstr)
			}
		})
	}
}

func TestBoardsAPIError(t *testing.T) {
	mc := &mockClient{
		getBoardsFn: func(_ context.Context) ([]trello.Board, error) {
			return nil, &trello.APIError{StatusCode: 401, Body: "unauthorized"}
		},
	}
	h := testHandler(mc, &config.Config{})

	result, err := h.Boards(testCtx(), testRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	if !strings.Contains(resultText(result), "Authentication failed") {
		t.Errorf("expected auth error message")
	}
}

func TestBoardsLogging(t *testing.T) {
	mc := &mockClient{
		getBoardsFn: func(_ context.Context) ([]trello.Board, error) {
			return []trello.Board{
				{ID: "b1", Name: "Alpha"},
				{ID: "b2", Name: "Beta"},
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.Boards(ctx, testRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventBoardList) {
		t.Fatal("expected INFO entry for eventBoardList")
	}
	entry := rec.Entry("INFO", eventBoardList)
	if !entry.HasField("count") {
		t.Error("expected field 'count' on eventBoardList entry")
	}
}

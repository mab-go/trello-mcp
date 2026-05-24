package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestLists(t *testing.T) {
	lists := []trello.List{
		{ID: "l1", Name: "To Do", Pos: 1024},
		{ID: "l2", Name: "In Progress", Pos: 2048},
		{ID: "l3", Name: "Done", Pos: 4096},
	}

	tests := []struct {
		name      string
		args      map[string]any
		cfg       *config.Config
		wantCount int
		wantErr   string
	}{
		{
			name:      "success with board_id",
			args:      map[string]any{"board_id": "board1"},
			cfg:       &config.Config{},
			wantCount: 3,
		},
		{
			name:      "success with default board",
			args:      map[string]any{},
			cfg:       &config.Config{DefaultBoard: "board1"},
			wantCount: 3,
		},
		{
			name:    "missing board_id",
			args:    map[string]any{},
			cfg:     &config.Config{},
			wantErr: "missing board_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return lists, nil
				},
			}
			h := testHandler(mc, tt.cfg)

			result, err := h.Lists(testCtx(), testRequest(tt.args))
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

			var resp listsResponse
			resultJSON(t, result, &resp)

			if resp.Count != tt.wantCount {
				t.Errorf("count = %d, want %d", resp.Count, tt.wantCount)
			}
			if resp.BoardID == "" {
				t.Error("board_id should be set")
			}
		})
	}
}

func TestListsLogging(t *testing.T) {
	mc := &mockClient{
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return []trello.List{
				{ID: "l1", Name: "To Do"},
				{ID: "l2", Name: "Done"},
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.Lists(ctx, testRequest(map[string]any{"board_id": "b1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventListList) {
		t.Fatal("expected INFO entry for eventListList")
	}
	entry := rec.Entry("INFO", eventListList)
	if !entry.HasField("board_id") {
		t.Error("expected field 'board_id' on eventListList entry")
	}
	if !entry.HasField("count") {
		t.Error("expected field 'count' on eventListList entry")
	}
}

package handler

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestCreateList(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		cfg     *config.Config
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"board_id": "b1", "name": "New List"},
			cfg:  &config.Config{},
			client: &mockClient{
				createListFn: func(_ context.Context, _ string, params url.Values) (*trello.List, error) {
					if params.Get("pos") != "bottom" {
						t.Errorf("pos = %q, want %q", params.Get("pos"), "bottom")
					}
					return &trello.List{ID: "l1", Name: params.Get("name"), Pos: 4096}, nil
				},
			},
		},
		{
			name: "success with position",
			args: map[string]any{"board_id": "b1", "name": "Top List", "position": "top"},
			cfg:  &config.Config{},
			client: &mockClient{
				createListFn: func(_ context.Context, _ string, params url.Values) (*trello.List, error) {
					if params.Get("pos") != "top" {
						t.Errorf("pos = %q, want %q", params.Get("pos"), "top")
					}
					return &trello.List{ID: "l1", Name: params.Get("name"), Pos: 1024}, nil
				},
			},
		},
		{
			name:    "missing board_id",
			args:    map[string]any{"name": "New List"},
			cfg:     &config.Config{},
			client:  &mockClient{},
			wantErr: "missing board_id",
		},
		{
			name:    "missing name",
			args:    map[string]any{"board_id": "b1"},
			cfg:     &config.Config{},
			client:  &mockClient{},
			wantErr: "missing required argument: name",
		},
		{
			name: "with default board",
			args: map[string]any{"name": "New List"},
			cfg:  &config.Config{DefaultBoard: "b1"},
			client: &mockClient{
				createListFn: func(_ context.Context, boardID string, _ url.Values) (*trello.List, error) {
					if boardID != "b1" {
						t.Errorf("boardID = %q, want %q", boardID, "b1")
					}
					return &trello.List{ID: "l1", Name: "New List", Pos: 4096}, nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, tt.cfg)

			result, err := h.CreateList(testCtx(), testRequest(tt.args))
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

			var resp createListResponse
			resultJSON(t, result, &resp)
			if resp.ListID != "l1" {
				t.Errorf("ListID = %q, want %q", resp.ListID, "l1")
			}
			if resp.BoardID == "" {
				t.Error("BoardID should be set")
			}
		})
	}
}

func TestCreateListLogging(t *testing.T) {
	mc := &mockClient{
		createListFn: func(_ context.Context, _ string, params url.Values) (*trello.List, error) {
			return &trello.List{ID: "l1", Name: params.Get("name"), Pos: 4096}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.CreateList(ctx, testRequest(map[string]any{"board_id": "b1", "name": "New List"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventListCreate) {
		t.Fatal("expected INFO entry for eventListCreate")
	}
	entry := rec.Entry("INFO", eventListCreate)
	if !entry.HasField("board_id") {
		t.Error("expected field 'board_id' on eventListCreate entry")
	}
	if !entry.HasField("list_id") {
		t.Error("expected field 'list_id' on eventListCreate entry")
	}
	if !entry.HasField("name") {
		t.Error("expected field 'name' on eventListCreate entry")
	}
}

package handler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestResolveBoardID(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		cfg       *config.Config
		wantID    string
		wantErr   bool
		errSubstr string
	}{
		{
			name:   "explicit board_id in args",
			args:   map[string]any{"board_id": "board123"},
			cfg:    &config.Config{},
			wantID: "board123",
		},
		{
			name:   "falls back to default_board",
			args:   map[string]any{},
			cfg:    &config.Config{DefaultBoard: "defaultABC"},
			wantID: "defaultABC",
		},
		{
			name:   "explicit overrides default",
			args:   map[string]any{"board_id": "explicit1"},
			cfg:    &config.Config{DefaultBoard: "default2"},
			wantID: "explicit1",
		},
		{
			name:      "no board_id and no default",
			args:      map[string]any{},
			cfg:       &config.Config{},
			wantErr:   true,
			errSubstr: "missing board_id argument",
		},
		{
			name:      "empty string board_id and no default",
			args:      map[string]any{"board_id": ""},
			cfg:       &config.Config{},
			wantErr:   true,
			errSubstr: "missing board_id argument",
		},
		{
			name:   "allowed board passes",
			args:   map[string]any{"board_id": "boardA"},
			cfg:    &config.Config{AllowedBoards: []string{"boardA", "boardB"}},
			wantID: "boardA",
		},
		{
			name:      "disallowed board fails",
			args:      map[string]any{"board_id": "boardX"},
			cfg:       &config.Config{AllowedBoards: []string{"boardA", "boardB"}},
			wantErr:   true,
			errSubstr: "not in the allowed list",
		},
		{
			name:      "default board not in allowed list",
			args:      map[string]any{},
			cfg:       &config.Config{DefaultBoard: "boardX", AllowedBoards: []string{"boardA"}},
			wantErr:   true,
			errSubstr: "not in the allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(&mockClient{}, tt.cfg)
			id, errResult := h.resolveBoardID(tt.args)

			if tt.wantErr {
				if errResult == nil {
					t.Fatal("expected error result, got nil")
				}
				if !errResult.IsError {
					t.Fatal("expected IsError to be true")
				}
				text := resultText(errResult)
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error text %q does not contain %q", text, tt.errSubstr)
				}
				return
			}

			if errResult != nil {
				t.Fatalf("unexpected error result: %s", resultText(errResult))
			}
			if id != tt.wantID {
				t.Errorf("got board ID %q, want %q", id, tt.wantID)
			}
		})
	}
}

func TestCheckAllowedBoard(t *testing.T) {
	tests := []struct {
		name      string
		boardID   string
		cfg       *config.Config
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "empty allowed list permits any board",
			boardID: "anyBoard",
			cfg:     &config.Config{},
			wantErr: false,
		},
		{
			name:    "board in allowed list",
			boardID: "boardA",
			cfg:     &config.Config{AllowedBoards: []string{"boardA", "boardB"}},
			wantErr: false,
		},
		{
			name:      "board not in allowed list",
			boardID:   "boardX",
			cfg:       &config.Config{AllowedBoards: []string{"boardA", "boardB"}},
			wantErr:   true,
			errSubstr: "not in the allowed list",
		},
		{
			name:    "nil allowed boards permits any board",
			boardID: "anyBoard",
			cfg:     &config.Config{AllowedBoards: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(&mockClient{}, tt.cfg)
			errResult := h.checkAllowedBoard(tt.boardID)

			if tt.wantErr {
				if errResult == nil {
					t.Fatal("expected error result, got nil")
				}
				if !errResult.IsError {
					t.Fatal("expected IsError to be true")
				}
				text := resultText(errResult)
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error text %q does not contain %q", text, tt.errSubstr)
				}
				return
			}

			if errResult != nil {
				t.Fatalf("unexpected error result: %s", resultText(errResult))
			}
		})
	}
}

func TestResolveListByName(t *testing.T) {
	lists := []trello.List{
		{ID: "list1", Name: "To Do"},
		{ID: "list2", Name: "In Progress"},
		{ID: "list3", Name: "Done"},
	}

	tests := []struct {
		name      string
		lists     []trello.List
		listName  string
		wantID    string
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "exact match",
			lists:    lists,
			listName: "To Do",
			wantID:   "list1",
		},
		{
			name:     "case-insensitive match lowercase",
			lists:    lists,
			listName: "to do",
			wantID:   "list1",
		},
		{
			name:     "case-insensitive match uppercase",
			lists:    lists,
			listName: "IN PROGRESS",
			wantID:   "list2",
		},
		{
			name:     "case-insensitive match mixed case",
			lists:    lists,
			listName: "dOnE",
			wantID:   "list3",
		},
		{
			name:      "no match lists available names",
			lists:     lists,
			listName:  "Backlog",
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name:      "no match error includes available list names",
			lists:     lists,
			listName:  "Backlog",
			wantErr:   true,
			errSubstr: "To Do",
		},
		{
			name:      "empty list name",
			lists:     lists,
			listName:  "",
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name:      "empty lists slice",
			lists:     []trello.List{},
			listName:  "To Do",
			wantErr:   true,
			errSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(&mockClient{}, nil)
			id, errResult := h.resolveListByName(tt.lists, tt.listName)

			if tt.wantErr {
				if errResult == nil {
					t.Fatal("expected error result, got nil")
				}
				if !errResult.IsError {
					t.Fatal("expected IsError to be true")
				}
				text := resultText(errResult)
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error text %q does not contain %q", text, tt.errSubstr)
				}
				return
			}

			if errResult != nil {
				t.Fatalf("unexpected error result: %s", resultText(errResult))
			}
			if id != tt.wantID {
				t.Errorf("got list ID %q, want %q", id, tt.wantID)
			}
		})
	}
}

func TestListDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		listID   string
		client   *mockClient
		wantName string
	}{
		{
			name:   "returns list name on success",
			listID: "list1",
			client: &mockClient{
				getListNameFn: func(_ context.Context, _ string) (string, error) {
					return "My List", nil
				},
			},
			wantName: "My List",
		},
		{
			name:   "returns empty string on error",
			listID: "list1",
			client: &mockClient{
				getListNameFn: func(_ context.Context, _ string) (string, error) {
					return "", fmt.Errorf("network error")
				},
			},
			wantName: "",
		},
		{
			name:     "returns empty string for empty listID",
			listID:   "",
			client:   &mockClient{},
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, nil)
			ctx := testCtx()
			got := h.listDisplayName(ctx, tt.listID)
			if got != tt.wantName {
				t.Errorf("got %q, want %q", got, tt.wantName)
			}
		})
	}
}

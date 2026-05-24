package handler

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestBuildUpdateParams(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		wantUpdate bool
		wantKeys   []string
	}{
		{
			name:       "name only",
			args:       map[string]any{"name": "New Name"},
			wantUpdate: true,
			wantKeys:   []string{"name"},
		},
		{
			name:       "description",
			args:       map[string]any{"description": "Hello"},
			wantUpdate: true,
			wantKeys:   []string{"desc"},
		},
		{
			name:       "due date",
			args:       map[string]any{"due": "2024-06-15"},
			wantUpdate: true,
			wantKeys:   []string{"due"},
		},
		{
			name:       "due_complete true",
			args:       map[string]any{"due_complete": true},
			wantUpdate: true,
			wantKeys:   []string{"dueComplete"},
		},
		{
			name:       "due_complete false",
			args:       map[string]any{"due_complete": false},
			wantUpdate: true,
			wantKeys:   []string{"dueComplete"},
		},
		{
			name:       "position",
			args:       map[string]any{"position": "top"},
			wantUpdate: true,
			wantKeys:   []string{"pos"},
		},
		{
			name:       "no fields",
			args:       map[string]any{},
			wantUpdate: false,
		},
		{
			name:       "multiple fields",
			args:       map[string]any{"name": "New", "description": "Desc", "due": "2024-01-01"},
			wantUpdate: true,
			wantKeys:   []string{"name", "desc", "due"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, hasUpdate := buildUpdateParams(tt.args)
			if hasUpdate != tt.wantUpdate {
				t.Errorf("hasUpdate = %v, want %v", hasUpdate, tt.wantUpdate)
			}
			for _, key := range tt.wantKeys {
				if params.Get(key) == "" {
					t.Errorf("expected param %q to be set", key)
				}
			}
		})
	}
}

func TestBuildUpdateParamsDueComplete(t *testing.T) {
	paramsTrue, _ := buildUpdateParams(map[string]any{"due_complete": true})
	if paramsTrue.Get("dueComplete") != "true" {
		t.Errorf("dueComplete = %q, want %q", paramsTrue.Get("dueComplete"), "true")
	}

	paramsFalse, _ := buildUpdateParams(map[string]any{"due_complete": false})
	if paramsFalse.Get("dueComplete") != "false" {
		t.Errorf("dueComplete = %q, want %q", paramsFalse.Get("dueComplete"), "false")
	}
}

func TestBuildUpdateResponse(t *testing.T) {
	due := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	card := &trello.Card{
		ID:       "c1",
		Name:     "Updated Card",
		Desc:     "New description",
		Due:      &due,
		IDList:   "l1",
		ShortURL: "https://trello.com/c/c1",
	}

	args := map[string]any{
		"description": "New description",
		"due":         "2024-06-15",
	}

	resp := buildUpdateResponse(args, card, "To Do")

	if resp.CardID != "c1" {
		t.Errorf("CardID = %q, want %q", resp.CardID, "c1")
	}
	if resp.Description == nil {
		t.Fatal("Description should be set")
	}
	if *resp.Description != "New description" {
		t.Errorf("Description = %q, want %q", *resp.Description, "New description")
	}
	if resp.Due == nil {
		t.Fatal("Due should be set")
	}
	if *resp.Due != "2024-06-15" {
		t.Errorf("Due = %q, want %q", *resp.Due, "2024-06-15")
	}
	if resp.Position != nil {
		t.Error("Position should be nil (not in args)")
	}
}

func TestUpdateCard(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		cfg     *config.Config
		client  *mockClient
		wantErr string
	}{
		{
			name: "success with name",
			args: map[string]any{"card_id": "c1", "name": "New Name"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1", IDList: "l1"}, nil
				},
				updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
					return &trello.Card{ID: "c1", Name: "New Name", IDList: "l1", ShortURL: "https://trello.com/c/c1"}, nil
				},
				getListNameFn: func(_ context.Context, _ string) (string, error) {
					return "To Do", nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{"name": "New Name"},
			cfg:     &config.Config{},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "no update fields",
			args:    map[string]any{"card_id": "c1"},
			cfg:     &config.Config{},
			client:  &mockClient{},
			wantErr: "at least one field to update is required",
		},
		{
			name: "board not allowed",
			args: map[string]any{"card_id": "c1", "name": "New"},
			cfg:  &config.Config{AllowedBoards: []string{"other"}},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
			},
			wantErr: "not in the allowed list",
		},
		{
			name: "with list_name",
			args: map[string]any{"card_id": "c1", "list_name": "Done"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1", IDList: "l1"}, nil
				},
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return []trello.List{{ID: "l1", Name: "To Do"}, {ID: "l2", Name: "Done"}}, nil
				},
				updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
					return &trello.Card{ID: "c1", Name: "Card", IDList: "l2", ShortURL: "https://trello.com/c/c1"}, nil
				},
				getListNameFn: func(_ context.Context, _ string) (string, error) {
					return "Done", nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, tt.cfg)

			result, err := h.UpdateCard(testCtx(), testRequest(tt.args))
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

			var resp updateCardResponse
			resultJSON(t, result, &resp)
			if resp.CardID != "c1" {
				t.Errorf("CardID = %q, want %q", resp.CardID, "c1")
			}
		})
	}
}

func TestUpdateCardLogging(t *testing.T) {
	mc := &mockClient{
		getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1", IDList: "l1"}, nil
		},
		updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
			return &trello.Card{ID: "c1", Name: "New Name", IDList: "l1", ShortURL: "https://trello.com/c/c1"}, nil
		},
		getListNameFn: func(_ context.Context, _ string) (string, error) {
			return "To Do", nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.UpdateCard(ctx, testRequest(map[string]any{"card_id": "c1", "name": "New Name"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCardUpdate) {
		t.Fatal("expected INFO entry for eventCardUpdate")
	}
	entry := rec.Entry("INFO", eventCardUpdate)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCardUpdate entry")
	}
	if !entry.HasField("name") {
		t.Error("expected field 'name' on eventCardUpdate entry")
	}
}

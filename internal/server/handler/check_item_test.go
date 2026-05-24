package handler

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestCheckItem(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "mark complete",
			args: map[string]any{"card_id": "c1", "item_id": "ci1", "complete": true},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				updateCheckItemFn: func(_ context.Context, _, _ string, params url.Values) (*trello.CheckItem, error) {
					if params.Get("state") != "complete" {
						t.Errorf("state = %q, want %q", params.Get("state"), "complete")
					}
					return &trello.CheckItem{ID: "ci1", Name: "Task", State: "complete"}, nil
				},
			},
		},
		{
			name: "mark incomplete",
			args: map[string]any{"card_id": "c1", "item_id": "ci1", "complete": false},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				updateCheckItemFn: func(_ context.Context, _, _ string, params url.Values) (*trello.CheckItem, error) {
					if params.Get("state") != "incomplete" {
						t.Errorf("state = %q, want %q", params.Get("state"), "incomplete")
					}
					return &trello.CheckItem{ID: "ci1", Name: "Task", State: "incomplete"}, nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{"item_id": "ci1", "complete": true},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "missing item_id",
			args:    map[string]any{"card_id": "c1", "complete": true},
			client:  &mockClient{},
			wantErr: "missing required argument: item_id",
		},
		{
			name:    "missing complete",
			args:    map[string]any{"card_id": "c1", "item_id": "ci1"},
			client:  &mockClient{},
			wantErr: "missing required argument: complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, &config.Config{})

			result, err := h.CheckItem(testCtx(), testRequest(tt.args))
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

			var resp checkItemResponse
			resultJSON(t, result, &resp)
			if resp.CardID != "c1" {
				t.Errorf("CardID = %q, want %q", resp.CardID, "c1")
			}
		})
	}
}

func TestAddCheckItem(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"checklist_id": "cl1", "name": "New Item"},
			client: &mockClient{
				getChecklistBoardFn: func(_ context.Context, _ string) (*trello.Board, error) {
					return &trello.Board{ID: "b1"}, nil
				},
				createCheckItemFn: func(_ context.Context, _ string, params url.Values) (*trello.CheckItem, error) {
					return &trello.CheckItem{ID: "ci1", Name: params.Get("name"), State: "incomplete"}, nil
				},
			},
		},
		{
			name: "success with checked",
			args: map[string]any{"checklist_id": "cl1", "name": "Done Item", "checked": true},
			client: &mockClient{
				getChecklistBoardFn: func(_ context.Context, _ string) (*trello.Board, error) {
					return &trello.Board{ID: "b1"}, nil
				},
				createCheckItemFn: func(_ context.Context, _ string, params url.Values) (*trello.CheckItem, error) {
					if params.Get("checked") != "true" {
						t.Errorf("checked = %q, want %q", params.Get("checked"), "true")
					}
					return &trello.CheckItem{ID: "ci1", Name: params.Get("name"), State: "complete"}, nil
				},
			},
		},
		{
			name:    "missing checklist_id",
			args:    map[string]any{"name": "Item"},
			client:  &mockClient{},
			wantErr: "missing required argument: checklist_id",
		},
		{
			name:    "missing name",
			args:    map[string]any{"checklist_id": "cl1"},
			client:  &mockClient{},
			wantErr: "missing required argument: name",
		},
		{
			name: "board not allowed",
			args: map[string]any{"checklist_id": "cl1", "name": "Item"},
			client: &mockClient{
				getChecklistBoardFn: func(_ context.Context, _ string) (*trello.Board, error) {
					return &trello.Board{ID: "b1"}, nil
				},
			},
			wantErr: "not in the allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			if tt.wantErr == "not in the allowed list" {
				cfg.AllowedBoards = []string{"other"}
			}
			h := testHandler(tt.client, cfg)

			result, err := h.AddCheckItem(testCtx(), testRequest(tt.args))
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

			var resp addCheckItemResponse
			resultJSON(t, result, &resp)
			if resp.ChecklistID != "cl1" {
				t.Errorf("ChecklistID = %q, want %q", resp.ChecklistID, "cl1")
			}
		})
	}
}

func TestCheckItemLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		updateCheckItemFn: func(_ context.Context, _, _ string, _ url.Values) (*trello.CheckItem, error) {
			return &trello.CheckItem{ID: "ci1", Name: "Task", State: "complete"}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.CheckItem(ctx, testRequest(map[string]any{
		"card_id":  "c1",
		"item_id":  "ci1",
		"complete": true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCheckItemUpdate) {
		t.Fatal("expected INFO entry for eventCheckItemUpdate")
	}
	entry := rec.Entry("INFO", eventCheckItemUpdate)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCheckItemUpdate entry")
	}
	if !entry.HasField("item_id") {
		t.Error("expected field 'item_id' on eventCheckItemUpdate entry")
	}
	if !entry.HasField("complete") {
		t.Error("expected field 'complete' on eventCheckItemUpdate entry")
	}
}

func TestAddCheckItemLogging(t *testing.T) {
	mc := &mockClient{
		getChecklistBoardFn: func(_ context.Context, _ string) (*trello.Board, error) {
			return &trello.Board{ID: "b1"}, nil
		},
		createCheckItemFn: func(_ context.Context, _ string, params url.Values) (*trello.CheckItem, error) {
			return &trello.CheckItem{ID: "ci1", Name: params.Get("name"), State: "incomplete"}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.AddCheckItem(ctx, testRequest(map[string]any{
		"checklist_id": "cl1",
		"name":         "New Item",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCheckItemAdd) {
		t.Fatal("expected INFO entry for eventCheckItemAdd")
	}
	entry := rec.Entry("INFO", eventCheckItemAdd)
	if !entry.HasField("checklist_id") {
		t.Error("expected field 'checklist_id' on eventCheckItemAdd entry")
	}
	if !entry.HasField("item_id") {
		t.Error("expected field 'item_id' on eventCheckItemAdd entry")
	}
}

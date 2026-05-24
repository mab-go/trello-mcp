package handler

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestBuildChecklistEntries(t *testing.T) {
	checklists := []trello.Checklist{
		{
			ID:   "cl1",
			Name: "Tasks",
			CheckItems: []trello.CheckItem{
				{ID: "ci1", Name: "Item 1", State: "complete"},
				{ID: "ci2", Name: "Item 2", State: "incomplete"},
			},
		},
		{
			ID:   "cl2",
			Name: "Empty Checklist",
		},
	}

	entries := buildChecklistEntries(checklists)

	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].ChecklistID != "cl1" {
		t.Errorf("ChecklistID = %q, want %q", entries[0].ChecklistID, "cl1")
	}
	if len(entries[0].Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(entries[0].Items))
	}
	if !entries[0].Items[0].Complete {
		t.Error("Item 1 should be complete")
	}
	if entries[0].Items[1].Complete {
		t.Error("Item 2 should not be complete")
	}
	if len(entries[1].Items) != 0 {
		t.Errorf("empty checklist items len = %d, want 0", len(entries[1].Items))
	}
}

func TestChecklists(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"card_id": "c1"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				getCardChecklistsFn: func(_ context.Context, _ string) ([]trello.Checklist, error) {
					return []trello.Checklist{
						{ID: "cl1", Name: "Tasks", CheckItems: []trello.CheckItem{
							{ID: "ci1", Name: "Do thing", State: "incomplete"},
						}},
					}, nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, &config.Config{})

			result, err := h.Checklists(testCtx(), testRequest(tt.args))
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

			var resp checklistsResponse
			resultJSON(t, result, &resp)
			if resp.Count != 1 {
				t.Errorf("Count = %d, want 1", resp.Count)
			}
		})
	}
}

func TestAddChecklist(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "success without items",
			args: map[string]any{"card_id": "c1", "name": "My Checklist"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				createChecklistFn: func(_ context.Context, _, name string) (*trello.Checklist, error) {
					return &trello.Checklist{ID: "cl1", Name: name}, nil
				},
			},
		},
		{
			name: "success with items",
			args: map[string]any{
				"card_id": "c1",
				"name":    "My Checklist",
				"items":   []any{"Item 1", "Item 2"},
			},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				createChecklistFn: func(_ context.Context, _, name string) (*trello.Checklist, error) {
					return &trello.Checklist{ID: "cl1", Name: name}, nil
				},
				createCheckItemFn: func(_ context.Context, _ string, params url.Values) (*trello.CheckItem, error) {
					return &trello.CheckItem{ID: "ci1", Name: params.Get("name")}, nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{"name": "CL"},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "missing name",
			args:    map[string]any{"card_id": "c1"},
			client:  &mockClient{},
			wantErr: "missing required argument: name",
		},
		{
			name:    "invalid items type",
			args:    map[string]any{"card_id": "c1", "name": "CL", "items": "not array"},
			client:  &mockClient{},
			wantErr: "items must be an array of strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, &config.Config{})

			result, err := h.AddChecklist(testCtx(), testRequest(tt.args))
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

			var resp addChecklistResponse
			resultJSON(t, result, &resp)
			if resp.ChecklistID != "cl1" {
				t.Errorf("ChecklistID = %q, want %q", resp.ChecklistID, "cl1")
			}
		})
	}
}

func TestChecklistsLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		getCardChecklistsFn: func(_ context.Context, _ string) ([]trello.Checklist, error) {
			return []trello.Checklist{
				{ID: "cl1", Name: "Tasks", CheckItems: []trello.CheckItem{
					{ID: "ci1", Name: "Do thing", State: "incomplete"},
				}},
				{ID: "cl2", Name: "Other", CheckItems: []trello.CheckItem{}},
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.Checklists(ctx, testRequest(map[string]any{"card_id": "c1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventChecklistList) {
		t.Fatal("expected INFO entry for eventChecklistList")
	}
	entry := rec.Entry("INFO", eventChecklistList)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventChecklistList entry")
	}
	if !entry.HasField("count") {
		t.Error("expected field 'count' on eventChecklistList entry")
	}
}

func TestAddChecklistLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		createChecklistFn: func(_ context.Context, _, name string) (*trello.Checklist, error) {
			return &trello.Checklist{ID: "cl1", Name: name}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.AddChecklist(ctx, testRequest(map[string]any{"card_id": "c1", "name": "My Checklist"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventChecklistAdd) {
		t.Fatal("expected INFO entry for eventChecklistAdd")
	}
	entry := rec.Entry("INFO", eventChecklistAdd)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventChecklistAdd entry")
	}
	if !entry.HasField("checklist_id") {
		t.Error("expected field 'checklist_id' on eventChecklistAdd entry")
	}
	if !entry.HasField("name") {
		t.Error("expected field 'name' on eventChecklistAdd entry")
	}
}

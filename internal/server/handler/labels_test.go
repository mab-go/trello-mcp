package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestLabels(t *testing.T) {
	boardLabels := []trello.Label{
		{ID: "l1", Name: "Bug", Color: "red"},
		{ID: "l2", Name: "", Color: "blue"},
		{ID: "l3", Name: "Feature", Color: "green"},
	}

	mc := &mockClient{
		getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
			return boardLabels, nil
		},
	}
	h := testHandler(mc, &config.Config{DefaultBoard: "b1"})

	result, err := h.Labels(testCtx(), testRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(result))
	}

	var resp labelsResponse
	resultJSON(t, result, &resp)

	if resp.Count != 2 {
		t.Errorf("count = %d, want 2 (empty names filtered)", resp.Count)
	}
	if len(resp.Labels) != 2 {
		t.Errorf("len(labels) = %d, want 2", len(resp.Labels))
	}
}

func TestLabelsMissingBoard(t *testing.T) {
	h := testHandler(&mockClient{}, &config.Config{})

	result, err := h.Labels(testCtx(), testRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	if !strings.Contains(resultText(result), "missing board_id") {
		t.Error("expected missing board_id error")
	}
}

func TestAddLabel(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"card_id": "c1", "label": "Bug"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
					return []trello.Label{{ID: "l1", Name: "Bug", Color: "red"}}, nil
				},
				addCardLabelFn: func(_ context.Context, _, _ string) error {
					return nil
				},
			},
		},
		{
			name: "case insensitive match",
			args: map[string]any{"card_id": "c1", "label": "bug"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
					return []trello.Label{{ID: "l1", Name: "Bug", Color: "red"}}, nil
				},
				addCardLabelFn: func(_ context.Context, _, _ string) error {
					return nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{"label": "Bug"},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "missing label",
			args:    map[string]any{"card_id": "c1"},
			client:  &mockClient{},
			wantErr: "missing required argument: label",
		},
		{
			name: "label not found",
			args: map[string]any{"card_id": "c1", "label": "nonexistent"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
					return []trello.Label{{ID: "l1", Name: "Bug"}}, nil
				},
			},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, &config.Config{})

			result, err := h.AddLabel(testCtx(), testRequest(tt.args))
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

			var resp addLabelResponse
			resultJSON(t, result, &resp)
			if resp.CardID != "c1" {
				t.Errorf("CardID = %q, want %q", resp.CardID, "c1")
			}
		})
	}
}

func TestFindLabelByID(t *testing.T) {
	labels := []trello.Label{
		{ID: "l1", Name: "Bug"},
		{ID: "l2", Name: "Feature"},
	}

	found := findLabelByID(labels, "l2")
	if found.Name != "Feature" {
		t.Errorf("Name = %q, want %q", found.Name, "Feature")
	}

	notFound := findLabelByID(labels, "l99")
	if notFound.ID != "" {
		t.Errorf("expected empty label for missing ID, got %q", notFound.ID)
	}
}

func TestRemoveLabel(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"card_id": "c1", "label": "Bug"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{
						ID: "c1", IDBoard: "b1",
						Labels: []trello.Label{{ID: "l1", Name: "Bug"}},
					}, nil
				},
				removeCardLabelFn: func(_ context.Context, _, _ string) error {
					return nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{"label": "Bug"},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "missing label",
			args:    map[string]any{"card_id": "c1"},
			client:  &mockClient{},
			wantErr: "missing required argument: label",
		},
		{
			name: "label not on card",
			args: map[string]any{"card_id": "c1", "label": "nonexistent"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{
						ID: "c1", IDBoard: "b1",
						Labels: []trello.Label{{ID: "l1", Name: "Bug"}},
					}, nil
				},
			},
			wantErr: "is not on this card",
		},
		{
			name: "label not on card - no labels",
			args: map[string]any{"card_id": "c1", "label": "Bug"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
			},
			wantErr: "has no labels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, &config.Config{})

			result, err := h.RemoveLabel(testCtx(), testRequest(tt.args))
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

			var resp removeLabelResponse
			resultJSON(t, result, &resp)
			if !resp.Removed {
				t.Error("expected Removed=true")
			}
		})
	}
}

func TestMatchCardLabel(t *testing.T) {
	labels := []trello.Label{
		{ID: "l1", Name: "Bug"},
		{ID: "l2", Name: "Feature"},
	}

	tests := []struct {
		name    string
		labels  []trello.Label
		search  string
		wantID  string
		wantErr bool
	}{
		{"exact match", labels, "Bug", "l1", false},
		{"case insensitive", labels, "bug", "l1", false},
		{"not found", labels, "Enhancement", "", true},
		{"empty labels", nil, "Bug", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, name, errResult := matchCardLabel(tt.labels, tt.search)
			if tt.wantErr {
				if errResult == nil {
					t.Fatal("expected error result")
				}
				return
			}
			if errResult != nil {
				t.Fatalf("unexpected error: %s", resultText(errResult))
			}
			if id != tt.wantID {
				t.Errorf("id = %q, want %q", id, tt.wantID)
			}
			if name == "" {
				t.Error("name should not be empty")
			}
		})
	}
}

func TestLabelsLogging(t *testing.T) {
	mc := &mockClient{
		getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
			return []trello.Label{
				{ID: "l1", Name: "Bug", Color: "red"},
				{ID: "l2", Name: "Feature", Color: "green"},
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{DefaultBoard: "b1"})
	ctx, rec := testCtxRecording()

	result, err := h.Labels(ctx, testRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventLabelList) {
		t.Fatal("expected INFO entry for eventLabelList")
	}
	entry := rec.Entry("INFO", eventLabelList)
	if !entry.HasField("board_id") {
		t.Error("expected field 'board_id' on eventLabelList entry")
	}
	if !entry.HasField("count") {
		t.Error("expected field 'count' on eventLabelList entry")
	}
}

func TestAddLabelLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
			return []trello.Label{{ID: "l1", Name: "Bug", Color: "red"}}, nil
		},
		addCardLabelFn: func(_ context.Context, _, _ string) error {
			return nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.AddLabel(ctx, testRequest(map[string]any{"card_id": "c1", "label": "Bug"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventLabelAdd) {
		t.Fatal("expected INFO entry for eventLabelAdd")
	}
	entry := rec.Entry("INFO", eventLabelAdd)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventLabelAdd entry")
	}
	if !entry.HasField("label") {
		t.Error("expected field 'label' on eventLabelAdd entry")
	}
}

func TestRemoveLabelLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{
				ID: "c1", IDBoard: "b1",
				Labels: []trello.Label{{ID: "l1", Name: "Bug"}},
			}, nil
		},
		removeCardLabelFn: func(_ context.Context, _, _ string) error {
			return nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.RemoveLabel(ctx, testRequest(map[string]any{"card_id": "c1", "label": "Bug"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventLabelRemove) {
		t.Fatal("expected INFO entry for eventLabelRemove")
	}
	entry := rec.Entry("INFO", eventLabelRemove)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventLabelRemove entry")
	}
	if !entry.HasField("label") {
		t.Error("expected field 'label' on eventLabelRemove entry")
	}
}

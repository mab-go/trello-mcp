package handler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestValidateListArgs(t *testing.T) {
	tests := []struct {
		name      string
		listID    string
		listName  string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "both empty returns error",
			listID:    "",
			listName:  "",
			wantErr:   true,
			errSubstr: "one of list_id or list_name is required",
		},
		{
			name:      "both non-empty returns error",
			listID:    "list1",
			listName:  "To Do",
			wantErr:   true,
			errSubstr: "provide either list_id or list_name, not both",
		},
		{
			name:     "only list_id returns nil",
			listID:   "list1",
			listName: "",
			wantErr:  false,
		},
		{
			name:     "only list_name returns nil",
			listID:   "",
			listName: "To Do",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateListArgs(tt.listID, tt.listName)

			if tt.wantErr {
				if result == nil {
					t.Fatal("expected error result, got nil")
				}
				if !result.IsError {
					t.Fatal("expected IsError to be true")
				}
				text := resultText(result)
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error text %q does not contain %q", text, tt.errSubstr)
				}
				return
			}

			if result != nil {
				t.Fatalf("unexpected error result: %s", resultText(result))
			}
		})
	}
}

func TestResolveListIDAndName(t *testing.T) {
	lists := []trello.List{
		{ID: "list1", Name: "To Do"},
		{ID: "list2", Name: "In Progress"},
		{ID: "list3", Name: "Done"},
	}

	tests := []struct {
		name      string
		lists     []trello.List
		listID    string
		listName  string
		wantID    string
		wantName  string
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "resolves list by name",
			lists:    lists,
			listID:   "",
			listName: "To Do",
			wantID:   "list1",
			wantName: "To Do",
		},
		{
			name:     "resolves list by name case-insensitive",
			lists:    lists,
			listID:   "",
			listName: "in progress",
			wantID:   "list2",
			wantName: "in progress",
		},
		{
			name:      "list name not found returns error",
			lists:     lists,
			listID:    "",
			listName:  "Backlog",
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name:     "resolves list by ID found in lists",
			lists:    lists,
			listID:   "list2",
			listName: "",
			wantID:   "list2",
			wantName: "In Progress",
		},
		{
			name:     "list ID not in lists returns ID with empty name",
			lists:    lists,
			listID:   "listX",
			listName: "",
			wantID:   "listX",
			wantName: "",
		},
		{
			name:     "list ID not in empty list returns ID with empty name",
			lists:    []trello.List{},
			listID:   "listX",
			listName: "",
			wantID:   "listX",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(&mockClient{}, nil)
			id, name, errResult := resolveListIDAndName(h, tt.lists, tt.listID, tt.listName)

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
				t.Errorf("got ID %q, want %q", id, tt.wantID)
			}
			if name != tt.wantName {
				t.Errorf("got name %q, want %q", name, tt.wantName)
			}
		})
	}
}

func TestMatchLabel(t *testing.T) {
	boardLabels := []trello.Label{
		{ID: "lbl1", Name: "Bug", Color: "red"},
		{ID: "lbl2", Name: "Feature", Color: "blue"},
		{ID: "lbl3", Name: "", Color: "green"},
		{ID: "lbl4", Name: "Urgent", Color: "orange"},
	}

	tests := []struct {
		name        string
		boardLabels []trello.Label
		labelName   string
		wantID      string
		wantErr     bool
		errSubstr   string
		errAbsent   string // substring that should NOT appear in error
	}{
		{
			name:        "exact match",
			boardLabels: boardLabels,
			labelName:   "Bug",
			wantID:      "lbl1",
		},
		{
			name:        "case-insensitive match",
			boardLabels: boardLabels,
			labelName:   "feature",
			wantID:      "lbl2",
		},
		{
			name:        "case-insensitive match uppercase",
			boardLabels: boardLabels,
			labelName:   "URGENT",
			wantID:      "lbl4",
		},
		{
			name:        "no match returns error listing available labels",
			boardLabels: boardLabels,
			labelName:   "Enhancement",
			wantErr:     true,
			errSubstr:   "not found",
		},
		{
			name:        "no match error includes available names",
			boardLabels: boardLabels,
			labelName:   "Enhancement",
			wantErr:     true,
			errSubstr:   "Bug",
		},
		{
			name:        "filters out empty-named labels from available list",
			boardLabels: boardLabels,
			labelName:   "Missing",
			wantErr:     true,
			errSubstr:   "Urgent",
			errAbsent:   ", ,",
		},
		{
			name:        "empty board labels",
			boardLabels: []trello.Label{},
			labelName:   "Bug",
			wantErr:     true,
			errSubstr:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, errResult := matchLabel(tt.boardLabels, tt.labelName)

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
				if tt.errAbsent != "" && strings.Contains(text, tt.errAbsent) {
					t.Errorf("error text %q should not contain %q", text, tt.errAbsent)
				}
				return
			}

			if errResult != nil {
				t.Fatalf("unexpected error result: %s", resultText(errResult))
			}
			if id != tt.wantID {
				t.Errorf("got label ID %q, want %q", id, tt.wantID)
			}
		})
	}
}

func TestBuildCreateParams(t *testing.T) {
	tests := []struct {
		name     string
		listID   string
		cardName string
		labelIDs []string
		args     map[string]any
		want     map[string]string // expected key-value pairs in url.Values
	}{
		{
			name:     "sets idList and name always",
			listID:   "list1",
			cardName: "My Card",
			args:     map[string]any{},
			want: map[string]string{
				"idList": "list1",
				"name":   "My Card",
				"pos":    "bottom",
			},
		},
		{
			name:     "sets desc if present",
			listID:   "list1",
			cardName: "Card",
			args:     map[string]any{"description": "Some description"},
			want: map[string]string{
				"idList": "list1",
				"name":   "Card",
				"desc":   "Some description",
				"pos":    "bottom",
			},
		},
		{
			name:     "sets due if present",
			listID:   "list1",
			cardName: "Card",
			args:     map[string]any{"due": "2026-01-15"},
			want: map[string]string{
				"idList": "list1",
				"name":   "Card",
				"due":    "2026-01-15",
				"pos":    "bottom",
			},
		},
		{
			name:     "sets idLabels joined by comma",
			listID:   "list1",
			cardName: "Card",
			labelIDs: []string{"lbl1", "lbl2", "lbl3"},
			args:     map[string]any{},
			want: map[string]string{
				"idList":   "list1",
				"name":     "Card",
				"idLabels": "lbl1,lbl2,lbl3",
				"pos":      "bottom",
			},
		},
		{
			name:     "defaults position to bottom",
			listID:   "list1",
			cardName: "Card",
			args:     map[string]any{},
			want: map[string]string{
				"pos": "bottom",
			},
		},
		{
			name:     "respects explicit position",
			listID:   "list1",
			cardName: "Card",
			args:     map[string]any{"position": "top"},
			want: map[string]string{
				"pos": "top",
			},
		},
		{
			name:     "empty labelIDs omits idLabels",
			listID:   "list1",
			cardName: "Card",
			labelIDs: []string{},
			args:     map[string]any{},
			want: map[string]string{
				"idList": "list1",
				"name":   "Card",
				"pos":    "bottom",
			},
		},
		{
			name:     "empty desc is not set",
			listID:   "list1",
			cardName: "Card",
			args:     map[string]any{"description": ""},
			want: map[string]string{
				"idList": "list1",
				"name":   "Card",
				"pos":    "bottom",
			},
		},
		{
			name:     "all fields together",
			listID:   "listA",
			cardName: "Full Card",
			labelIDs: []string{"lbl1"},
			args: map[string]any{
				"description": "A full card",
				"due":         "2026-06-01",
				"position":    "top",
			},
			want: map[string]string{
				"idList":   "listA",
				"name":     "Full Card",
				"desc":     "A full card",
				"due":      "2026-06-01",
				"idLabels": "lbl1",
				"pos":      "top",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := buildCreateParams(tt.listID, tt.cardName, tt.labelIDs, tt.args)

			for key, wantVal := range tt.want {
				got := params.Get(key)
				if got != wantVal {
					t.Errorf("params[%q] = %q, want %q", key, got, wantVal)
				}
			}

			// Verify idLabels is absent when not expected.
			if _, exists := tt.want["idLabels"]; !exists {
				if params.Get("idLabels") != "" {
					t.Errorf("expected no idLabels, got %q", params.Get("idLabels"))
				}
			}

			// Verify desc is absent when not expected.
			if _, exists := tt.want["desc"]; !exists {
				if params.Get("desc") != "" {
					t.Errorf("expected no desc, got %q", params.Get("desc"))
				}
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		want   []string
		wantOK bool
	}{
		{
			name:   "[]any with strings",
			input:  []any{"a", "b"},
			want:   []string{"a", "b"},
			wantOK: true,
		},
		{
			name:   "[]any with non-string element",
			input:  []any{1},
			want:   nil,
			wantOK: false,
		},
		{
			name:   "plain string not an array",
			input:  "string",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "empty []any",
			input:  []any{},
			want:   []string{},
			wantOK: true,
		},
		{
			name:   "mixed types in []any",
			input:  []any{"a", 2, "c"},
			want:   nil,
			wantOK: false,
		},
		{
			name:   "nil input",
			input:  nil,
			want:   nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toStringSlice(tt.input)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}

			if tt.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCreateCard(t *testing.T) {
	boardLists := []trello.List{
		{ID: "list1", Name: "To Do"},
		{ID: "list2", Name: "In Progress"},
		{ID: "list3", Name: "Done"},
	}

	boardLabels := []trello.Label{
		{ID: "lbl1", Name: "Bug", Color: "red"},
		{ID: "lbl2", Name: "Feature", Color: "blue"},
	}

	tests := []struct {
		name       string
		args       map[string]any
		cfg        *config.Config
		client     *mockClient
		wantErr    bool
		errSubstr  string
		wantProto  bool // true = expect (nil, error) protocol error
		wantCardID string
	}{
		{
			name: "success with list_id",
			args: map[string]any{
				"board_id": "board1",
				"name":     "New Card",
				"list_id":  "list1",
			},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return boardLists, nil
				},
				createCardFn: func(_ context.Context, params url.Values) (*trello.Card, error) {
					return &trello.Card{
						ID:       "card1",
						Name:     params.Get("name"),
						ShortURL: "https://trello.com/c/abc",
					}, nil
				},
			},
			wantCardID: "card1",
		},
		{
			name: "success with list_name",
			args: map[string]any{
				"board_id":  "board1",
				"name":      "Another Card",
				"list_name": "In Progress",
			},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return boardLists, nil
				},
				createCardFn: func(_ context.Context, params url.Values) (*trello.Card, error) {
					if params.Get("idList") != "list2" {
						t.Errorf("expected idList=list2, got %q", params.Get("idList"))
					}
					return &trello.Card{
						ID:       "card2",
						Name:     params.Get("name"),
						ShortURL: "https://trello.com/c/def",
					}, nil
				},
			},
			wantCardID: "card2",
		},
		{
			name: "success with labels",
			args: map[string]any{
				"board_id": "board1",
				"name":     "Labeled Card",
				"list_id":  "list1",
				"labels":   []any{"Bug", "Feature"},
			},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return boardLists, nil
				},
				getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
					return boardLabels, nil
				},
				createCardFn: func(_ context.Context, params url.Values) (*trello.Card, error) {
					idLabels := params.Get("idLabels")
					if idLabels != "lbl1,lbl2" {
						t.Errorf("expected idLabels=lbl1,lbl2, got %q", idLabels)
					}
					return &trello.Card{
						ID:       "card3",
						Name:     params.Get("name"),
						ShortURL: "https://trello.com/c/ghi",
					}, nil
				},
			},
			wantCardID: "card3",
		},
		{
			name: "missing name returns tool error",
			args: map[string]any{
				"board_id": "board1",
				"list_id":  "list1",
			},
			client:    &mockClient{},
			wantErr:   true,
			errSubstr: "missing required argument: name",
		},
		{
			name: "missing list returns tool error",
			args: map[string]any{
				"board_id": "board1",
				"name":     "No List Card",
			},
			client:    &mockClient{},
			wantErr:   true,
			errSubstr: "one of list_id or list_name is required",
		},
		{
			name: "missing board_id and no default returns tool error",
			args: map[string]any{
				"name":    "Orphan Card",
				"list_id": "list1",
			},
			cfg:       &config.Config{},
			client:    &mockClient{},
			wantErr:   true,
			errSubstr: "missing board_id argument",
		},
		{
			name: "invalid label returns tool error",
			args: map[string]any{
				"board_id": "board1",
				"name":     "Bad Label Card",
				"list_id":  "list1",
				"labels":   []any{"NonExistent"},
			},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return boardLists, nil
				},
				getBoardLabelsFn: func(_ context.Context, _ string) ([]trello.Label, error) {
					return boardLabels, nil
				},
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "list_name not found returns tool error",
			args: map[string]any{
				"board_id":  "board1",
				"name":      "Bad List Card",
				"list_name": "Nonexistent List",
			},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return boardLists, nil
				},
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "API error on create returns tool error",
			args: map[string]any{
				"board_id": "board1",
				"name":     "Fail Card",
				"list_id":  "list1",
			},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return boardLists, nil
				},
				createCardFn: func(_ context.Context, _ url.Values) (*trello.Card, error) {
					return nil, &trello.APIError{StatusCode: 400, Body: "invalid value"}
				},
			},
			wantErr:   true,
			errSubstr: "Trello rejected the request (400)",
		},
		{
			name: "API error on create with non-API error returns protocol error",
			args: map[string]any{
				"board_id": "board1",
				"name":     "Fail Card",
				"list_id":  "list1",
			},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return boardLists, nil
				},
				createCardFn: func(_ context.Context, _ url.Values) (*trello.Card, error) {
					return nil, fmt.Errorf("network timeout")
				},
			},
			wantProto: true,
		},
		{
			name: "uses default_board from config",
			args: map[string]any{
				"name":    "Default Board Card",
				"list_id": "list1",
			},
			cfg: &config.Config{DefaultBoard: "boardDefault"},
			client: &mockClient{
				getBoardListsFn: func(_ context.Context, boardID string) ([]trello.List, error) {
					if boardID != "boardDefault" {
						t.Errorf("expected boardID=boardDefault, got %q", boardID)
					}
					return boardLists, nil
				},
				createCardFn: func(_ context.Context, _ url.Values) (*trello.Card, error) {
					return &trello.Card{
						ID:       "card4",
						Name:     "Default Board Card",
						ShortURL: "https://trello.com/c/jkl",
					}, nil
				},
			},
			wantCardID: "card4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, tt.cfg)
			ctx := testCtx()
			req := testRequest(tt.args)

			result, err := h.CreateCard(ctx, req)

			if tt.wantProto {
				if err == nil {
					t.Fatal("expected protocol error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected protocol error: %v", err)
			}

			if tt.wantErr {
				if result == nil {
					t.Fatal("expected tool error result, got nil")
				}
				if !result.IsError {
					t.Fatal("expected IsError to be true")
				}
				text := resultText(result)
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error text %q does not contain %q", text, tt.errSubstr)
				}
				return
			}

			if result == nil {
				t.Fatal("expected result, got nil")
			}

			var resp createCardResponse
			resultJSON(t, result, &resp)

			if resp.CardID != tt.wantCardID {
				t.Errorf("card_id = %q, want %q", resp.CardID, tt.wantCardID)
			}
			if resp.URL == "" {
				t.Error("expected non-empty url")
			}
		})
	}
}

func TestCreateCardLogging(t *testing.T) {
	mc := &mockClient{
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return []trello.List{{ID: "list1", Name: "To Do"}}, nil
		},
		createCardFn: func(_ context.Context, params url.Values) (*trello.Card, error) {
			return &trello.Card{
				ID:       "card1",
				Name:     params.Get("name"),
				ShortURL: "https://trello.com/c/abc",
			}, nil
		},
	}
	h := testHandler(mc, nil)
	ctx, rec := testCtxRecording()

	result, err := h.CreateCard(ctx, testRequest(map[string]any{
		"board_id": "board1",
		"name":     "New Card",
		"list_id":  "list1",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCardCreate) {
		t.Fatal("expected INFO entry for eventCardCreate")
	}
	entry := rec.Entry("INFO", eventCardCreate)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCardCreate entry")
	}
	if !entry.HasField("name") {
		t.Error("expected field 'name' on eventCardCreate entry")
	}
	if !entry.HasField("list") {
		t.Error("expected field 'list' on eventCardCreate entry")
	}
}

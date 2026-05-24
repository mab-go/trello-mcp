package handler

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestValidateTargetListArgs(t *testing.T) {
	tests := []struct {
		name    string
		listID  string
		listNm  string
		wantErr bool
	}{
		{"both empty", "", "", true},
		{"both set", "id", "name", true},
		{"id only", "id", "", false},
		{"name only", "", "name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateTargetListArgs(tt.listID, tt.listNm)
			if tt.wantErr && result == nil {
				t.Error("expected error result")
			}
			if !tt.wantErr && result != nil {
				t.Errorf("unexpected error: %s", resultText(result))
			}
		})
	}
}

func TestBuildMoveParams(t *testing.T) {
	tests := []struct {
		name       string
		listID     string
		boardID    string
		crossBoard bool
		args       map[string]any
		wantBoard  bool
		wantPos    string
	}{
		{
			name:       "same board",
			listID:     "l1",
			boardID:    "b1",
			crossBoard: false,
			args:       map[string]any{},
			wantBoard:  false,
		},
		{
			name:       "cross board",
			listID:     "l2",
			boardID:    "b2",
			crossBoard: true,
			args:       map[string]any{},
			wantBoard:  true,
		},
		{
			name:       "with position",
			listID:     "l1",
			boardID:    "b1",
			crossBoard: false,
			args:       map[string]any{"position": "top"},
			wantPos:    "top",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := buildMoveParams(tt.listID, tt.boardID, tt.crossBoard, tt.args)
			if params.Get("idList") != tt.listID {
				t.Errorf("idList = %q, want %q", params.Get("idList"), tt.listID)
			}
			if tt.wantBoard && params.Get("idBoard") != tt.boardID {
				t.Errorf("idBoard = %q, want %q", params.Get("idBoard"), tt.boardID)
			}
			if !tt.wantBoard && params.Get("idBoard") != "" {
				t.Errorf("idBoard should not be set, got %q", params.Get("idBoard"))
			}
			if tt.wantPos != "" && params.Get("pos") != tt.wantPos {
				t.Errorf("pos = %q, want %q", params.Get("pos"), tt.wantPos)
			}
		})
	}
}

func TestMoveCard(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		cfg     *config.Config
		client  *mockClient
		wantErr string
	}{
		{
			name: "success with target_list_id",
			args: map[string]any{"card_id": "c1", "target_list_id": "l2"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1", IDList: "l1"}, nil
				},
				updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
					return &trello.Card{ID: "c1", Name: "Moved Card", IDList: "l2", ShortURL: "https://trello.com/c/c1"}, nil
				},
				getBoardNameFn: func(_ context.Context, _ string) (string, error) {
					return "My Board", nil
				},
				getListNameFn: func(_ context.Context, listID string) (string, error) {
					names := map[string]string{"l1": "To Do", "l2": "Done"}
					return names[listID], nil
				},
			},
		},
		{
			name: "success with target_list_name",
			args: map[string]any{"card_id": "c1", "target_list_name": "Done"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1", IDList: "l1"}, nil
				},
				getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
					return []trello.List{{ID: "l1", Name: "To Do"}, {ID: "l2", Name: "Done"}}, nil
				},
				updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
					return &trello.Card{ID: "c1", Name: "Moved Card", IDList: "l2", ShortURL: "https://trello.com/c/c1"}, nil
				},
				getBoardNameFn: func(_ context.Context, _ string) (string, error) {
					return "My Board", nil
				},
				getListNameFn: func(_ context.Context, listID string) (string, error) {
					names := map[string]string{"l1": "To Do", "l2": "Done"}
					return names[listID], nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{},
			cfg:     &config.Config{},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "missing target list",
			args:    map[string]any{"card_id": "c1"},
			cfg:     &config.Config{},
			client:  &mockClient{},
			wantErr: "one of target_list_id or target_list_name is required",
		},
		{
			name:    "both target list args",
			args:    map[string]any{"card_id": "c1", "target_list_id": "l1", "target_list_name": "Done"},
			cfg:     &config.Config{},
			client:  &mockClient{},
			wantErr: "provide either target_list_id or target_list_name, not both",
		},
		{
			name: "board not allowed",
			args: map[string]any{"card_id": "c1", "target_list_id": "l2"},
			cfg:  &config.Config{AllowedBoards: []string{"other"}},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
			},
			wantErr: "not in the allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, tt.cfg)

			result, err := h.MoveCard(testCtx(), testRequest(tt.args))
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

			var resp moveCardResponse
			resultJSON(t, result, &resp)
			if resp.CardID != "c1" {
				t.Errorf("CardID = %q, want %q", resp.CardID, "c1")
			}
		})
	}
}

func TestMoveCardLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1", IDList: "l1"}, nil
		},
		updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
			return &trello.Card{ID: "c1", Name: "Moved Card", IDList: "l2", ShortURL: "https://trello.com/c/c1"}, nil
		},
		getBoardNameFn: func(_ context.Context, _ string) (string, error) {
			return "My Board", nil
		},
		getListNameFn: func(_ context.Context, listID string) (string, error) {
			names := map[string]string{"l1": "To Do", "l2": "Done"}
			return names[listID], nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.MoveCard(ctx, testRequest(map[string]any{"card_id": "c1", "target_list_id": "l2"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCardMove) {
		t.Fatal("expected INFO entry for eventCardMove")
	}
	entry := rec.Entry("INFO", eventCardMove)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCardMove entry")
	}
	if !entry.HasField("target_list") {
		t.Error("expected field 'target_list' on eventCardMove entry")
	}
}

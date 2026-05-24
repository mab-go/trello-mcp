package handler

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestArchiveCard(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		cfg     *config.Config
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"card_id": "c1"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				updateCardFn: func(_ context.Context, cardID string, params url.Values) (*trello.Card, error) {
					if params.Get("closed") != "true" {
						t.Errorf("expected closed=true, got %q", params.Get("closed"))
					}
					return &trello.Card{ID: cardID, Name: "Archived Card"}, nil
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
			name: "board not allowed",
			args: map[string]any{"card_id": "c1"},
			cfg:  &config.Config{AllowedBoards: []string{"other"}},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
			},
			wantErr: "not in the allowed list",
		},
		{
			name: "card not found",
			args: map[string]any{"card_id": "missing"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return nil, &trello.APIError{StatusCode: 404, Body: "not found"}
				},
			},
			wantErr: "Resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, tt.cfg)

			result, err := h.ArchiveCard(testCtx(), testRequest(tt.args))
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

			var resp archiveCardResponse
			resultJSON(t, result, &resp)
			if !resp.Archived {
				t.Error("expected Archived=true")
			}
		})
	}
}

func TestUnarchiveCard(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		cfg     *config.Config
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"card_id": "c1"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				updateCardFn: func(_ context.Context, cardID string, params url.Values) (*trello.Card, error) {
					if params.Get("closed") != "false" {
						t.Errorf("expected closed=false, got %q", params.Get("closed"))
					}
					return &trello.Card{ID: cardID, Name: "Restored Card"}, nil
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, tt.cfg)

			result, err := h.UnarchiveCard(testCtx(), testRequest(tt.args))
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

			var resp unarchiveCardResponse
			resultJSON(t, result, &resp)
			if resp.Archived {
				t.Error("expected Archived=false")
			}
		})
	}
}

func TestArchiveCardLogging(t *testing.T) {
	mc := &mockClient{
		getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
			return &trello.Card{ID: "c1", Name: "Archived Card"}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.ArchiveCard(ctx, testRequest(map[string]any{"card_id": "c1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCardArchive) {
		t.Fatal("expected INFO entry for eventCardArchive")
	}
	entry := rec.Entry("INFO", eventCardArchive)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCardArchive entry")
	}
	if !entry.HasField("name") {
		t.Error("expected field 'name' on eventCardArchive entry")
	}
}

func TestUnarchiveCardLogging(t *testing.T) {
	mc := &mockClient{
		getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		updateCardFn: func(_ context.Context, _ string, _ url.Values) (*trello.Card, error) {
			return &trello.Card{ID: "c1", Name: "Restored Card"}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.UnarchiveCard(ctx, testRequest(map[string]any{"card_id": "c1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCardUnarchive) {
		t.Fatal("expected INFO entry for eventCardUnarchive")
	}
	entry := rec.Entry("INFO", eventCardUnarchive)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCardUnarchive entry")
	}
	if !entry.HasField("name") {
		t.Error("expected field 'name' on eventCardUnarchive entry")
	}
}

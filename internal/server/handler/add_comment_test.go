package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestAddComment(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "success",
			args: map[string]any{"card_id": "c1", "text": "Hello world"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				addCommentFn: func(_ context.Context, _, text string) (*trello.Action, error) {
					return &trello.Action{
						ID:   "a1",
						Data: trello.ActionData{Text: text},
					}, nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{"text": "Hello"},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "missing text",
			args:    map[string]any{"card_id": "c1"},
			client:  &mockClient{},
			wantErr: "missing required argument: text",
		},
		{
			name: "board not allowed",
			args: map[string]any{"card_id": "c1", "text": "Hello"},
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
			cfg := &config.Config{}
			if tt.wantErr == "not in the allowed list" {
				cfg.AllowedBoards = []string{"other"}
			}
			h := testHandler(tt.client, cfg)

			result, err := h.AddComment(testCtx(), testRequest(tt.args))
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

			var resp addCommentResponse
			resultJSON(t, result, &resp)
			if resp.CommentID != "a1" {
				t.Errorf("CommentID = %q, want %q", resp.CommentID, "a1")
			}
			if resp.Text != "Hello world" {
				t.Errorf("Text = %q, want %q", resp.Text, "Hello world")
			}
		})
	}
}

func TestAddCommentTextTruncation(t *testing.T) {
	longText := strings.Repeat("a", 300)

	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		addCommentFn: func(_ context.Context, _, _ string) (*trello.Action, error) {
			return &trello.Action{
				ID:   "a1",
				Data: trello.ActionData{Text: longText},
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{})

	result, err := h.AddComment(testCtx(), testRequest(map[string]any{"card_id": "c1", "text": longText}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp addCommentResponse
	resultJSON(t, result, &resp)

	if len(resp.Text) != 200 {
		t.Errorf("text length = %d, want 200 (truncated)", len(resp.Text))
	}
}

func TestAddCommentLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		addCommentFn: func(_ context.Context, _, text string) (*trello.Action, error) {
			return &trello.Action{
				ID:   "a1",
				Data: trello.ActionData{Text: text},
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.AddComment(ctx, testRequest(map[string]any{"card_id": "c1", "text": "Hello world"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCommentAdd) {
		t.Fatal("expected INFO entry for eventCommentAdd")
	}
	entry := rec.Entry("INFO", eventCommentAdd)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCommentAdd entry")
	}
	if !entry.HasField("comment_id") {
		t.Error("expected field 'comment_id' on eventCommentAdd entry")
	}
}

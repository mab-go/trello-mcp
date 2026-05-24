package handler

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestAddAttachment(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		client  *mockClient
		wantErr string
	}{
		{
			name: "success with https",
			args: map[string]any{"card_id": "c1", "url": "https://example.com/file.pdf"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				addAttachmentFn: func(_ context.Context, _ string, params url.Values) (*trello.Attachment, error) {
					return &trello.Attachment{
						ID:   "att1",
						Name: "file.pdf",
						URL:  params.Get("url"),
					}, nil
				},
			},
		},
		{
			name: "success with http",
			args: map[string]any{"card_id": "c1", "url": "http://example.com/file.pdf"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				addAttachmentFn: func(_ context.Context, _ string, params url.Values) (*trello.Attachment, error) {
					return &trello.Attachment{ID: "att1", Name: "file.pdf", URL: params.Get("url")}, nil
				},
			},
		},
		{
			name: "success with custom name",
			args: map[string]any{"card_id": "c1", "url": "https://example.com/file.pdf", "name": "My File"},
			client: &mockClient{
				getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
				},
				addAttachmentFn: func(_ context.Context, _ string, params url.Values) (*trello.Attachment, error) {
					if params.Get("name") != "My File" {
						t.Errorf("name = %q, want %q", params.Get("name"), "My File")
					}
					return &trello.Attachment{ID: "att1", Name: "My File", URL: params.Get("url")}, nil
				},
			},
		},
		{
			name:    "missing card_id",
			args:    map[string]any{"url": "https://example.com"},
			client:  &mockClient{},
			wantErr: "missing required argument: card_id",
		},
		{
			name:    "missing url",
			args:    map[string]any{"card_id": "c1"},
			client:  &mockClient{},
			wantErr: "missing required argument: url",
		},
		{
			name:    "invalid scheme ftp",
			args:    map[string]any{"card_id": "c1", "url": "ftp://example.com/file"},
			client:  &mockClient{},
			wantErr: "http:// or https://",
		},
		{
			name:    "no scheme",
			args:    map[string]any{"card_id": "c1", "url": "example.com/file"},
			client:  &mockClient{},
			wantErr: "http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, &config.Config{})

			result, err := h.AddAttachment(testCtx(), testRequest(tt.args))
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

			var resp addAttachmentResponse
			resultJSON(t, result, &resp)
			if resp.AttachmentID != "att1" {
				t.Errorf("AttachmentID = %q, want %q", resp.AttachmentID, "att1")
			}
		})
	}
}

func TestAddAttachmentLogging(t *testing.T) {
	mc := &mockClient{
		getCardBasicFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{ID: "c1", IDBoard: "b1"}, nil
		},
		addAttachmentFn: func(_ context.Context, _ string, params url.Values) (*trello.Attachment, error) {
			return &trello.Attachment{
				ID:   "att1",
				Name: "file.pdf",
				URL:  params.Get("url"),
			}, nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.AddAttachment(ctx, testRequest(map[string]any{
		"card_id": "c1",
		"url":     "https://example.com/file.pdf",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventAttachmentAdd) {
		t.Fatal("expected INFO entry for eventAttachmentAdd")
	}
	entry := rec.Entry("INFO", eventAttachmentAdd)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventAttachmentAdd entry")
	}
	if !entry.HasField("attachment_id") {
		t.Error("expected field 'attachment_id' on eventAttachmentAdd entry")
	}
}

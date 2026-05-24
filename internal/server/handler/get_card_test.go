package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid RFC3339",
			input: "2024-01-15T10:30:00Z",
			want:  "2024-01-15 10:30",
		},
		{
			name:  "valid RFC3339 with offset",
			input: "2024-06-01T14:00:00+05:00",
			want:  "2024-06-01 09:00",
		},
		{
			name:  "invalid string returns empty",
			input: "not-a-date",
			want:  "",
		},
		{
			name:  "empty string returns empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDateTime(tt.input)
			if got != tt.want {
				t.Errorf("formatDateTime(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLabelNames(t *testing.T) {
	tests := []struct {
		name   string
		labels []trello.Label
		want   []string
	}{
		{
			name: "filters out empty names",
			labels: []trello.Label{
				{ID: "1", Name: "Bug"},
				{ID: "2", Name: ""},
				{ID: "3", Name: "Feature"},
			},
			want: []string{"Bug", "Feature"},
		},
		{
			name:   "empty slice returns empty",
			labels: []trello.Label{},
			want:   []string{},
		},
		{
			name:   "nil slice returns empty",
			labels: nil,
			want:   []string{},
		},
		{
			name: "all empty names",
			labels: []trello.Label{
				{ID: "1", Name: ""},
				{ID: "2", Name: ""},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := labelNames(tt.labels)
			if len(got) != len(tt.want) {
				t.Fatalf("labelNames() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("labelNames()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMemberUsernames(t *testing.T) {
	tests := []struct {
		name    string
		members []trello.CardMember
		want    []string
	}{
		{
			name: "extracts usernames",
			members: []trello.CardMember{
				{FullName: "John Doe", Username: "john"},
				{FullName: "Jane Smith", Username: "jane"},
			},
			want: []string{"john", "jane"},
		},
		{
			name:    "empty slice returns empty",
			members: []trello.CardMember{},
			want:    []string{},
		},
		{
			name:    "nil slice returns empty",
			members: nil,
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := memberUsernames(tt.members)
			if len(got) != len(tt.want) {
				t.Fatalf("memberUsernames() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("memberUsernames()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildChecklists(t *testing.T) {
	tests := []struct {
		name       string
		checklists []trello.Checklist
		want       []checklistDetail
	}{
		{
			name: "converts items with state",
			checklists: []trello.Checklist{
				{
					ID:   "cl1",
					Name: "Setup Tasks",
					CheckItems: []trello.CheckItem{
						{ID: "ci1", Name: "Install deps", State: "complete", Pos: 1},
						{ID: "ci2", Name: "Write tests", State: "incomplete", Pos: 2},
					},
				},
			},
			want: []checklistDetail{
				{
					ChecklistName: "Setup Tasks",
					Items: []checkItemDetail{
						{ItemName: "Install deps", Complete: true},
						{ItemName: "Write tests", Complete: false},
					},
				},
			},
		},
		{
			name: "multiple checklists",
			checklists: []trello.Checklist{
				{ID: "cl1", Name: "First", CheckItems: []trello.CheckItem{
					{ID: "ci1", Name: "A", State: "complete"},
				}},
				{ID: "cl2", Name: "Second", CheckItems: []trello.CheckItem{
					{ID: "ci2", Name: "B", State: "incomplete"},
				}},
			},
			want: []checklistDetail{
				{ChecklistName: "First", Items: []checkItemDetail{{ItemName: "A", Complete: true}}},
				{ChecklistName: "Second", Items: []checkItemDetail{{ItemName: "B", Complete: false}}},
			},
		},
		{
			name:       "empty slice returns empty",
			checklists: []trello.Checklist{},
			want:       []checklistDetail{},
		},
		{
			name: "checklist with no items",
			checklists: []trello.Checklist{
				{ID: "cl1", Name: "Empty List", CheckItems: []trello.CheckItem{}},
			},
			want: []checklistDetail{
				{ChecklistName: "Empty List", Items: []checkItemDetail{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildChecklists(tt.checklists)
			if len(got) != len(tt.want) {
				t.Fatalf("buildChecklists() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].ChecklistName != tt.want[i].ChecklistName {
					t.Errorf("checklist[%d].ChecklistName = %q, want %q",
						i, got[i].ChecklistName, tt.want[i].ChecklistName)
				}
				if len(got[i].Items) != len(tt.want[i].Items) {
					t.Fatalf("checklist[%d] has %d items, want %d",
						i, len(got[i].Items), len(tt.want[i].Items))
				}
				for j := range got[i].Items {
					if got[i].Items[j].ItemName != tt.want[i].Items[j].ItemName {
						t.Errorf("checklist[%d].Items[%d].ItemName = %q, want %q",
							i, j, got[i].Items[j].ItemName, tt.want[i].Items[j].ItemName)
					}
					if got[i].Items[j].Complete != tt.want[i].Items[j].Complete {
						t.Errorf("checklist[%d].Items[%d].Complete = %v, want %v",
							i, j, got[i].Items[j].Complete, tt.want[i].Items[j].Complete)
					}
				}
			}
		})
	}
}

func TestBuildComments(t *testing.T) {
	tests := []struct {
		name    string
		actions []trello.Action
		want    []commentDetail
	}{
		{
			name: "extracts comment fields",
			actions: []trello.Action{
				{
					ID:   "act1",
					Type: "commentCard",
					Data: trello.ActionData{Text: "Looks good!"},
					Date: "2024-03-10T14:30:00Z",
					MemberCreator: trello.ActionMember{
						FullName: "Alice",
						Username: "alice",
					},
				},
			},
			want: []commentDetail{
				{Author: "alice", Text: "Looks good!", Date: "2024-03-10 14:30"},
			},
		},
		{
			name: "invalid date formats to empty",
			actions: []trello.Action{
				{
					ID:            "act1",
					Date:          "bad-date",
					Data:          trello.ActionData{Text: "Hi"},
					MemberCreator: trello.ActionMember{Username: "bob"},
				},
			},
			want: []commentDetail{
				{Author: "bob", Text: "Hi", Date: ""},
			},
		},
		{
			name:    "empty actions returns empty",
			actions: []trello.Action{},
			want:    []commentDetail{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildComments(tt.actions)
			if len(got) != len(tt.want) {
				t.Fatalf("buildComments() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Author != tt.want[i].Author {
					t.Errorf("comment[%d].Author = %q, want %q", i, got[i].Author, tt.want[i].Author)
				}
				if got[i].Text != tt.want[i].Text {
					t.Errorf("comment[%d].Text = %q, want %q", i, got[i].Text, tt.want[i].Text)
				}
				if got[i].Date != tt.want[i].Date {
					t.Errorf("comment[%d].Date = %q, want %q", i, got[i].Date, tt.want[i].Date)
				}
			}
		})
	}
}

func TestBuildAttachments(t *testing.T) {
	tests := []struct {
		name        string
		attachments []trello.Attachment
		want        []attachmentDetail
	}{
		{
			name: "maps attachment fields",
			attachments: []trello.Attachment{
				{
					ID:   "att1",
					Name: "screenshot.png",
					URL:  "https://trello.com/att1",
					Date: "2024-05-20T09:15:00Z",
				},
			},
			want: []attachmentDetail{
				{Name: "screenshot.png", URL: "https://trello.com/att1", Date: "2024-05-20 09:15"},
			},
		},
		{
			name:        "empty attachments returns empty",
			attachments: []trello.Attachment{},
			want:        []attachmentDetail{},
		},
		{
			name: "multiple attachments",
			attachments: []trello.Attachment{
				{ID: "a1", Name: "file1.pdf", URL: "https://example.com/1", Date: "2024-01-01T00:00:00Z"},
				{ID: "a2", Name: "file2.txt", URL: "https://example.com/2", Date: "2024-12-31T23:59:00Z"},
			},
			want: []attachmentDetail{
				{Name: "file1.pdf", URL: "https://example.com/1", Date: "2024-01-01 00:00"},
				{Name: "file2.txt", URL: "https://example.com/2", Date: "2024-12-31 23:59"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAttachments(tt.attachments)
			if len(got) != len(tt.want) {
				t.Fatalf("buildAttachments() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("attachment[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].URL != tt.want[i].URL {
					t.Errorf("attachment[%d].URL = %q, want %q", i, got[i].URL, tt.want[i].URL)
				}
				if got[i].Date != tt.want[i].Date {
					t.Errorf("attachment[%d].Date = %q, want %q", i, got[i].Date, tt.want[i].Date)
				}
			}
		})
	}
}

func TestBuildGetCardResponse(t *testing.T) {
	due := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		card      *trello.Card
		listName  string
		boardName string
		want      getCardResponse
	}{
		{
			name: "full card with all fields",
			card: &trello.Card{
				ID:               "card1",
				Name:             "Test Card",
				Desc:             "A description",
				IDList:           "list1",
				IDBoard:          "board1",
				ShortURL:         "https://trello.com/c/abc",
				DateLastActivity: "2024-06-15T12:00:00Z",
				Due:              &due,
				DueComplete:      true,
				Labels: []trello.Label{
					{ID: "l1", Name: "Bug"},
					{ID: "l2", Name: "Urgent"},
				},
				Members: []trello.CardMember{
					{FullName: "Alice", Username: "alice"},
				},
				Checklists: []trello.Checklist{
					{
						ID:   "cl1",
						Name: "Tasks",
						CheckItems: []trello.CheckItem{
							{ID: "ci1", Name: "Item 1", State: "complete"},
						},
					},
				},
				Actions: []trello.Action{
					{
						ID:            "act1",
						Date:          "2024-06-15T10:00:00Z",
						Data:          trello.ActionData{Text: "Nice work"},
						MemberCreator: trello.ActionMember{Username: "bob"},
					},
				},
				Attachments: []trello.Attachment{
					{ID: "att1", Name: "file.pdf", URL: "https://example.com/file.pdf", Date: "2024-06-14T08:00:00Z"},
				},
			},
			listName:  "In Progress",
			boardName: "My Board",
			want: getCardResponse{
				CardID:      "card1",
				Name:        "Test Card",
				List:        "In Progress",
				Board:       "My Board",
				URL:         "https://trello.com/c/abc",
				Description: "A description",
				Due:         "2024-06-15",
				DueComplete: true,
				Labels:      []string{"Bug", "Urgent"},
				Members:     []string{"alice"},
				Checklists: []checklistDetail{
					{ChecklistName: "Tasks", Items: []checkItemDetail{{ItemName: "Item 1", Complete: true}}},
				},
				Comments: []commentDetail{
					{Author: "bob", Text: "Nice work", Date: "2024-06-15 10:00"},
				},
				Attachments: []attachmentDetail{
					{Name: "file.pdf", URL: "https://example.com/file.pdf", Date: "2024-06-14 08:00"},
				},
			},
		},
		{
			name: "card with no due date",
			card: &trello.Card{
				ID:       "card2",
				Name:     "No Due",
				IDList:   "list1",
				IDBoard:  "board1",
				ShortURL: "https://trello.com/c/def",
			},
			listName:  "To Do",
			boardName: "Board",
			want: getCardResponse{
				CardID:      "card2",
				Name:        "No Due",
				List:        "To Do",
				Board:       "Board",
				URL:         "https://trello.com/c/def",
				Due:         "",
				DueComplete: false,
				Labels:      []string{},
				Members:     []string{},
				Checklists:  []checklistDetail{},
				Comments:    []commentDetail{},
				Attachments: []attachmentDetail{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildGetCardResponse(tt.card, tt.listName, tt.boardName)

			if got.CardID != tt.want.CardID {
				t.Errorf("CardID = %q, want %q", got.CardID, tt.want.CardID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.List != tt.want.List {
				t.Errorf("List = %q, want %q", got.List, tt.want.List)
			}
			if got.Board != tt.want.Board {
				t.Errorf("Board = %q, want %q", got.Board, tt.want.Board)
			}
			if got.URL != tt.want.URL {
				t.Errorf("URL = %q, want %q", got.URL, tt.want.URL)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.want.Description)
			}
			if got.Due != tt.want.Due {
				t.Errorf("Due = %q, want %q", got.Due, tt.want.Due)
			}
			if got.DueComplete != tt.want.DueComplete {
				t.Errorf("DueComplete = %v, want %v", got.DueComplete, tt.want.DueComplete)
			}
			if len(got.Labels) != len(tt.want.Labels) {
				t.Fatalf("Labels has %d items, want %d", len(got.Labels), len(tt.want.Labels))
			}
			for i := range got.Labels {
				if got.Labels[i] != tt.want.Labels[i] {
					t.Errorf("Labels[%d] = %q, want %q", i, got.Labels[i], tt.want.Labels[i])
				}
			}
			if len(got.Members) != len(tt.want.Members) {
				t.Fatalf("Members has %d items, want %d", len(got.Members), len(tt.want.Members))
			}
			for i := range got.Members {
				if got.Members[i] != tt.want.Members[i] {
					t.Errorf("Members[%d] = %q, want %q", i, got.Members[i], tt.want.Members[i])
				}
			}
			if len(got.Checklists) != len(tt.want.Checklists) {
				t.Fatalf("Checklists has %d items, want %d", len(got.Checklists), len(tt.want.Checklists))
			}
			if len(got.Comments) != len(tt.want.Comments) {
				t.Fatalf("Comments has %d items, want %d", len(got.Comments), len(tt.want.Comments))
			}
			if len(got.Attachments) != len(tt.want.Attachments) {
				t.Fatalf("Attachments has %d items, want %d", len(got.Attachments), len(tt.want.Attachments))
			}
		})
	}
}

func TestGetCard(t *testing.T) {
	due := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	fullCard := &trello.Card{
		ID:               "card123",
		Name:             "My Card",
		Desc:             "Card description",
		IDList:           "list1",
		IDBoard:          "board1",
		ShortURL:         "https://trello.com/c/xyz",
		DateLastActivity: "2024-06-15T12:00:00Z",
		Due:              &due,
		DueComplete:      false,
		Labels:           []trello.Label{{ID: "l1", Name: "Bug"}},
		Members:          []trello.CardMember{{FullName: "Alice", Username: "alice"}},
	}

	tests := []struct {
		name      string
		args      map[string]any
		cfg       *config.Config
		client    *mockClient
		wantErr   bool
		errSubstr string
	}{
		{
			name: "success",
			args: map[string]any{"card_id": "card123"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardFn: func(_ context.Context, cardID string) (*trello.Card, error) {
					if cardID != "card123" {
						t.Errorf("GetCard called with %q, want %q", cardID, "card123")
					}
					return fullCard, nil
				},
				getListNameFn: func(_ context.Context, _ string) (string, error) {
					return "To Do", nil
				},
				getBoardNameFn: func(_ context.Context, _ string) (string, error) {
					return "My Board", nil
				},
			},
			wantErr: false,
		},
		{
			name:      "missing card_id",
			args:      map[string]any{},
			cfg:       &config.Config{},
			client:    &mockClient{},
			wantErr:   true,
			errSubstr: "missing required argument: card_id",
		},
		{
			name:      "empty card_id",
			args:      map[string]any{"card_id": ""},
			cfg:       &config.Config{},
			client:    &mockClient{},
			wantErr:   true,
			errSubstr: "missing required argument: card_id",
		},
		{
			name: "card not found 404",
			args: map[string]any{"card_id": "missing"},
			cfg:  &config.Config{},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return nil, &trello.APIError{StatusCode: 404, Body: "not found"}
				},
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "board not allowed",
			args: map[string]any{"card_id": "card123"},
			cfg:  &config.Config{AllowedBoards: []string{"otherBoard"}},
			client: &mockClient{
				getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
					return fullCard, nil
				},
			},
			wantErr:   true,
			errSubstr: "not in the allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(tt.client, tt.cfg)
			ctx := testCtx()
			req := testRequest(tt.args)

			result, err := h.GetCard(ctx, req)
			if err != nil {
				t.Fatalf("GetCard returned unexpected protocol error: %v", err)
			}
			if result == nil {
				t.Fatal("GetCard returned nil result")
			}

			if tt.wantErr {
				if !result.IsError {
					t.Fatalf("expected tool error, got success: %s", resultText(result))
				}
				text := resultText(result)
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error text %q does not contain %q", text, tt.errSubstr)
				}
				return
			}

			if result.IsError {
				t.Fatalf("unexpected tool error: %s", resultText(result))
			}

			var resp getCardResponse
			resultJSON(t, result, &resp)

			if resp.CardID != "card123" {
				t.Errorf("CardID = %q, want %q", resp.CardID, "card123")
			}
			if resp.Name != "My Card" {
				t.Errorf("Name = %q, want %q", resp.Name, "My Card")
			}
			if resp.List != "To Do" {
				t.Errorf("List = %q, want %q", resp.List, "To Do")
			}
			if resp.Board != "My Board" {
				t.Errorf("Board = %q, want %q", resp.Board, "My Board")
			}
			if resp.Due != "2024-06-15" {
				t.Errorf("Due = %q, want %q", resp.Due, "2024-06-15")
			}
			if len(resp.Labels) != 1 || resp.Labels[0] != "Bug" {
				t.Errorf("Labels = %v, want [Bug]", resp.Labels)
			}
			if len(resp.Members) != 1 || resp.Members[0] != "alice" {
				t.Errorf("Members = %v, want [alice]", resp.Members)
			}
		})
	}
}

func TestGetCardLogging(t *testing.T) {
	mc := &mockClient{
		getCardFn: func(_ context.Context, _ string) (*trello.Card, error) {
			return &trello.Card{
				ID:      "card123",
				Name:    "My Card",
				IDList:  "list1",
				IDBoard: "board1",
			}, nil
		},
		getListNameFn: func(_ context.Context, _ string) (string, error) {
			return "To Do", nil
		},
		getBoardNameFn: func(_ context.Context, _ string) (string, error) {
			return "My Board", nil
		},
	}
	h := testHandler(mc, &config.Config{})
	ctx, rec := testCtxRecording()

	result, err := h.GetCard(ctx, testRequest(map[string]any{"card_id": "card123"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventCardGet) {
		t.Fatal("expected INFO entry for eventCardGet")
	}
	entry := rec.Entry("INFO", eventCardGet)
	if !entry.HasField("card_id") {
		t.Error("expected field 'card_id' on eventCardGet entry")
	}
}

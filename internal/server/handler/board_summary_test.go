package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestClassifyDueCards(t *testing.T) {
	now := time.Now().UTC()
	listNames := map[string]string{"l1": "To Do", "l2": "Done"}

	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)
	nextWeek := now.Add(8 * 24 * time.Hour)

	cards := []trello.Card{
		{ID: "c1", Name: "Overdue", IDList: "l1", Due: &yesterday},
		{ID: "c2", Name: "Due Soon", IDList: "l1", Due: &tomorrow},
		{ID: "c3", Name: "Due Later", IDList: "l2", Due: &nextWeek},
		{ID: "c4", Name: "No Due", IDList: "l1"},
		{ID: "c5", Name: "Completed", IDList: "l1", Due: &yesterday, DueComplete: true},
	}

	overdue, dueSoon := classifyDueCards(cards, listNames)

	if len(overdue) != 1 {
		t.Errorf("overdue count = %d, want 1", len(overdue))
	} else if overdue[0].CardID != "c1" {
		t.Errorf("overdue[0].CardID = %q, want %q", overdue[0].CardID, "c1")
	}

	if len(dueSoon) != 1 {
		t.Errorf("dueSoon count = %d, want 1", len(dueSoon))
	} else if dueSoon[0].CardID != "c2" {
		t.Errorf("dueSoon[0].CardID = %q, want %q", dueSoon[0].CardID, "c2")
	}
}

func TestClassifyDueCardsEmpty(t *testing.T) {
	overdue, dueSoon := classifyDueCards(nil, nil)
	if overdue != nil {
		t.Errorf("overdue = %v, want nil", overdue)
	}
	if dueSoon != nil {
		t.Errorf("dueSoon = %v, want nil", dueSoon)
	}
}

func TestBuildSummaryCardEntry(t *testing.T) {
	due := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	listNames := map[string]string{"l1": "To Do"}

	card := trello.Card{
		ID:     "c1",
		Name:   "Test Card",
		IDList: "l1",
		Due:    &due,
		Labels: []trello.Label{{Name: "Bug"}, {Name: ""}, {Name: "P1"}},
	}

	entry := buildSummaryCardEntry(card, listNames)

	if entry.CardID != "c1" {
		t.Errorf("CardID = %q, want %q", entry.CardID, "c1")
	}
	if entry.ListName != "To Do" {
		t.Errorf("ListName = %q, want %q", entry.ListName, "To Do")
	}
	if entry.Due != "2024-06-15" {
		t.Errorf("Due = %q, want %q", entry.Due, "2024-06-15")
	}
	if len(entry.Labels) != 2 {
		t.Errorf("len(Labels) = %d, want 2", len(entry.Labels))
	}
}

func TestBoardSummary(t *testing.T) {
	due := time.Now().UTC().Add(-24 * time.Hour)
	lists := []trello.List{
		{ID: "l1", Name: "To Do", Pos: 1024},
		{ID: "l2", Name: "Done", Pos: 2048},
	}
	cards := []trello.Card{
		{ID: "c1", Name: "Overdue Task", IDList: "l1", Due: &due},
		{ID: "c2", Name: "No Due Task", IDList: "l1"},
		{ID: "c3", Name: "Done Task", IDList: "l2"},
	}

	mc := &mockClient{
		getBoardNameFn: func(_ context.Context, _ string) (string, error) {
			return "Test Board", nil
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return lists, nil
		},
		getBoardCardsFn: func(_ context.Context, _ string, _ string) ([]trello.Card, error) {
			return cards, nil
		},
	}
	h := testHandler(mc, &config.Config{DefaultBoard: "b1"})

	result, err := h.BoardSummary(testCtx(), testRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	var resp boardSummaryResponse
	resultJSON(t, result, &resp)

	if resp.BoardName != "Test Board" {
		t.Errorf("BoardName = %q, want %q", resp.BoardName, "Test Board")
	}
	if resp.TotalCards != 3 {
		t.Errorf("TotalCards = %d, want 3", resp.TotalCards)
	}
	if resp.OverdueCount != 1 {
		t.Errorf("OverdueCount = %d, want 1", resp.OverdueCount)
	}
	if len(resp.Lists) != 2 {
		t.Errorf("len(Lists) = %d, want 2", len(resp.Lists))
	}
}

func TestBoardSummaryMissingBoard(t *testing.T) {
	h := testHandler(&mockClient{}, &config.Config{})

	result, err := h.BoardSummary(testCtx(), testRequest(map[string]any{}))
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

func TestBoardSummaryLogging(t *testing.T) {
	lists := []trello.List{
		{ID: "l1", Name: "To Do", Pos: 1024},
		{ID: "l2", Name: "Done", Pos: 2048},
	}
	cards := []trello.Card{
		{ID: "c1", Name: "Task 1", IDList: "l1"},
		{ID: "c2", Name: "Task 2", IDList: "l2"},
	}

	mc := &mockClient{
		getBoardNameFn: func(_ context.Context, _ string) (string, error) {
			return "Test Board", nil
		},
		getBoardListsFn: func(_ context.Context, _ string) ([]trello.List, error) {
			return lists, nil
		},
		getBoardCardsFn: func(_ context.Context, _, _ string) ([]trello.Card, error) {
			return cards, nil
		},
	}
	h := testHandler(mc, &config.Config{DefaultBoard: "b1"})
	ctx, rec := testCtxRecording()

	result, err := h.BoardSummary(ctx, testRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", resultText(result))
	}

	if !rec.HasEntry("INFO", eventBoardSummarize) {
		t.Fatal("expected INFO entry for eventBoardSummarize")
	}
	entry := rec.Entry("INFO", eventBoardSummarize)
	if !entry.HasField("board_id") {
		t.Error("expected field 'board_id' on eventBoardSummarize entry")
	}
	if !entry.HasField("total_cards") {
		t.Error("expected field 'total_cards' on eventBoardSummarize entry")
	}
}

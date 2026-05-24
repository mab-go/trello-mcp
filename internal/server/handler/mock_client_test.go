package handler

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type mockClient struct {
	getBoardsFn         func(ctx context.Context) ([]trello.Board, error)
	getBoardListsFn     func(ctx context.Context, boardID string) ([]trello.List, error)
	getBoardCardsFn     func(ctx context.Context, boardID, filter string) ([]trello.Card, error)
	getListCardsFn      func(ctx context.Context, listID, filter string) ([]trello.Card, error)
	getCardFn           func(ctx context.Context, cardID string) (*trello.Card, error)
	getCardBasicFn      func(ctx context.Context, cardID string) (*trello.Card, error)
	createCardFn        func(ctx context.Context, params url.Values) (*trello.Card, error)
	updateCardFn        func(ctx context.Context, cardID string, params url.Values) (*trello.Card, error)
	getBoardLabelsFn    func(ctx context.Context, boardID string) ([]trello.Label, error)
	getCardChecklistsFn func(ctx context.Context, cardID string) ([]trello.Checklist, error)
	updateCheckItemFn   func(ctx context.Context, cardID, itemID string, params url.Values) (*trello.CheckItem, error)
	createChecklistFn   func(ctx context.Context, cardID, name string) (*trello.Checklist, error)
	createCheckItemFn   func(ctx context.Context, checklistID string, params url.Values) (*trello.CheckItem, error)
	addCardLabelFn      func(ctx context.Context, cardID, labelID string) error
	removeCardLabelFn   func(ctx context.Context, cardID, labelID string) error
	getBoardNameFn      func(ctx context.Context, boardID string) (string, error)
	getListNameFn       func(ctx context.Context, listID string) (string, error)
	searchFn            func(ctx context.Context, query string, boardIDs []string, limit int) (*trello.SearchResult, error)
	addCommentFn        func(ctx context.Context, cardID, text string) (*trello.Action, error)
	addAttachmentFn     func(ctx context.Context, cardID string, params url.Values) (*trello.Attachment, error)
	getChecklistBoardFn func(ctx context.Context, checklistID string) (*trello.Board, error)
	createListFn        func(ctx context.Context, boardID string, params url.Values) (*trello.List, error)
}

func (m *mockClient) GetBoards(ctx context.Context) ([]trello.Board, error) {
	if m.getBoardsFn == nil {
		panic("unexpected call to GetBoards")
	}
	return m.getBoardsFn(ctx)
}

func (m *mockClient) GetBoardLists(ctx context.Context, boardID string) ([]trello.List, error) {
	if m.getBoardListsFn == nil {
		panic("unexpected call to GetBoardLists")
	}
	return m.getBoardListsFn(ctx, boardID)
}

func (m *mockClient) GetBoardCards(ctx context.Context, boardID, filter string) ([]trello.Card, error) {
	if m.getBoardCardsFn == nil {
		panic("unexpected call to GetBoardCards")
	}
	return m.getBoardCardsFn(ctx, boardID, filter)
}

func (m *mockClient) GetListCards(ctx context.Context, listID, filter string) ([]trello.Card, error) {
	if m.getListCardsFn == nil {
		panic("unexpected call to GetListCards")
	}
	return m.getListCardsFn(ctx, listID, filter)
}

func (m *mockClient) GetCard(ctx context.Context, cardID string) (*trello.Card, error) {
	if m.getCardFn == nil {
		panic("unexpected call to GetCard")
	}
	return m.getCardFn(ctx, cardID)
}

func (m *mockClient) GetCardBasic(ctx context.Context, cardID string) (*trello.Card, error) {
	if m.getCardBasicFn == nil {
		panic("unexpected call to GetCardBasic")
	}
	return m.getCardBasicFn(ctx, cardID)
}

func (m *mockClient) CreateCard(ctx context.Context, params url.Values) (*trello.Card, error) {
	if m.createCardFn == nil {
		panic("unexpected call to CreateCard")
	}
	return m.createCardFn(ctx, params)
}

func (m *mockClient) UpdateCard(ctx context.Context, cardID string, params url.Values) (*trello.Card, error) {
	if m.updateCardFn == nil {
		panic("unexpected call to UpdateCard")
	}
	return m.updateCardFn(ctx, cardID, params)
}

func (m *mockClient) GetBoardLabels(ctx context.Context, boardID string) ([]trello.Label, error) {
	if m.getBoardLabelsFn == nil {
		panic("unexpected call to GetBoardLabels")
	}
	return m.getBoardLabelsFn(ctx, boardID)
}

func (m *mockClient) GetCardChecklists(ctx context.Context, cardID string) ([]trello.Checklist, error) {
	if m.getCardChecklistsFn == nil {
		panic("unexpected call to GetCardChecklists")
	}
	return m.getCardChecklistsFn(ctx, cardID)
}

func (m *mockClient) UpdateCheckItem(ctx context.Context, cardID, itemID string, params url.Values) (*trello.CheckItem, error) {
	if m.updateCheckItemFn == nil {
		panic("unexpected call to UpdateCheckItem")
	}
	return m.updateCheckItemFn(ctx, cardID, itemID, params)
}

func (m *mockClient) CreateChecklist(ctx context.Context, cardID, name string) (*trello.Checklist, error) {
	if m.createChecklistFn == nil {
		panic("unexpected call to CreateChecklist")
	}
	return m.createChecklistFn(ctx, cardID, name)
}

func (m *mockClient) CreateCheckItem(ctx context.Context, checklistID string, params url.Values) (*trello.CheckItem, error) {
	if m.createCheckItemFn == nil {
		panic("unexpected call to CreateCheckItem")
	}
	return m.createCheckItemFn(ctx, checklistID, params)
}

func (m *mockClient) AddCardLabel(ctx context.Context, cardID, labelID string) error {
	if m.addCardLabelFn == nil {
		panic("unexpected call to AddCardLabel")
	}
	return m.addCardLabelFn(ctx, cardID, labelID)
}

func (m *mockClient) RemoveCardLabel(ctx context.Context, cardID, labelID string) error {
	if m.removeCardLabelFn == nil {
		panic("unexpected call to RemoveCardLabel")
	}
	return m.removeCardLabelFn(ctx, cardID, labelID)
}

func (m *mockClient) GetBoardName(ctx context.Context, boardID string) (string, error) {
	if m.getBoardNameFn == nil {
		panic("unexpected call to GetBoardName")
	}
	return m.getBoardNameFn(ctx, boardID)
}

func (m *mockClient) GetListName(ctx context.Context, listID string) (string, error) {
	if m.getListNameFn == nil {
		panic("unexpected call to GetListName")
	}
	return m.getListNameFn(ctx, listID)
}

func (m *mockClient) Search(ctx context.Context, query string, boardIDs []string, limit int) (*trello.SearchResult, error) {
	if m.searchFn == nil {
		panic("unexpected call to Search")
	}
	return m.searchFn(ctx, query, boardIDs, limit)
}

func (m *mockClient) AddComment(ctx context.Context, cardID, text string) (*trello.Action, error) {
	if m.addCommentFn == nil {
		panic("unexpected call to AddComment")
	}
	return m.addCommentFn(ctx, cardID, text)
}

func (m *mockClient) AddAttachment(ctx context.Context, cardID string, params url.Values) (*trello.Attachment, error) {
	if m.addAttachmentFn == nil {
		panic("unexpected call to AddAttachment")
	}
	return m.addAttachmentFn(ctx, cardID, params)
}

func (m *mockClient) GetChecklistBoard(ctx context.Context, checklistID string) (*trello.Board, error) {
	if m.getChecklistBoardFn == nil {
		panic("unexpected call to GetChecklistBoard")
	}
	return m.getChecklistBoardFn(ctx, checklistID)
}

func (m *mockClient) CreateList(ctx context.Context, boardID string, params url.Values) (*trello.List, error) {
	if m.createListFn == nil {
		panic("unexpected call to CreateList")
	}
	return m.createListFn(ctx, boardID, params)
}

// testCtx returns a context with a no-op logger suitable for tests.
func testCtx() context.Context {
	log := logging.NewDefaultLogger()
	return logging.NewContext(context.Background(), log)
}

// testCtxRecording returns a context with a recording logger for asserting log output.
func testCtxRecording() (context.Context, *recordingLogger) {
	rec := newRecordingLogger()
	ctx := logging.NewContext(context.Background(), rec)
	return ctx, rec
}

// testHandler creates a TrelloHandler with the given mock client and config.
func testHandler(client TrelloClient, cfg *config.Config) *TrelloHandler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return NewTrelloHandler(client, cfg)
}

// testRequest builds a CallToolRequest with the given arguments.
func testRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// resultText extracts the text content from a tool result.
func resultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// resultJSON unmarshals a tool result's text content into v.
func resultJSON(t *testing.T, result *mcp.CallToolResult, v any) {
	t.Helper()
	text := resultText(result)
	if text == "" {
		t.Fatal("result has no text content")
	}
	if err := json.Unmarshal([]byte(text), v); err != nil {
		t.Fatalf("unmarshal result JSON: %v", err)
	}
}

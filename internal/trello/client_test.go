package trello

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testServer(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClientWithHTTP("testkey", "testtoken", srv.Client(), srv.URL)
}

func TestAuthParams(t *testing.T) {
	var gotKey, gotToken string
	mux := http.NewServeMux()
	mux.HandleFunc("/members/me/boards", func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.URL.Query().Get("key")
		gotToken = r.URL.Query().Get("token")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	})

	client := testServer(t, mux)
	_, _ = client.GetBoards(context.Background())

	if gotKey != "testkey" {
		t.Errorf("key = %q, want %q", gotKey, "testkey")
	}
	if gotToken != "testtoken" {
		t.Errorf("token = %q, want %q", gotToken, "testtoken")
	}
}

func TestGetBoards(t *testing.T) {
	boards := []Board{
		{ID: "b1", Name: "Board One"},
		{ID: "b2", Name: "Board Two"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/members/me/boards", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filter") != "open" {
			t.Errorf("filter = %q, want %q", r.URL.Query().Get("filter"), "open")
		}
		_ = json.NewEncoder(w).Encode(boards)
	})

	client := testServer(t, mux)
	result, err := client.GetBoards(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
	if result[0].Name != "Board One" {
		t.Errorf("name = %q, want %q", result[0].Name, "Board One")
	}
}

func TestGetBoardLists(t *testing.T) {
	lists := []List{
		{ID: "l1", Name: "To Do", Pos: 1024},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/boards/b1/lists", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(lists)
	})

	client := testServer(t, mux)
	result, err := client.GetBoardLists(context.Background(), "b1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Name != "To Do" {
		t.Errorf("name = %q, want %q", result[0].Name, "To Do")
	}
}

func TestGetCard(t *testing.T) {
	card := Card{ID: "c1", Name: "Test Card", Desc: "A description"}
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(card)
	})

	client := testServer(t, mux)
	result, err := client.GetCard(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Test Card" {
		t.Errorf("name = %q, want %q", result.Name, "Test Card")
	}
}

func TestCreateCard(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		card := Card{ID: "c1", Name: r.URL.Query().Get("name")}
		_ = json.NewEncoder(w).Encode(card)
	})

	client := testServer(t, mux)
	params := url.Values{}
	params.Set("name", "New Card")
	params.Set("idList", "l1")

	result, err := client.CreateCard(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "c1" {
		t.Errorf("ID = %q, want %q", result.ID, "c1")
	}
}

func TestUpdateCard(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		card := Card{ID: "c1", Name: "Updated"}
		_ = json.NewEncoder(w).Encode(card)
	})

	client := testServer(t, mux)
	params := url.Values{}
	params.Set("name", "Updated")

	result, err := client.UpdateCard(context.Background(), "c1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated" {
		t.Errorf("name = %q, want %q", result.Name, "Updated")
	}
}

func TestAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"bad request", 400, "invalid value"},
		{"unauthorized", 401, "unauthorized"},
		{"forbidden", 403, "access denied"},
		{"not found", 404, "resource not found"},
		{"rate limited", 429, "rate limit exceeded"},
		{"server error", 500, "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/boards/b1/lists", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			})

			client := testServer(t, mux)
			_, err := client.GetBoardLists(context.Background(), "b1")
			if err == nil {
				t.Fatal("expected error")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if apiErr.Body != tt.body {
				t.Errorf("Body = %q, want %q", apiErr.Body, tt.body)
			}
		})
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &APIError{StatusCode: 404, Body: "not found"}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Error() = %q, missing status code", err.Error())
	}
}

func TestGetBoardName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/boards/b1", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(Board{ID: "b1", Name: "My Board"})
	})

	client := testServer(t, mux)
	name, err := client.GetBoardName(context.Background(), "b1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "My Board" {
		t.Errorf("name = %q, want %q", name, "My Board")
	}
}

func TestGetListName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/lists/l1", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(List{ID: "l1", Name: "To Do"})
	})

	client := testServer(t, mux)
	name, err := client.GetListName(context.Background(), "l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "To Do" {
		t.Errorf("name = %q, want %q", name, "To Do")
	}
}

func TestSearch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("query") != "test" {
			t.Errorf("query = %q, want %q", q.Get("query"), "test")
		}
		if q.Get("idBoards") != "b1" {
			t.Errorf("idBoards = %q, want %q", q.Get("idBoards"), "b1")
		}
		result := SearchResult{
			Cards:  []Card{{ID: "c1", Name: "Found Card"}},
			Boards: []Board{{ID: "b1", Name: "Found Board"}},
		}
		_ = json.NewEncoder(w).Encode(result)
	})

	client := testServer(t, mux)
	result, err := client.Search(context.Background(), "test", []string{"b1"}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Cards) != 1 {
		t.Errorf("card count = %d, want 1", len(result.Cards))
	}
	if len(result.Boards) != 1 {
		t.Errorf("board count = %d, want 1", len(result.Boards))
	}
}

func TestAddComment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1/actions/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		action := Action{ID: "a1", Data: ActionData{Text: r.URL.Query().Get("text")}}
		_ = json.NewEncoder(w).Encode(action)
	})

	client := testServer(t, mux)
	result, err := client.AddComment(context.Background(), "c1", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "a1" {
		t.Errorf("ID = %q, want %q", result.ID, "a1")
	}
}

func TestAddAttachment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1/attachments", func(w http.ResponseWriter, r *http.Request) {
		att := Attachment{ID: "att1", Name: "file.pdf", URL: r.URL.Query().Get("url")}
		_ = json.NewEncoder(w).Encode(att)
	})

	client := testServer(t, mux)
	params := url.Values{}
	params.Set("url", "https://example.com/file.pdf")

	result, err := client.AddAttachment(context.Background(), "c1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "att1" {
		t.Errorf("ID = %q, want %q", result.ID, "att1")
	}
}

func TestGetBoardLabels(t *testing.T) {
	labels := []Label{
		{ID: "l1", Name: "Bug", Color: "red"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/boards/b1/labels", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(labels)
	})

	client := testServer(t, mux)
	result, err := client.GetBoardLabels(context.Background(), "b1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Name != "Bug" {
		t.Errorf("name = %q, want %q", result[0].Name, "Bug")
	}
}

func TestAddCardLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1/idLabels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("null"))
	})

	client := testServer(t, mux)
	err := client.AddCardLabel(context.Background(), "c1", "l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveCardLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1/idLabels/l1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("null"))
	})

	client := testServer(t, mux)
	err := client.RemoveCardLabel(context.Background(), "c1", "l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateChecklist(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1/checklists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		cl := Checklist{ID: "cl1", Name: r.URL.Query().Get("name")}
		_ = json.NewEncoder(w).Encode(cl)
	})

	client := testServer(t, mux)
	result, err := client.CreateChecklist(context.Background(), "c1", "Tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Tasks" {
		t.Errorf("Name = %q, want %q", result.Name, "Tasks")
	}
}

func TestCreateList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/boards/b1/lists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		list := List{ID: "l1", Name: r.URL.Query().Get("name"), Pos: 4096}
		_ = json.NewEncoder(w).Encode(list)
	})

	client := testServer(t, mux)
	params := url.Values{}
	params.Set("name", "New List")
	params.Set("pos", "bottom")

	result, err := client.CreateList(context.Background(), "b1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "New List" {
		t.Errorf("Name = %q, want %q", result.Name, "New List")
	}
}

func TestContextCancellation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/members/me/boards", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("[]"))
	})

	client := testServer(t, mux)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetBoards(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGetMember(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/members/me", func(w http.ResponseWriter, _ *http.Request) {
		m := Member{ID: "m1", FullName: "Test User", Username: "testuser"}
		_ = json.NewEncoder(w).Encode(m)
	})

	client := testServer(t, mux)
	result, err := client.GetMember(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Username != "testuser" {
		t.Errorf("Username = %q, want %q", result.Username, "testuser")
	}
}

func TestGetCardBasic(t *testing.T) {
	card := Card{ID: "c1", IDBoard: "b1", IDList: "l1", Name: "Basic Card", Labels: []Label{{ID: "lb1", Name: "Bug", Color: "red"}}}
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		fields := r.URL.Query().Get("fields")
		if fields != "idBoard,idList,name,labels" {
			t.Errorf("fields = %q, want %q", fields, "idBoard,idList,name,labels")
		}
		_ = json.NewEncoder(w).Encode(card)
	})

	client := testServer(t, mux)
	result, err := client.GetCardBasic(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Basic Card" {
		t.Errorf("Name = %q, want %q", result.Name, "Basic Card")
	}
	if result.IDBoard != "b1" {
		t.Errorf("IDBoard = %q, want %q", result.IDBoard, "b1")
	}
	if len(result.Labels) != 1 {
		t.Fatalf("len(Labels) = %d, want 1", len(result.Labels))
	}
	if result.Labels[0].Name != "Bug" {
		t.Errorf("Labels[0].Name = %q, want %q", result.Labels[0].Name, "Bug")
	}
}

func TestGetCardChecklists(t *testing.T) {
	checklists := []Checklist{
		{
			ID:   "cl1",
			Name: "Tasks",
			CheckItems: []CheckItem{
				{ID: "ci1", Name: "Write tests", State: "incomplete", Pos: 1024},
				{ID: "ci2", Name: "Run linter", State: "complete", Pos: 2048},
			},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1/checklists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		checkItemFields := r.URL.Query().Get("checkItem_fields")
		if checkItemFields != "name,state,pos" {
			t.Errorf("checkItem_fields = %q, want %q", checkItemFields, "name,state,pos")
		}
		_ = json.NewEncoder(w).Encode(checklists)
	})

	client := testServer(t, mux)
	result, err := client.GetCardChecklists(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Name != "Tasks" {
		t.Errorf("Name = %q, want %q", result[0].Name, "Tasks")
	}
	if len(result[0].CheckItems) != 2 {
		t.Fatalf("len(CheckItems) = %d, want 2", len(result[0].CheckItems))
	}
	if result[0].CheckItems[0].Name != "Write tests" {
		t.Errorf("CheckItems[0].Name = %q, want %q", result[0].CheckItems[0].Name, "Write tests")
	}
	if result[0].CheckItems[1].State != "complete" {
		t.Errorf("CheckItems[1].State = %q, want %q", result[0].CheckItems[1].State, "complete")
	}
}

func TestUpdateCheckItem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/c1/checkItem/ci1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		item := CheckItem{ID: "ci1", Name: r.URL.Query().Get("name"), State: "complete", Pos: 1024}
		_ = json.NewEncoder(w).Encode(item)
	})

	client := testServer(t, mux)
	params := url.Values{}
	params.Set("name", "Updated Item")
	params.Set("state", "complete")

	result, err := client.UpdateCheckItem(context.Background(), "c1", "ci1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "ci1" {
		t.Errorf("ID = %q, want %q", result.ID, "ci1")
	}
	if result.Name != "Updated Item" {
		t.Errorf("Name = %q, want %q", result.Name, "Updated Item")
	}
	if result.State != "complete" {
		t.Errorf("State = %q, want %q", result.State, "complete")
	}
}

func TestCreateCheckItem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/checklists/cl1/checkItems", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		item := CheckItem{ID: "ci1", Name: r.URL.Query().Get("name"), State: "incomplete", Pos: 1024}
		_ = json.NewEncoder(w).Encode(item)
	})

	client := testServer(t, mux)
	params := url.Values{}
	params.Set("name", "New Item")

	result, err := client.CreateCheckItem(context.Background(), "cl1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "ci1" {
		t.Errorf("ID = %q, want %q", result.ID, "ci1")
	}
	if result.Name != "New Item" {
		t.Errorf("Name = %q, want %q", result.Name, "New Item")
	}
	if result.State != "incomplete" {
		t.Errorf("State = %q, want %q", result.State, "incomplete")
	}
}

func TestGetChecklistBoard(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/checklists/cl1/board", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		fields := r.URL.Query().Get("fields")
		if fields != "name" {
			t.Errorf("fields = %q, want %q", fields, "name")
		}
		_ = json.NewEncoder(w).Encode(Board{ID: "b1", Name: "My Board"})
	})

	client := testServer(t, mux)
	result, err := client.GetChecklistBoard(context.Background(), "cl1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "b1" {
		t.Errorf("ID = %q, want %q", result.ID, "b1")
	}
	if result.Name != "My Board" {
		t.Errorf("Name = %q, want %q", result.Name, "My Board")
	}
}

func TestGetBoardCards(t *testing.T) {
	cards := []Card{
		{ID: "c1", Name: "Card One", IDList: "l1", IDBoard: "b1"},
		{ID: "c2", Name: "Card Two", IDList: "l2", IDBoard: "b1"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/boards/b1/cards", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		filter := r.URL.Query().Get("filter")
		if filter != "open" {
			t.Errorf("filter = %q, want %q", filter, "open")
		}
		_ = json.NewEncoder(w).Encode(cards)
	})

	client := testServer(t, mux)
	result, err := client.GetBoardCards(context.Background(), "b1", "open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].Name != "Card One" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "Card One")
	}
	if result[1].Name != "Card Two" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "Card Two")
	}
}

func TestGetListCards(t *testing.T) {
	cards := []Card{
		{ID: "c1", Name: "List Card One", IDList: "l1"},
		{ID: "c2", Name: "List Card Two", IDList: "l1"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/lists/l1/cards", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		filter := r.URL.Query().Get("filter")
		if filter != "all" {
			t.Errorf("filter = %q, want %q", filter, "all")
		}
		_ = json.NewEncoder(w).Encode(cards)
	})

	client := testServer(t, mux)
	result, err := client.GetListCards(context.Background(), "l1", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].Name != "List Card One" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "List Card One")
	}
	if result[1].Name != "List Card Two" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "List Card Two")
	}
}

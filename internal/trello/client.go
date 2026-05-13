// Package trello provides a thin HTTP client for the Trello REST API.
package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://api.trello.com/1"

// APIError represents a non-2xx response from the Trello API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("trello API error %d: %s", e.StatusCode, e.Body)
}

// Client is a Trello API client that appends key+token to every request.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	token      string
}

// NewClient returns a Client with default HTTP settings.
func NewClient(apiKey, token string) *Client {
	return NewClientWithHTTP(apiKey, token, &http.Client{Timeout: 30 * time.Second}, baseURL)
}

// NewClientWithHTTP returns a Client with a caller-supplied HTTP client and base URL.
func NewClientWithHTTP(apiKey, token string, httpClient *http.Client, base string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(base, "/"),
		apiKey:     apiKey,
		token:      token,
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	u := c.baseURL + path

	if query == nil {
		query = url.Values{}
	}
	query.Set("key", c.apiKey)
	query.Set("token", c.token)

	req, err := http.NewRequestWithContext(ctx, method, u+"?"+query.Encode(), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return c.httpClient.Do(req)
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, target any) error {
	resp, err := c.doRequest(ctx, method, path, query, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(respBody)),
		}
	}

	if target != nil {
		if err := json.Unmarshal(respBody, target); err != nil {
			return fmt.Errorf("decode response JSON: %w", err)
		}
	}

	return nil
}

// GetMember returns the authenticated member's profile.
func (c *Client) GetMember(ctx context.Context) (*Member, error) {
	var m Member
	err := c.doJSON(ctx, http.MethodGet, "/members/me", nil, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetBoards returns all open boards for the authenticated member.
func (c *Client) GetBoards(ctx context.Context) ([]Board, error) {
	q := url.Values{}
	q.Set("filter", "open")
	q.Set("fields", "name,url,shortUrl,dateLastActivity")

	var boards []Board
	err := c.doJSON(ctx, http.MethodGet, "/members/me/boards", q, &boards)
	if err != nil {
		return nil, err
	}
	return boards, nil
}

// GetBoardLists returns all open lists on a board.
func (c *Client) GetBoardLists(ctx context.Context, boardID string) ([]List, error) {
	q := url.Values{}
	q.Set("filter", "open")
	q.Set("fields", "name,pos")

	var lists []List
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/boards/%s/lists", boardID), q, &lists)
	if err != nil {
		return nil, err
	}
	return lists, nil
}

// GetBoardCards returns cards on a board with summary fields.
func (c *Client) GetBoardCards(ctx context.Context, boardID string) ([]Card, error) {
	q := url.Values{}
	q.Set("fields", "name,due,dueComplete,idList,labels,shortUrl,dateLastActivity,closed")
	q.Set("members", "true")
	q.Set("member_fields", "fullName,username")

	var cards []Card
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/boards/%s/cards", boardID), q, &cards)
	if err != nil {
		return nil, err
	}
	return cards, nil
}

// GetListCards returns cards in a specific list with summary fields.
func (c *Client) GetListCards(ctx context.Context, listID string) ([]Card, error) {
	q := url.Values{}
	q.Set("fields", "name,due,dueComplete,idList,labels,shortUrl,dateLastActivity,closed")
	q.Set("members", "true")
	q.Set("member_fields", "fullName,username")

	var cards []Card
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/lists/%s/cards", listID), q, &cards)
	if err != nil {
		return nil, err
	}
	return cards, nil
}

// GetCard returns full card detail including checklists, comments, and attachments.
func (c *Client) GetCard(ctx context.Context, cardID string) (*Card, error) {
	q := url.Values{}
	q.Set("checklists", "all")
	q.Set("checklist_fields", "name")
	q.Set("members", "true")
	q.Set("member_fields", "fullName,username")
	q.Set("attachments", "true")
	q.Set("attachment_fields", "name,url,date")
	q.Set("actions", "commentCard")
	q.Set("actions_limit", "10")
	q.Set("action_fields", "data,date,memberCreator")
	q.Set("action_memberCreator_fields", "fullName,username")

	var card Card
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/cards/%s", cardID), q, &card)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

// CreateCard creates a new card with the given parameters.
func (c *Client) CreateCard(ctx context.Context, params url.Values) (*Card, error) {
	var card Card
	err := c.doJSON(ctx, http.MethodPost, "/cards", params, &card)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

// UpdateCard updates fields on an existing card.
func (c *Client) UpdateCard(ctx context.Context, cardID string, params url.Values) (*Card, error) {
	var card Card
	err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("/cards/%s", cardID), params, &card)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

// GetBoardLabels returns all labels defined on a board.
func (c *Client) GetBoardLabels(ctx context.Context, boardID string) ([]Label, error) {
	q := url.Values{}
	q.Set("fields", "name,color")

	var labels []Label
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/boards/%s/labels", boardID), q, &labels)
	if err != nil {
		return nil, err
	}
	return labels, nil
}

// GetBoardName returns the name of a board by ID.
func (c *Client) GetBoardName(ctx context.Context, boardID string) (string, error) {
	q := url.Values{}
	q.Set("fields", "name")

	var board Board
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/boards/%s", boardID), q, &board)
	if err != nil {
		return "", err
	}
	return board.Name, nil
}

// GetListName returns the name of a list by ID.
func (c *Client) GetListName(ctx context.Context, listID string) (string, error) {
	q := url.Values{}
	q.Set("fields", "name")

	var list List
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/lists/%s", listID), q, &list)
	if err != nil {
		return "", err
	}
	return list.Name, nil
}

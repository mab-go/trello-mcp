package trello

import "time"

// Member represents a Trello member (user account).
type Member struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Username string `json:"username"`
}

// Board represents a Trello board.
type Board struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	URL              string `json:"url"`
	ShortURL         string `json:"shortUrl"`
	DateLastActivity string `json:"dateLastActivity"`
}

// List represents a Trello list on a board.
type List struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Pos  float64 `json:"pos"`
}

// Card represents a Trello card.
type Card struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Desc             string       `json:"desc"`
	Due              *time.Time   `json:"due"`
	DueComplete      bool         `json:"dueComplete"`
	IDList           string       `json:"idList"`
	IDBoard          string       `json:"idBoard"`
	Labels           []Label      `json:"labels"`
	ShortURL         string       `json:"shortUrl"`
	DateLastActivity string       `json:"dateLastActivity"`
	Closed           bool         `json:"closed"`
	Members          []CardMember `json:"members"`
	Checklists       []Checklist  `json:"checklists"`
	Actions          []Action     `json:"actions"`
	Attachments      []Attachment `json:"attachments"`
	URL              string       `json:"url"`
}

// CardMember represents a member assigned to a card.
type CardMember struct {
	FullName string `json:"fullName"`
	Username string `json:"username"`
}

// Label represents a label on a Trello board.
type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Checklist represents a checklist attached to a card.
type Checklist struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	CheckItems []CheckItem `json:"checkItems"`
}

// CheckItem represents a single item in a checklist.
type CheckItem struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	State string  `json:"state"`
	Pos   float64 `json:"pos"`
}

// Action represents a card action (e.g. a comment).
type Action struct {
	ID            string       `json:"id"`
	Type          string       `json:"type"`
	Data          ActionData   `json:"data"`
	Date          string       `json:"date"`
	MemberCreator ActionMember `json:"memberCreator"`
}

// ActionData holds the data payload of an action.
type ActionData struct {
	Text string `json:"text"`
}

// ActionMember identifies the member who created an action.
type ActionMember struct {
	FullName string `json:"fullName"`
	Username string `json:"username"`
}

// Attachment represents a file or URL attached to a card.
type Attachment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Date string `json:"date"`
}

// SearchResult holds results from a Trello search query.
type SearchResult struct {
	Cards  []Card  `json:"cards"`
	Boards []Board `json:"boards"`
}

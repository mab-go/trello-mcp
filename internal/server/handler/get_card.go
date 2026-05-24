package handler

import (
	"context"
	"time"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type checkItemDetail struct {
	ItemName string `json:"item_name"`
	Complete bool   `json:"complete"`
}

type checklistDetail struct {
	ChecklistName string            `json:"checklist_name"`
	Items         []checkItemDetail `json:"items"`
}

type commentDetail struct {
	Author string `json:"author"`
	Text   string `json:"text"`
	Date   string `json:"date"`
}

type attachmentDetail struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Date string `json:"date"`
}

type getCardResponse struct {
	CardID      string             `json:"card_id"`
	Name        string             `json:"name"`
	List        string             `json:"list"`
	Board       string             `json:"board"`
	URL         string             `json:"url"`
	Description string             `json:"description,omitempty"`
	Due         string             `json:"due,omitempty"`
	DueComplete bool               `json:"due_complete"`
	Labels      []string           `json:"labels"`
	Members     []string           `json:"members"`
	Checklists  []checklistDetail  `json:"checklists,omitempty"`
	Comments    []commentDetail    `json:"comments,omitempty"`
	Attachments []attachmentDetail `json:"attachments,omitempty"`
}

// GetCard handles the trello_get_card tool.
func (h *TrelloHandler) GetCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_get_card")

	args := req.GetArguments()
	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	card, err := h.client.GetCard(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}

	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	listName, err := h.client.GetListName(ctx, card.IDList)
	if err != nil {
		return mapAPIError(log, err)
	}

	boardName, err := h.client.GetBoardName(ctx, card.IDBoard)
	if err != nil {
		return mapAPIError(log, err)
	}

	log.WithField("card_id", cardID).Info(eventCardGet)

	return jsonResult(buildGetCardResponse(card, listName, boardName))
}

func buildGetCardResponse(card *trello.Card, listName, boardName string) getCardResponse {
	resp := getCardResponse{
		CardID:      card.ID,
		Name:        card.Name,
		List:        listName,
		Board:       boardName,
		URL:         card.ShortURL,
		Description: card.Desc,
		DueComplete: card.DueComplete,
	}

	if card.Due != nil {
		resp.Due = card.Due.UTC().Format("2006-01-02")
	}

	resp.Labels = labelNames(card.Labels)
	resp.Members = memberUsernames(card.Members)
	resp.Checklists = buildChecklists(card.Checklists)
	resp.Comments = buildComments(card.Actions)
	resp.Attachments = buildAttachments(card.Attachments)

	return resp
}

func labelNames(labels []trello.Label) []string {
	names := make([]string, 0, len(labels))
	for _, l := range labels {
		if l.Name != "" {
			names = append(names, l.Name)
		}
	}
	return names
}

func memberUsernames(members []trello.CardMember) []string {
	usernames := make([]string, 0, len(members))
	for _, m := range members {
		usernames = append(usernames, m.Username)
	}
	return usernames
}

func buildChecklists(checklists []trello.Checklist) []checklistDetail {
	result := make([]checklistDetail, 0, len(checklists))
	for _, cl := range checklists {
		items := make([]checkItemDetail, len(cl.CheckItems))
		for j, ci := range cl.CheckItems {
			items[j] = checkItemDetail{
				ItemName: ci.Name,
				Complete: ci.State == "complete",
			}
		}
		result = append(result, checklistDetail{
			ChecklistName: cl.Name,
			Items:         items,
		})
	}
	return result
}

func buildComments(actions []trello.Action) []commentDetail {
	result := make([]commentDetail, 0, len(actions))
	for _, a := range actions {
		result = append(result, commentDetail{
			Author: a.MemberCreator.Username,
			Text:   a.Data.Text,
			Date:   formatDateTime(a.Date),
		})
	}
	return result
}

func buildAttachments(attachments []trello.Attachment) []attachmentDetail {
	result := make([]attachmentDetail, 0, len(attachments))
	for _, att := range attachments {
		result = append(result, attachmentDetail{
			Name: att.Name,
			URL:  att.URL,
			Date: formatDateTime(att.Date),
		})
	}
	return result
}

func formatDateTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04")
}

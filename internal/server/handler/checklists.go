package handler

import (
	"context"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type checklistItemEntry struct {
	ItemID   string `json:"item_id"`
	Name     string `json:"name"`
	Complete bool   `json:"complete"`
}

type checklistEntry struct {
	ChecklistID string               `json:"checklist_id"`
	Name        string               `json:"name"`
	Items       []checklistItemEntry `json:"items"`
}

type checklistsResponse struct {
	CardID     string           `json:"card_id"`
	Count      int              `json:"count"`
	Checklists []checklistEntry `json:"checklists"`
}

// Checklists handles the trello_checklists tool.
func (h *TrelloHandler) Checklists(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_checklists")

	args := req.GetArguments()

	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	card, err := h.client.GetCardBasic(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}
	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	checklists, err := h.client.GetCardChecklists(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}

	entries := buildChecklistEntries(checklists)

	log.WithFields(logging.Fields{"card_id": cardID, "count": len(entries)}).Info(eventChecklistList)

	return jsonResult(checklistsResponse{
		CardID:     cardID,
		Count:      len(entries),
		Checklists: entries,
	})
}

func buildChecklistEntries(checklists []trello.Checklist) []checklistEntry {
	entries := make([]checklistEntry, len(checklists))
	for i, cl := range checklists {
		items := make([]checklistItemEntry, len(cl.CheckItems))
		for j, ci := range cl.CheckItems {
			items[j] = checklistItemEntry{
				ItemID:   ci.ID,
				Name:     ci.Name,
				Complete: ci.State == "complete",
			}
		}
		entries[i] = checklistEntry{
			ChecklistID: cl.ID,
			Name:        cl.Name,
			Items:       items,
		}
	}
	return entries
}

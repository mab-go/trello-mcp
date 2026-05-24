package handler

import (
	"context"
	"net/url"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type updateCardResponse struct {
	CardID      string  `json:"card_id"`
	Name        string  `json:"name"`
	List        string  `json:"list"`
	URL         string  `json:"url"`
	Description *string `json:"description,omitempty"`
	Due         *string `json:"due,omitempty"`
	DueComplete *bool   `json:"due_complete,omitempty"`
	Position    *string `json:"position,omitempty"`
}

// UpdateCard handles the trello_update_card tool.
func (h *TrelloHandler) UpdateCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_update_card")

	args := req.GetArguments()
	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	params, hasUpdate := buildUpdateParams(args)

	listUpdated, errResult, err := h.resolveListMove(ctx, log, args, cardID, params)
	if err != nil {
		return nil, err
	}
	if errResult != nil {
		return errResult, nil
	}
	hasUpdate = hasUpdate || listUpdated

	if !hasUpdate {
		return mcp.NewToolResultError(
			"at least one field to update is required (name, description, due, " +
				"due_complete, list_id, list_name, position)",
		), nil
	}

	existingCard, err := h.client.GetCard(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}
	if errResult := h.checkAllowedBoard(existingCard.IDBoard); errResult != nil {
		return errResult, nil
	}

	updated, err := h.client.UpdateCard(ctx, cardID, params)
	if err != nil {
		return mapAPIError(log, err)
	}

	resp := buildUpdateResponse(args, updated, h.listDisplayName(ctx, updated.IDList))

	log.WithFields(logging.Fields{"card_id": updated.ID, "name": updated.Name}).Info(eventCardUpdate)

	return jsonResult(resp)
}

func buildUpdateParams(args map[string]any) (url.Values, bool) {
	params := url.Values{}
	hasUpdate := false

	if name, ok := args["name"].(string); ok {
		params.Set("name", name)
		hasUpdate = true
	}
	if desc, ok := args["description"].(string); ok {
		params.Set("desc", desc)
		hasUpdate = true
	}
	if due, ok := args["due"].(string); ok {
		params.Set("due", due)
		hasUpdate = true
	}
	if dc, ok := args["due_complete"].(bool); ok {
		if dc {
			params.Set("dueComplete", "true")
		} else {
			params.Set("dueComplete", "false")
		}
		hasUpdate = true
	}
	if pos, ok := args["position"].(string); ok {
		params.Set("pos", pos)
		hasUpdate = true
	}

	return params, hasUpdate
}

func (h *TrelloHandler) resolveListMove(
	ctx context.Context,
	log logging.Logger,
	args map[string]any,
	cardID string,
	params url.Values,
) (bool, *mcp.CallToolResult, error) {
	listID, hasListID := args["list_id"].(string)
	listName, hasListName := args["list_name"].(string)

	if hasListID && listID != "" {
		params.Set("idList", listID)
		return true, nil, nil
	}
	if !hasListName || listName == "" {
		return false, nil, nil
	}

	boardID, errResult, err := h.resolveBoardForListMove(ctx, log, args, cardID)
	if err != nil {
		return false, nil, err
	}
	if errResult != nil {
		return false, errResult, nil
	}

	lists, listsErr := h.client.GetBoardLists(ctx, boardID)
	if listsErr != nil {
		result, apiErr := mapAPIError(log, listsErr)
		return false, result, apiErr
	}

	resolvedID, resolveErr := h.resolveListByName(lists, listName)
	if resolveErr != nil {
		return false, resolveErr, nil
	}
	params.Set("idList", resolvedID)
	return true, nil, nil
}

func buildUpdateResponse(args map[string]any, updated *trello.Card, listName string) updateCardResponse {
	resp := updateCardResponse{
		CardID: updated.ID,
		Name:   updated.Name,
		List:   listName,
		URL:    updated.ShortURL,
	}
	if _, ok := args["description"].(string); ok {
		resp.Description = &updated.Desc
	}
	if _, ok := args["due"].(string); ok {
		var dueStr string
		if updated.Due != nil {
			dueStr = updated.Due.UTC().Format("2006-01-02")
		}
		resp.Due = &dueStr
	}
	if _, ok := args["due_complete"].(bool); ok {
		resp.DueComplete = &updated.DueComplete
	}
	if pos, ok := args["position"].(string); ok {
		resp.Position = &pos
	}
	return resp
}

func (h *TrelloHandler) resolveBoardForListMove(
	ctx context.Context,
	log logging.Logger,
	args map[string]any,
	cardID string,
) (string, *mcp.CallToolResult, error) {
	boardID, _ := args["board_id"].(string)
	if boardID != "" {
		return boardID, nil, nil
	}

	card, err := h.client.GetCard(ctx, cardID)
	if err != nil {
		result, apiErr := mapAPIError(log, err)
		return "", result, apiErr
	}
	if card.IDBoard != "" {
		return card.IDBoard, nil, nil
	}

	if h.config.DefaultBoard != "" {
		return h.config.DefaultBoard, nil, nil
	}

	return "", mcp.NewToolResultError(
		"board_id is required when using list_name (no default_board configured)",
	), nil
}

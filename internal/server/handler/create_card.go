package handler

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type createCardResponse struct {
	CardID string `json:"card_id"`
	Name   string `json:"name"`
	List   string `json:"list"`
	URL    string `json:"url"`
}

// CreateCard handles the trello_create_card tool.
func (h *TrelloHandler) CreateCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_create_card")

	args := req.GetArguments()

	boardID, errResult := h.resolveBoardID(args)
	if errResult != nil {
		return errResult, nil
	}

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("missing required argument: name"), nil
	}

	listID, _ := args["list_id"].(string)
	listName, _ := args["list_name"].(string)

	if errResult := validateListArgs(listID, listName); errResult != nil {
		return errResult, nil
	}

	lists, err := h.client.GetBoardLists(ctx, boardID)
	if err != nil {
		return mapAPIError(log, err)
	}

	listID, resolvedListName, errResult := resolveListIDAndName(h, lists, listID, listName)
	if errResult != nil {
		return errResult, nil
	}

	labelIDs, errResult, err := h.resolveLabels(ctx, log, args, boardID)
	if errResult != nil {
		return errResult, nil
	}
	if err != nil {
		return nil, err
	}

	params := buildCreateParams(listID, name, labelIDs, args)

	card, err := h.client.CreateCard(ctx, params)
	if err != nil {
		return mapAPIError(log, err)
	}

	log.WithFields(logging.Fields{"card_id": card.ID, "name": card.Name, "list": resolvedListName}).Info(eventCardCreate)

	return jsonResult(createCardResponse{
		CardID: card.ID,
		Name:   card.Name,
		List:   resolvedListName,
		URL:    card.ShortURL,
	})
}

func validateListArgs(listID, listName string) *mcp.CallToolResult {
	if listID == "" && listName == "" {
		return mcp.NewToolResultError(
			"one of list_id or list_name is required. " +
				"Use trello_lists to see available lists.",
		)
	}
	if listID != "" && listName != "" {
		return mcp.NewToolResultError(
			"provide either list_id or list_name, not both",
		)
	}
	return nil
}

func resolveListIDAndName(h *TrelloHandler, lists []trello.List, listID, listName string) (string, string, *mcp.CallToolResult) {
	if listName != "" {
		id, errResult := h.resolveListByName(lists, listName)
		if errResult != nil {
			return "", "", errResult
		}
		return id, listName, nil
	}
	for _, l := range lists {
		if l.ID == listID {
			return listID, l.Name, nil
		}
	}
	return listID, "", nil
}

func (h *TrelloHandler) resolveLabels(
	ctx context.Context,
	log logging.Logger,
	args map[string]any,
	boardID string,
) ([]string, *mcp.CallToolResult, error) {
	labelsRaw, ok := args["labels"]
	if !ok {
		return nil, nil, nil
	}

	labelNames, ok := toStringSlice(labelsRaw)
	if !ok {
		return nil, mcp.NewToolResultError("labels must be an array of strings"), nil
	}
	if len(labelNames) == 0 {
		return nil, nil, nil
	}

	boardLabels, err := h.client.GetBoardLabels(ctx, boardID)
	if err != nil {
		result, apiErr := mapAPIError(log, err)
		return nil, result, apiErr
	}

	labelIDs := make([]string, 0, len(labelNames))
	for _, ln := range labelNames {
		id, errResult := matchLabel(boardLabels, ln)
		if errResult != nil {
			return nil, errResult, nil
		}
		labelIDs = append(labelIDs, id)
	}
	return labelIDs, nil, nil
}

func matchLabel(boardLabels []trello.Label, name string) (string, *mcp.CallToolResult) {
	for _, bl := range boardLabels {
		if strings.EqualFold(bl.Name, name) {
			return bl.ID, nil
		}
	}
	available := make([]string, 0, len(boardLabels))
	for _, bl := range boardLabels {
		if bl.Name != "" {
			available = append(available, bl.Name)
		}
	}
	return "", mcp.NewToolResultError(
		fmt.Sprintf("label %q not found. Available labels: %s",
			name, strings.Join(available, ", ")),
	)
}

func buildCreateParams(listID, name string, labelIDs []string, args map[string]any) url.Values {
	params := url.Values{}
	params.Set("idList", listID)
	params.Set("name", name)

	if desc, ok := args["description"].(string); ok && desc != "" {
		params.Set("desc", desc)
	}
	if due, ok := args["due"].(string); ok && due != "" {
		params.Set("due", due)
	}
	if len(labelIDs) > 0 {
		params.Set("idLabels", strings.Join(labelIDs, ","))
	}

	pos, _ := args["position"].(string)
	if pos == "" {
		pos = "bottom"
	}
	params.Set("pos", pos)

	return params
}

func toStringSlice(v any) ([]string, bool) {
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		result[i] = s
	}
	return result, true
}

package handler

import (
	"context"
	"net/url"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

type moveCardResponse struct {
	CardID    string `json:"card_id"`
	Name      string `json:"name"`
	FromBoard string `json:"from_board"`
	FromList  string `json:"from_list"`
	ToBoard   string `json:"to_board"`
	ToList    string `json:"to_list"`
	URL       string `json:"url"`
}

// MoveCard handles the trello_move_card tool.
func (h *TrelloHandler) MoveCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log, _ := logging.FromContext(ctx)
	log = log.WithField("tool", "trello_move_card")

	args := req.GetArguments()

	cardID, _ := args["card_id"].(string)
	if cardID == "" {
		return mcp.NewToolResultError("missing required argument: card_id"), nil
	}

	targetListID, _ := args["target_list_id"].(string)
	targetListName, _ := args["target_list_name"].(string)

	if errResult := validateTargetListArgs(targetListID, targetListName); errResult != nil {
		return errResult, nil
	}

	card, err := h.client.GetCardBasic(ctx, cardID)
	if err != nil {
		return mapAPIError(log, err)
	}
	if errResult := h.checkAllowedBoard(card.IDBoard); errResult != nil {
		return errResult, nil
	}

	targetBoardID, crossBoard, errResult := h.resolveTargetBoard(args, card.IDBoard)
	if errResult != nil {
		return errResult, nil
	}

	targetListID, errResult, err = h.resolveTargetList(ctx, log, targetListID, targetListName, targetBoardID)
	if errResult != nil {
		return errResult, nil
	}
	if err != nil {
		return nil, err
	}

	params := buildMoveParams(targetListID, targetBoardID, crossBoard, args)

	updated, err := h.client.UpdateCard(ctx, cardID, params)
	if err != nil {
		return mapAPIError(log, err)
	}

	log.WithFields(logging.Fields{"card_id": updated.ID, "target_list": targetListID}).Info(eventCardMove)

	return h.buildMoveResponse(ctx, log, updated, card, targetBoardID, targetListID, crossBoard)
}

func (h *TrelloHandler) resolveTargetBoard(args map[string]any, sourceBoardID string) (string, bool, *mcp.CallToolResult) {
	targetBoardID, _ := args["target_board"].(string)
	crossBoard := targetBoardID != "" && targetBoardID != sourceBoardID
	if targetBoardID == "" {
		targetBoardID = sourceBoardID
	}
	if crossBoard {
		if errResult := h.checkAllowedBoard(targetBoardID); errResult != nil {
			return "", false, errResult
		}
	}
	return targetBoardID, crossBoard, nil
}

func (h *TrelloHandler) resolveTargetList(
	ctx context.Context,
	log logging.Logger,
	listID,
	listName,
	boardID string,
) (string, *mcp.CallToolResult, error) {
	if listName == "" {
		return listID, nil, nil
	}
	lists, err := h.client.GetBoardLists(ctx, boardID)
	if err != nil {
		result, apiErr := mapAPIError(log, err)
		return "", result, apiErr
	}
	resolved, errResult := h.resolveListByName(lists, listName)
	if errResult != nil {
		return "", errResult, nil
	}
	return resolved, nil, nil
}

func buildMoveParams(targetListID, targetBoardID string, crossBoard bool, args map[string]any) url.Values {
	params := url.Values{}
	params.Set("idList", targetListID)
	if crossBoard {
		params.Set("idBoard", targetBoardID)
	}
	if pos, _ := args["position"].(string); pos != "" {
		params.Set("pos", pos)
	}
	return params
}

func (h *TrelloHandler) buildMoveResponse(
	ctx context.Context,
	log logging.Logger,
	updated *trello.Card,
	source *trello.Card,
	targetBoardID,
	targetListID string,
	crossBoard bool,
) (*mcp.CallToolResult, error) {
	fromBoardName, err := h.client.GetBoardName(ctx, source.IDBoard)
	if err != nil {
		return mapAPIError(log, err)
	}
	fromListName := h.listDisplayName(ctx, source.IDList)

	toBoardName := fromBoardName
	if crossBoard {
		toBoardName, err = h.client.GetBoardName(ctx, targetBoardID)
		if err != nil {
			return mapAPIError(log, err)
		}
	}
	toListName := h.listDisplayName(ctx, targetListID)

	return jsonResult(moveCardResponse{
		CardID:    updated.ID,
		Name:      updated.Name,
		FromBoard: fromBoardName,
		FromList:  fromListName,
		ToBoard:   toBoardName,
		ToList:    toListName,
		URL:       updated.ShortURL,
	})
}

func validateTargetListArgs(listID, listName string) *mcp.CallToolResult {
	if listID == "" && listName == "" {
		return mcp.NewToolResultError(
			"one of target_list_id or target_list_name is required. " +
				"Use trello_lists to see available lists.",
		)
	}
	if listID != "" && listName != "" {
		return mcp.NewToolResultError(
			"provide either target_list_id or target_list_name, not both",
		)
	}
	return nil
}

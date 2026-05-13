// Package handler implements MCP tool handlers for Trello operations.
package handler

import (
	"errors"
	"fmt"

	"github.com/mab-go/trello-mcp/internal/trello"

	"github.com/mark3labs/mcp-go/mcp"
)

func mapAPIError(err error) (*mcp.CallToolResult, error) {
	var apiErr *trello.APIError
	if !errors.As(err, &apiErr) {
		return nil, err
	}

	switch apiErr.StatusCode {
	case 400:
		return mcp.NewToolResultError(
			fmt.Sprintf("Trello rejected the request (400): %s", apiErr.Body),
		), nil
	case 401:
		return mcp.NewToolResultError(
			"Authentication failed. Verify your API key and token in " +
				"~/.config/trello-mcp/config.json.",
		), nil
	case 403:
		return mcp.NewToolResultError(
			"Access denied (403). Verify you have access to this board/card.",
		), nil
	case 404:
		return mcp.NewToolResultError(
			"Resource not found (404). Verify the board/card/list ID.",
		), nil
	case 429:
		return mcp.NewToolResultError(
			"Trello API rate limit exceeded (429). Wait a few seconds and retry.",
		), nil
	default:
		return nil, err
	}
}

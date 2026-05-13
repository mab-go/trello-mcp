package server

import (
	"context"
	"fmt"

	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/logging"
	"github.com/mab-go/trello-mcp/internal/server/handler"
	"github.com/mab-go/trello-mcp/internal/trello"
	"github.com/mab-go/trello-mcp/internal/version"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func newTrelloServer(ctx context.Context, h *handler.TrelloHandler) *mcpserver.MCPServer {
	log, _ := logging.FromContext(ctx)

	hooks := &mcpserver.Hooks{}
	hooks.AddBeforeInitialize(hookAddBeforeInitialize(log))
	hooks.AddAfterInitialize(hookAddAfterInitialize(log))

	s := mcpserver.NewMCPServer(
		"trello-mcp",
		version.Version,
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithLogging(),
		mcpserver.WithHooks(hooks),
	)

	s.AddTool(toolTrelloBoards, h.Boards)
	s.AddTool(toolTrelloLists, h.Lists)
	s.AddTool(toolTrelloCards, h.Cards)
	s.AddTool(toolTrelloGetCard, h.GetCard)
	s.AddTool(toolTrelloCreateCard, h.CreateCard)
	s.AddTool(toolTrelloUpdateCard, h.UpdateCard)
	s.AddTool(toolTrelloArchiveCard, h.ArchiveCard)
	s.AddTool(toolTrelloUnarchiveCard, h.UnarchiveCard)

	return s
}

// RunStdioServer loads config, creates a Trello client, and serves over stdio.
func RunStdioServer() error {
	log := logging.NewDefaultLogger()
	ctx := logging.NewContext(context.Background(), log)

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	client := trello.NewClient(cfg.APIKey, cfg.Token)
	h := handler.NewTrelloHandler(client, cfg)
	srv := newTrelloServer(ctx, h)

	return mcpserver.ServeStdio(srv)
}

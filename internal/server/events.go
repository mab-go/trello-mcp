package server

import "github.com/mab-go/logging"

const (
	eventConfigLoad  logging.Event = "config.load"  // Configuration loaded
	eventServerInit  logging.Event = "server.init"  // Initialized server
	eventServerStart logging.Event = "server.start" // Starting MCP server
)

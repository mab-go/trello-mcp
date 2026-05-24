// Package main is the entry point for the trello-mcp CLI.
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/config"
	"github.com/mab-go/trello-mcp/internal/server"
	"github.com/mab-go/trello-mcp/internal/trello"
	"github.com/mab-go/trello-mcp/internal/version"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const eventCmdFail logging.Event = "cmd.fail" // Failed to execute root command

var (
	cmd = &cobra.Command{
		Use:     "trello-mcp",
		Short:   "Trello MCP Server",
		Long:    "An MCP server for managing Trello boards, lists, and cards.",
		Version: fmt.Sprintf("trello-mcp %s (%s; %s)", version.Version, version.ShortCommit(), version.Date),
		RunE: func(_ *cobra.Command, _ []string) error {
			return server.RunStdioServer()
		},
	}

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server (stdio transport)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return server.RunStdioServer()
		},
	}

	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Validate Trello API credentials",
		RunE:  runAuth,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information and exit",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("trello-mcp %s (%s; %s)\n", version.Version, version.ShortCommit(), version.Date)
		},
	}
)

func runAuth(cmd *cobra.Command, _ []string) error {
	status, _ := cmd.Flags().GetBool("status")

	if status {
		return runAuthStatus()
	}
	return runAuthValidate()
}

func runAuthStatus() error {
	path, err := config.Path()
	if err != nil {
		fmt.Printf("Cannot determine config path: %v\n", err)
		return nil
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Not configured: %v\n", err)
		return nil
	}

	fmt.Printf("Config: %s\n", path)
	fmt.Printf("  API key: present\n")
	fmt.Printf("  Token: present\n")
	if cfg.DefaultBoard != "" {
		fmt.Printf("  Default board: %s\n", cfg.DefaultBoard)
	}
	if len(cfg.AllowedBoards) > 0 {
		fmt.Printf("  Allowed boards: %v\n", cfg.AllowedBoards)
	}
	return nil
}

func runAuthValidate() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := trello.NewClient(cfg.APIKey, cfg.Token)
	member, err := client.GetMember(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Authenticated as %s (@%s)\n", member.FullName, member.Username)
	return nil
}

func init() {
	cmd.SetGlobalNormalizationFunc(wordSepNormalizeFunc)
	cmd.SetVersionTemplate("{{.Version}}\n")

	authCmd.Flags().Bool("status", false, "Check config file state without calling the API")

	cmd.AddCommand(serveCmd)
	cmd.AddCommand(authCmd)
	cmd.AddCommand(versionCmd)
}

func wordSepNormalizeFunc(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	name = strings.ReplaceAll(name, "_", "-")
	return pflag.NormalizedName(name)
}

func main() {
	if err := cmd.Execute(); err != nil {
		logging.WithError(err).Fatal(eventCmdFail)
	}
}

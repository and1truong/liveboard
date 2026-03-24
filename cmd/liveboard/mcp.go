package main

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	livemcp "github.com/and1truong/liveboard/internal/mcp"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server over stdio (for Claude Code / AI assistant integration)",
		RunE: func(_ *cobra.Command, _ []string) error {
			srv := livemcp.New(ws, eng, version)
			return srv.MCP().Run(context.Background(), &mcpsdk.StdioTransport{})
		},
	}
}

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/api"
)

func serveCmd() *cobra.Command {
	var (
		host     string
		port     int
		readOnly bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the REST API and Web UI server",
		RunE: func(c *cobra.Command, _ []string) error {
			if usingCloud && !c.Flags().Changed("port") {
				port = 7777
			}
			noCache := os.Getenv("NO_CACHE") != ""
			srv := api.NewServer(ws, ws.Engine, noCache, readOnly, false, version)
			addr := fmt.Sprintf("%s:%d", host, port)
			fmt.Printf("LiveBoard Web UI: http://%s:%d\n", host, port)
			fmt.Printf("REST API: http://%s:%d/boards\n", host, port)
			fmt.Printf("MCP: http://%s:%d/mcp\n", host, port)
			return srv.Start(addr)
		},
	}

	defaultHost := "127.0.0.1"
	if v := os.Getenv("LIVEBOARD_HOST"); v != "" {
		defaultHost = v
	}

	defaultPort := 7070
	if v := os.Getenv("LIVEBOARD_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			defaultPort = p
		}
	}

	cmd.Flags().StringVar(&host, "host", defaultHost, "server host")
	cmd.Flags().IntVarP(&port, "port", "p", defaultPort, "server port")
	cmd.Flags().BoolVar(&readOnly, "readonly", false, "start in read-only mode (no writes allowed)")
	return cmd
}

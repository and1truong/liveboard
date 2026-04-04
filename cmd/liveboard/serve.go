package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/api"
	"github.com/and1truong/liveboard/internal/defaults"
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
		RunE: func(_ *cobra.Command, _ []string) error {
			noCache := os.Getenv("NO_CACHE") != ""
			srv := api.NewServer(ws, ws.Engine, noCache, readOnly, false, version)
			addr := fmt.Sprintf("%s:%d", host, port)
			fmt.Printf("LiveBoard Web UI: http://%s:%d\n", host, port)
			fmt.Printf("REST API: http://%s:%d/boards\n", host, port)
			fmt.Printf("MCP: http://%s:%d/mcp\n", host, port)
			return srv.Start(addr)
		},
	}

	cfg := defaults.LoadCLIConfig()

	// Priority: flags > env vars > config file > hardcoded defaults
	defaultHost := "127.0.0.1"
	if cfg.Host != "" {
		defaultHost = cfg.Host
	}
	if v := os.Getenv("LIVEBOARD_HOST"); v != "" {
		defaultHost = v
	}

	defaultPort := 7070
	if cfg.Port != 0 {
		defaultPort = cfg.Port
	}
	if v := os.Getenv("LIVEBOARD_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			defaultPort = p
		}
	}

	defaultReadOnly := cfg.ReadOnly

	cmd.Flags().StringVar(&host, "host", defaultHost, "server host")
	cmd.Flags().IntVarP(&port, "port", "p", defaultPort, "server port")
	cmd.Flags().BoolVar(&readOnly, "readonly", defaultReadOnly, "start in read-only mode (no writes allowed)")
	return cmd
}

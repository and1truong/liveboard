package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/api"
)

func serveCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the REST API and Web UI server",
		RunE: func(_ *cobra.Command, _ []string) error {
			srv := api.NewServer(ws, ws.Engine, gitRepo)
			addr := fmt.Sprintf(":%d", port)
			fmt.Printf("LiveBoard Web UI: http://localhost:%d\n", port)
			fmt.Printf("REST API: http://localhost:%d/boards\n", port)
			return srv.Start(addr)
		},
	}

	defaultPort := 7070
	if v := os.Getenv("LIVEBOARD_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			defaultPort = p
		}
	}

	cmd.Flags().IntVarP(&port, "port", "p", defaultPort, "server port")
	return cmd
}

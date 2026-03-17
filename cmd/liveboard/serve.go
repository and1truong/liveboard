package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/and1truong/liveboard/internal/api"
	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the REST API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := api.NewServer(ws, ws.Engine, gitRepo)
			addr := fmt.Sprintf(":%d", port)
			fmt.Printf("LiveBoard API listening on http://localhost:%d\n", port)
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

package main

import (
	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/export"
	"github.com/and1truong/liveboard/internal/web"
)

func exportCmd() *cobra.Command {
	var outputDir string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export workspace to static HTML files",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := web.LoadSettingsFromDir(ws.Dir)
			opts := export.Options{
				Theme:      s.Theme,
				ColorTheme: s.ColorTheme,
				SiteName:   s.SiteName,
			}
			return export.Run(ws, outputDir, opts)
		},
	}
	cmd.Flags().StringVarP(&outputDir, "output", "o", "./export", "output directory")
	return cmd
}

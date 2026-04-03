package web

import (
	"net/http"

	"github.com/and1truong/liveboard/internal/export"
)

// exportHandler returns an HTTP handler function that exports the workspace as a ZIP of static HTML files.
func exportHandler(b *Base) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		settings := b.loadSettings()
		opts := export.Options{
			Theme:      settings.Theme,
			ColorTheme: settings.ColorTheme,
			SiteName:   settings.SiteName,
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="liveboard-export.zip"`)
		if err := export.WriteZipTo(w, b.ws, opts); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

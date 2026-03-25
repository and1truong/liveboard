package web

import (
	"net/http"

	"github.com/and1truong/liveboard/internal/export"
)

// ExportHandler returns an HTTP handler that exports the workspace as a ZIP of static HTML files.
func (h *Handler) ExportHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		settings := h.loadSettings()
		opts := export.Options{
			Theme:      settings.Theme,
			ColorTheme: settings.ColorTheme,
			SiteName:   settings.SiteName,
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="liveboard-export.zip"`)
		if err := export.WriteZipTo(w, h.ws, opts); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

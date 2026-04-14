package v1

import (
	"encoding/json"
	"net/http"
)

// VersionsHandler returns the /api/versions probe handler.
// Mounted at /api/versions by the parent server.
func VersionsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Supported []string `json:"supported"`
			Current   string   `json:"current"`
		}{
			Supported: []string{"v1"},
			Current:   "v1",
		})
	})
}

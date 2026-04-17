package v1

import (
	"encoding/json"
	"net/http"

	"github.com/and1truong/liveboard/internal/web"
)

func (d Deps) getAppSettings(w http.ResponseWriter, _ *http.Request) {
	s := web.LoadSettingsFromDir(d.Dir)
	_ = json.NewEncoder(w).Encode(s)
}

func (d Deps) putAppSettings(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > 1<<20 {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	current := web.LoadSettingsFromDir(d.Dir)
	if err := json.NewDecoder(r.Body).Decode(&current); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	web.SanitizeSettings(&current)
	if err := web.SaveSettingsToDir(d.Dir, current); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

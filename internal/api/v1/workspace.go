package v1

import (
	"encoding/json"
	"net/http"

	"github.com/and1truong/liveboard/internal/web"
)

type workspaceResponse struct {
	Name       string `json:"name"`
	Dir        string `json:"dir"`
	BoardCount int    `json:"board_count"`
}

func (d Deps) getWorkspace(w http.ResponseWriter, _ *http.Request) {
	boards, err := d.Workspace.ListBoards()
	if err != nil {
		writeError(w, err)
		return
	}
	settings := web.LoadSettingsFromDir(d.Workspace.Dir)
	_ = json.NewEncoder(w).Encode(workspaceResponse{
		Name:       settings.SiteName,
		Dir:        d.Workspace.Dir,
		BoardCount: len(boards),
	})
}

package v1

import (
	"encoding/json"
	"net/http"
)

type workspaceResponse struct {
	Dir        string `json:"dir"`
	BoardCount int    `json:"board_count"`
}

func (d Deps) getWorkspace(w http.ResponseWriter, _ *http.Request) {
	boards, err := d.Workspace.ListBoards()
	if err != nil {
		writeError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(workspaceResponse{
		Dir:        d.Workspace.Dir,
		BoardCount: len(boards),
	})
}

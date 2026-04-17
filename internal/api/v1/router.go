// Package v1 implements the /api/v1 REST API for LiveBoard.
package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/search"
	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/internal/workspace"
)

// Deps is the set of dependencies the v1 handlers need.
type Deps struct {
	Dir       string // workspace directory for settings.json
	Workspace *workspace.Workspace
	Engine    *board.Engine
	SSE       *web.SSEBroker
	Search    *search.Index
}

// Router returns a chi subrouter with all /api/v1 routes registered.
//
// Board ids may contain a single "/" separating a folder from the file stem
// (e.g. "work/ideas"). chi cannot match "/" inside a single {param}, so all
// per-board endpoints place the id as a trailing catch-all ("*") after a
// fixed discriminator segment ("board", "mutate", "pin", "settings").
func Router(d Deps) chi.Router {
	r := chi.NewRouter()
	r.Use(jsonContentType)
	r.Get("/settings", d.getAppSettings)
	r.Put("/settings", d.putAppSettings)
	r.Get("/workspace", d.getWorkspace)
	r.Get("/events", d.getEvents)
	r.Get("/search", d.getSearch)
	r.Get("/cards/{cardId}/backlinks", d.getBacklinks)
	r.Route("/boards", func(r chi.Router) {
		r.Get("/", d.listBoards)
		r.Post("/", d.createBoard)
		r.Get("/list-lite", d.listBoardsLite)

		// Folder CRUD (fixed prefix — no ambiguity with wildcards below).
		r.Get("/folders", d.listFolders)
		r.Post("/folders", d.createFolder)
		r.Patch("/folders/*", d.renameFolder)
		r.Delete("/folders/*", d.deleteFolder)

		// Per-board endpoints — id at the tail as catch-all "*".
		r.Get("/board/*", d.getBoard)
		r.Patch("/board/*", d.renameBoard)
		r.Delete("/board/*", d.deleteBoard)
		r.Post("/mutate/*", d.postMutation)
		r.Post("/pin/*", d.toggleBoardPin)
		r.Get("/settings/*", d.getBoardSettings)
		r.Put("/settings/*", d.putBoardSettings)
	})
	return r
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

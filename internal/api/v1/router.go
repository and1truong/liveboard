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
		r.Get("/list-lite", d.listBoardsLite)
		r.Post("/", d.createBoard)
		r.Get("/{slug}", d.getBoard)
		r.Patch("/{slug}", d.renameBoard)
		r.Delete("/{slug}", d.deleteBoard)
		r.Post("/{slug}/mutations", d.postMutation)
		r.Post("/{slug}/pin", d.toggleBoardPin)
		r.Get("/{slug}/settings", d.getBoardSettings)
		r.Put("/{slug}/settings", d.putBoardSettings)
	})
	return r
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

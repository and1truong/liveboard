package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jfyne/live"

	"github.com/and1truong/liveboard/internal/board"
	gitpkg "github.com/and1truong/liveboard/internal/git"
	"github.com/and1truong/liveboard/internal/search"
	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/internal/workspace"
)

// Server is the REST API server for LiveBoard.
type Server struct {
	ws          *workspace.Workspace
	eng         *board.Engine
	git         *gitpkg.Repository
	search      *search.Index
	liveHandler *web.Handler
	router      chi.Router
}

// NewServer creates a Server with all routes registered.
func NewServer(ws *workspace.Workspace, eng *board.Engine, git *gitpkg.Repository) *Server {
	s := &Server{
		ws:          ws,
		eng:         eng,
		git:         git,
		liveHandler: web.NewHandler(ws, eng, git),
	}

	// Initialize search index and populate from existing boards.
	if idx, err := search.NewIndex(""); err == nil {
		s.search = idx
		s.liveHandler.SetSearch(idx)
		s.rebuildSearchIndex()
	}

	s.router = s.buildRouter()
	return s
}

// rebuildSearchIndex indexes all boards in the workspace.
func (s *Server) rebuildSearchIndex() {
	if s.search == nil {
		return
	}
	boards, err := s.ws.ListBoards()
	if err != nil {
		return
	}
	for _, b := range boards {
		slug := boardSlugFromPath(b.FilePath)
		_ = s.search.IndexBoard(slug, &b)
	}
}

// reindexBoard re-indexes a single board after mutation.
func (s *Server) reindexBoard(slug string) {
	if s.search == nil {
		return
	}
	_ = s.search.RemoveBoard(slug)
	board, err := s.ws.LoadBoard(slug)
	if err != nil {
		return
	}
	_ = s.search.IndexBoard(slug, board)
}

// boardSlugFromPath extracts the slug from a board file path.
func boardSlugFromPath(path string) string {
	base := path
	// Find last path separator
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			base = path[i+1:]
			break
		}
	}
	// Remove .md extension
	if len(base) > 3 && base[len(base)-3:] == ".md" {
		return base[:len(base)-3]
	}
	return base
}

// Router returns the http.Handler for use with httptest.
func (s *Server) Router() http.Handler {
	return s.router
}

// Start begins listening on the given address.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serve static assets
	r.Get("/static/*", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.StripPrefix("/static/", http.FileServer(http.Dir("web"))).ServeHTTP(w, req)
	})

	// LiveView JavaScript (required)
	r.Handle("/live.js", live.Javascript{})

	// Web UI routes
	r.Handle("/", s.liveHandler.BoardListHandler())
	r.Handle("/board/{name}", s.liveHandler.BoardViewHandler())
	r.Handle("/search-ui", s.liveHandler.SearchHandler())
	r.Handle("/settings", s.liveHandler.SettingsHandler())
	r.Handle("/api/settings", s.liveHandler.SettingsAPIHandler())

	// REST API routes (with JSON content type)
	r.Route("/boards", func(r chi.Router) {
		r.Use(jsonContentType)
		r.Get("/", s.listBoards)
		r.Post("/", s.createBoard)
		r.Route("/{board}", func(r chi.Router) {
			r.Get("/", s.getBoard)
			r.Delete("/", s.deleteBoard)
			r.Post("/columns", s.addColumn)
			r.Route("/columns/{column}", func(r chi.Router) {
				r.Delete("/", s.deleteColumn)
				r.Post("/move", s.moveColumn)
				r.Patch("/", s.stubHandler)
				r.Post("/cards", s.addCard)
			})
		})
	})

	// Card operations: /boards/{board}/columns/{column}/cards is for adding,
	// individual card ops use index-based paths:
	r.Route("/boards/{board}/cols/{colIdx}/cards/{cardIdx}", func(r chi.Router) {
		r.Use(jsonContentType)
		r.Get("/", s.getCard)
		r.Delete("/", s.deleteCard)
		r.Post("/move", s.moveCard)
		r.Post("/complete", s.completeCard)
		r.Post("/tag", s.tagCard)
	})

	r.Get("/search", s.searchHandler)
	r.Get("/events", s.stubHandler)
	r.Get("/events/ws", s.stubHandler)

	return r
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) stubHandler(w http.ResponseWriter, _ *http.Request) {
	respondError(w, http.StatusNotImplemented, "not yet implemented")
}

func (s *Server) gitCommit(relPath, message string) {
	if s.git != nil {
		_ = s.git.Commit(relPath, message)
	}
}

func (s *Server) gitCommitRemove(relPath, message string) {
	if s.git != nil {
		_ = s.git.CommitRemove(relPath, message)
	}
}

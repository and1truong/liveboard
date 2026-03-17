package api

import (
	"net/http"

	"github.com/and1truong/liveboard/internal/board"
	gitpkg "github.com/and1truong/liveboard/internal/git"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the REST API server for LiveBoard.
type Server struct {
	ws     *workspace.Workspace
	eng    *board.Engine
	git    *gitpkg.Repository
	router chi.Router
}

// NewServer creates a Server with all routes registered.
func NewServer(ws *workspace.Workspace, eng *board.Engine, git *gitpkg.Repository) *Server {
	s := &Server{ws: ws, eng: eng, git: git}
	s.router = s.buildRouter()
	return s
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
	r.Use(jsonContentType)

	r.Route("/boards", func(r chi.Router) {
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

	r.Route("/cards/{id}", func(r chi.Router) {
		r.Get("/", s.getCard)
		r.Delete("/", s.deleteCard)
		r.Post("/move", s.moveCard)
		r.Post("/complete", s.completeCard)
		r.Post("/tag", s.tagCard)
		r.Patch("/", s.patchCard)
	})

	r.Get("/search", s.stubHandler)
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

func (s *Server) stubHandler(w http.ResponseWriter, r *http.Request) {
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

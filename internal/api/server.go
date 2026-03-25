package api

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/and1truong/liveboard/internal/board"
	livemcp "github.com/and1truong/liveboard/internal/mcp"
	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/internal/workspace"
	staticweb "github.com/and1truong/liveboard/web"
)

// Server is the REST API server for LiveBoard.
type Server struct {
	ws         *workspace.Workspace
	eng        *board.Engine
	webHandler *web.Handler
	mcpServer  *livemcp.Server
	router     chi.Router
	httpServer *http.Server
	noCache    bool
	readOnly   bool
}

// NewServer creates a Server with all routes registered.
func NewServer(ws *workspace.Workspace, eng *board.Engine, noCache, readOnly bool, version string) *Server {
	s := &Server{
		ws:         ws,
		eng:        eng,
		webHandler: web.NewHandler(ws, eng, version, readOnly),
		mcpServer:  livemcp.New(ws, eng, version),
		noCache:    noCache,
		readOnly:   readOnly,
	}
	s.router = s.buildRouter()
	return s
}

// Router returns the http.Handler for use with httptest.
func (s *Server) Router() http.Handler {
	return s.router
}

// Start begins listening on the given address (blocking).
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

// ListenAndServe starts the server in a goroutine and returns the bound address.
// Use Shutdown to stop the server. Useful when binding to port 0.
func (s *Server) ListenAndServe(addr string) (net.Addr, error) {
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}
	s.httpServer = &http.Server{Handler: s.router}
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()
	return ln.Addr(), nil
}

// Shutdown gracefully stops the server started via ListenAndServe.
// It first closes all SSE connections so long-lived streams don't block the
// HTTP server's graceful drain.
func (s *Server) Shutdown(ctx context.Context) error {
	s.webHandler.SSE.Shutdown()
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if s.readOnly {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions {
					http.Error(w, "read-only mode", http.StatusMethodNotAllowed)
					return
				}
				next.ServeHTTP(w, req)
			})
		})
	}

	// Serve static assets
	r.Get("/static/*", func(w http.ResponseWriter, req *http.Request) {
		if s.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		http.StripPrefix("/static/", http.FileServer(http.FS(staticweb.FS))).ServeHTTP(w, req)
	})

	// Web UI routes (HTMX)
	r.Get("/", s.webHandler.BoardListPage)
	r.Post("/boards/new", s.webHandler.HandleCreateBoard)
	r.Post("/boards/{slug}/delete", s.webHandler.HandleDeleteBoard)
	r.Post("/boards/{slug}/icon", s.webHandler.HandleSetBoardIconList)

	r.Get("/board/{slug}", s.webHandler.BoardViewPage)
	r.Get("/board/{slug}/content", s.webHandler.BoardContent)
	r.Get("/board/{slug}/events", s.webHandler.SSE.ServeHTTP)
	r.Post("/board/{slug}/cards", s.webHandler.HandleCreateCard)
	r.Post("/board/{slug}/cards/move", s.webHandler.HandleMoveCard)
	r.Post("/board/{slug}/cards/reorder", s.webHandler.HandleReorderCard)
	r.Post("/board/{slug}/cards/delete", s.webHandler.HandleDeleteCard)
	r.Post("/board/{slug}/cards/complete", s.webHandler.HandleToggleComplete)
	r.Post("/board/{slug}/cards/edit", s.webHandler.HandleEditCard)
	r.Post("/board/{slug}/columns", s.webHandler.HandleCreateColumn)
	r.Post("/board/{slug}/columns/rename", s.webHandler.HandleRenameColumn)
	r.Post("/board/{slug}/columns/delete", s.webHandler.HandleDeleteColumn)
	r.Post("/board/{slug}/columns/sort", s.webHandler.HandleSortColumn)
	r.Post("/board/{slug}/columns/move", s.webHandler.HandleMoveColumn)
	r.Post("/board/{slug}/meta", s.webHandler.HandleUpdateBoardMeta)
	r.Post("/board/{slug}/settings", s.webHandler.HandleUpdateBoardSettings)
	r.Post("/board/{slug}/icon", s.webHandler.HandleSetBoardIcon)

	r.Post("/api/boards/pin", s.webHandler.HandleTogglePin)
	r.Get("/api/boards/sidebar", s.webHandler.HandleSidebarBoards)

	r.Handle("/settings", s.webHandler.SettingsHandler())
	r.Handle("/api/settings", s.webHandler.SettingsAPIHandler())
	r.Get("/api/export", s.webHandler.ExportHandler().ServeHTTP)

	s.mountAPIRoutes(r)

	return r
}

func (s *Server) mountAPIRoutes(r chi.Router) {
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

	// MCP server (Streamable HTTP transport)
	r.Mount("/mcp", s.mcpServer.StreamableHTTPHandler())

	r.Get("/search", s.stubHandler)
	r.Get("/events", s.stubHandler)
	r.Get("/events/ws", s.stubHandler)
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

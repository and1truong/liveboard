package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"io/fs"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	apiv1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/board"
	livemcp "github.com/and1truong/liveboard/internal/mcp"
	"github.com/and1truong/liveboard/internal/reminder"
	"github.com/and1truong/liveboard/internal/search"
	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/internal/workspace"
	"github.com/and1truong/liveboard/pkg/models"
	staticweb "github.com/and1truong/liveboard/web"
	renderer "github.com/and1truong/liveboard/web/renderer/default"
	shell "github.com/and1truong/liveboard/web/shell"
)

// Server is the REST API server for LiveBoard.
type Server struct {
	ws                *workspace.Workspace
	eng               *board.Engine
	webHandler        *web.Handler
	mcpServer         *livemcp.Server
	reminderScheduler *reminder.Scheduler
	router            chi.Router
	httpServer        *http.Server
	noCache           bool
	readOnly          bool
	basicAuthUser     string
	basicAuthPass     string
}

// NewServer creates a Server with all routes registered.
func NewServer(ws *workspace.Workspace, eng *board.Engine, noCache, readOnly, isDesktop bool, version string, basicAuthUser, basicAuthPass string) *Server {
	h := web.NewHandler(ws, eng, version, readOnly, isDesktop)
	s := &Server{
		ws:            ws,
		eng:           eng,
		webHandler:    h,
		mcpServer:     livemcp.New(ws, eng, version),
		noCache:       noCache,
		readOnly:      readOnly,
		basicAuthUser: basicAuthUser,
		basicAuthPass: basicAuthPass,
	}

	// Initialize reminder scheduler if enabled
	settings := web.LoadSettingsFromDir(ws.Dir)
	if settings.ReminderEnabled {
		s.startReminderScheduler(ws, h.ReminderStore(), isDesktop, settings)
	}

	s.router = s.buildRouter()
	return s
}

func makeReminderNotifyFn(sse *web.SSEBroker, isDesktop bool) reminder.NotifyFunc {
	return func(r reminder.Reminder, cardTitle string, stats *reminder.BoardStats) {
		payload := map[string]any{
			"id":         r.ID,
			"type":       r.Type,
			"board_slug": r.BoardSlug,
			"card_id":    r.CardID,
			"card_title": cardTitle,
		}
		if stats != nil {
			payload["message"] = fmt.Sprintf("%d open, %d overdue, %d due this week", stats.TotalOpen, stats.Overdue, stats.DueThisWeek)
		}
		data, _ := json.Marshal(payload)
		sse.PublishGlobal(web.SSEEvent{Type: "reminder-fire", Payload: string(data)})

		if isDesktop {
			sendDesktopReminderNotification(r, cardTitle, stats)
		}
	}
}

func sendDesktopReminderNotification(r reminder.Reminder, cardTitle string, stats *reminder.BoardStats) {
	title := "Reminder"
	body := cardTitle
	if r.Type == reminder.ReminderTypeBoard {
		title = "Board Reminder"
		body = r.BoardSlug
		if stats != nil {
			body = fmt.Sprintf("%s: %d open, %d overdue", r.BoardSlug, stats.TotalOpen, stats.Overdue)
		}
	}
	_ = reminder.SendSystemNotification(title, body, "")
}

func computeBoardStats(b *models.Board) reminder.BoardStats {
	var bs reminder.BoardStats
	now := time.Now()
	today := now.Format("2006-01-02")
	weekEnd := now.AddDate(0, 0, 7-int(now.Weekday()))
	weekEndStr := weekEnd.Format("2006-01-02")

	for _, col := range b.Columns {
		for _, card := range col.Cards {
			if card.Completed {
				continue
			}
			bs.TotalOpen++
			if card.Due != "" && card.Due < today {
				bs.Overdue++
			}
			if card.Due != "" && card.Due >= today && card.Due <= weekEndStr {
				bs.DueThisWeek++
			}
		}
	}
	return bs
}

func makeBoardStatsFn(ws *workspace.Workspace) reminder.BoardStatsFunc {
	return func(slug string) reminder.BoardStats {
		b, err := ws.LoadBoard(slug)
		if err != nil {
			return reminder.BoardStats{}
		}
		return computeBoardStats(b)
	}
}

func (s *Server) startReminderScheduler(ws *workspace.Workspace, store *reminder.Store, isDesktop bool, _ web.AppSettings) {
	s.reminderScheduler = reminder.NewScheduler(
		store, time.Minute,
		makeReminderNotifyFn(s.webHandler.SSE, isDesktop),
		makeBoardStatsFn(ws),
	)
	s.reminderScheduler.Start()
	log.Println("reminder scheduler started")
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
	if s.reminderScheduler != nil {
		s.reminderScheduler.Stop()
	}
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

	if s.basicAuthUser != "" && s.basicAuthPass != "" {
		r.Use(middleware.BasicAuth("LiveBoard", map[string]string{
			s.basicAuthUser: s.basicAuthPass,
		}))
	}

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

	// Serve static assets (handler allocated once, not per-request)
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticweb.FS)))
	r.Get("/static/*", func(w http.ResponseWriter, req *http.Request) {
		if s.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		staticHandler.ServeHTTP(w, req)
	})

	if os.Getenv("LIVEBOARD_APP_SHELL") == "1" {
		s.mountShellRoutes(r)
		log.Println("shell mounted at /app/")
	}

	s.mountWebRoutes(r)
	s.mountAPIRoutes(r)

	if os.Getenv("LIVEBOARD_PPROF") != "" {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		r.HandleFunc("/debug/pprof/{name}", pprof.Index)
		log.Println("pprof profiling enabled at /debug/pprof/")
	}

	return r
}

func (s *Server) mountWebRoutes(r chi.Router) {
	h := s.webHandler

	// Board list routes
	r.Get("/", h.BoardList.BoardListPage)
	r.Post("/boards/new", h.BoardList.HandleCreateBoard)
	r.Post("/boards/{slug}/delete", h.BoardList.HandleDeleteBoard)
	r.Post("/boards/{slug}/icon", h.BoardList.HandleSetBoardIconList)

	// Board view + mutation routes
	r.Get("/board/{slug}", h.BoardView.BoardViewPage)
	r.Get("/board/{slug}/content", h.BoardView.BoardContent)
	r.Get("/board/{slug}/events", h.SSE.ServeHTTP)
	r.Post("/board/{slug}/cards", h.BoardView.HandleCreateCard)
	r.Post("/board/{slug}/cards/move", h.BoardView.HandleMoveCard)
	r.Post("/board/{slug}/cards/move-to-board", h.BoardView.HandleMoveCardToBoard)
	r.Post("/board/{slug}/cards/reorder", h.BoardView.HandleReorderCard)
	r.Post("/board/{slug}/cards/delete", h.BoardView.HandleDeleteCard)
	r.Post("/board/{slug}/cards/complete", h.BoardView.HandleToggleComplete)
	r.Post("/board/{slug}/cards/edit", h.BoardView.HandleEditCard)
	r.Post("/board/{slug}/columns", h.BoardView.HandleCreateColumn)
	r.Post("/board/{slug}/columns/rename", h.BoardView.HandleRenameColumn)
	r.Post("/board/{slug}/columns/delete", h.BoardView.HandleDeleteColumn)
	r.Post("/board/{slug}/columns/collapse", h.BoardView.HandleToggleColumnCollapse)
	r.Post("/board/{slug}/columns/sort", h.BoardView.HandleSortColumn)
	r.Post("/board/{slug}/columns/move", h.BoardView.HandleMoveColumn)
	r.Post("/board/{slug}/meta", h.BoardView.HandleUpdateBoardMeta)
	r.Post("/board/{slug}/settings", h.BoardView.HandleUpdateBoardSettings)
	r.Post("/board/{slug}/icon", h.BoardView.HandleSetBoardIcon)

	// Board API routes
	r.Post("/api/boards/pin", h.BoardList.HandleTogglePin)
	r.Get("/api/boards/sidebar", h.BoardList.HandleSidebarBoards)
	r.Get("/api/boards/list-lite", h.BoardList.HandleBoardsListLite)

	// Settings routes
	r.Handle("/settings", h.Settings.SettingsHandler())
	r.Handle("/api/settings", h.Settings.SettingsAPIHandler())
	r.Get("/api/export", h.ExportHandler().ServeHTTP)

	// Global SSE events (reminders, notifications)
	r.Get("/events/global", h.SSE.ServeGlobalSSE)

	// Reminder routes
	r.Get("/reminders", h.Reminders.RemindersPage)
	r.Post("/reminders/set", h.Reminders.HandleSetReminder)
	r.Post("/reminders/dismiss/{id}", h.Reminders.HandleDismissReminder)
	r.Post("/reminders/snooze/{id}", h.Reminders.HandleSnoozeReminder)
	r.Delete("/reminders/{id}", h.Reminders.HandleDeleteReminder)
	r.Post("/reminders/clear-fired", h.Reminders.HandleClearFired)
	r.Post("/reminders/clear-history", h.Reminders.HandleClearHistory)
	r.Post("/reminders/settings", h.Reminders.HandleUpdateReminderSettings)
}

func (s *Server) mountAPIRoutes(r chi.Router) {
	idx, err := search.New()
	if err != nil {
		log.Printf("search: failed to init index: %v", err)
	}
	if idx != nil {
		if boards, err := s.ws.ListBoards(); err == nil {
			for i := range boards {
				b := boards[i]
				slug := strings.TrimSuffix(filepath.Base(b.FilePath), ".md")
				if slug == "" {
					continue
				}
				_ = idx.UpdateBoard(slug, &b)
			}
		}
	}
	r.Mount("/api/v1", apiv1.Router(apiv1.Deps{
		Workspace: s.ws,
		Engine:    s.eng,
		SSE:       s.webHandler.SSE,
		Search:    idx,
	}))
	r.Method(http.MethodGet, "/api/versions", apiv1.VersionsHandler())

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

	// Board-level card operations (non-indexed):
	r.Route("/boards/{board}/cards", func(r chi.Router) {
		r.Use(jsonContentType)
		r.Post("/move-to-board", s.moveCardToBoard)
	})

	// MCP server (Streamable HTTP transport)
	r.Mount("/mcp", s.mcpServer.StreamableHTTPHandler())

	r.Get("/search", s.stubHandler)
	r.Get("/events", s.stubHandler)
	r.Get("/events/ws", s.stubHandler)
}

const liveboardConfigMarker = `/*__LIVEBOARD_CONFIG__*/ { adapter: 'local' }`
const liveboardConfigServer = `{ adapter: 'server', baseUrl: '/api/v1' }`

func injectLiveboardConfig(html []byte) []byte {
	return bytes.Replace(html, []byte(liveboardConfigMarker), []byte(liveboardConfigServer), 1)
}

func (s *Server) mountShellRoutes(r chi.Router) {
	shellSub, err := fs.Sub(shell.FS, "dist")
	if err != nil {
		log.Printf("shell embed: %v", err)
		return
	}
	rendererSub, err := fs.Sub(renderer.FS, "dist")
	if err != nil {
		log.Printf("renderer embed: %v", err)
		return
	}

	// Pre-load and patch the shell index once at startup.
	indexBytes, err := fs.ReadFile(shellSub, "index.html")
	if err != nil {
		log.Printf("shell index: %v", err)
		return
	}
	indexPatched := injectLiveboardConfig(indexBytes)

	shellHandler := http.StripPrefix("/app/", http.FileServer(http.FS(shellSub)))
	rendererHandler := http.StripPrefix("/app/renderer/default/", http.FileServer(http.FS(rendererSub)))

	serveIndex := func(w http.ResponseWriter, _ *http.Request) {
		if s.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexPatched)
	}

	r.Get("/app", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/app/", http.StatusMovedPermanently)
	})
	r.Get("/app/*", func(w http.ResponseWriter, req *http.Request) {
		if s.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store")
		}
		path := req.URL.Path
		if strings.HasPrefix(path, "/app/renderer/default/") {
			rendererHandler.ServeHTTP(w, req)
			return
		}
		if path == "/app/" || path == "/app/index.html" || strings.HasPrefix(path, "/app/b/") {
			serveIndex(w, req)
			return
		}
		shellHandler.ServeHTTP(w, req)
	})
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

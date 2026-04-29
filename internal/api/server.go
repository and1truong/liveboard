// Package api wires the chi router, shell/renderer mount, and /api/export
// handler. The canonical JSON API lives under internal/api/v1.
package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"
	"os"
	"strconv"

	"io/fs"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	apiv1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/attachments"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/export"
	livemcp "github.com/and1truong/liveboard/internal/mcp"
	"github.com/and1truong/liveboard/internal/search"
	"github.com/and1truong/liveboard/internal/web"
	"github.com/and1truong/liveboard/internal/workspace"
	renderer "github.com/and1truong/liveboard/web/renderer/default"
	shell "github.com/and1truong/liveboard/web/shell"
)

// Server is the REST API server for LiveBoard.
type Server struct {
	ws            *workspace.Workspace
	eng           *board.Engine
	sse           *web.SSEBroker
	mcpServer     *livemcp.Server
	router        chi.Router
	httpServer    *http.Server
	noCache       bool
	readOnly      bool
	basicAuthUser string
	basicAuthPass string
}

// NewServer creates a Server with all routes registered.
//
// The isDesktop parameter is retained for call-site compatibility with
// cmd/liveboard-desktop and mobile/gobridge; it has no effect now that the
// shell is always mounted and the reminder scheduler has been removed.
func NewServer(ws *workspace.Workspace, eng *board.Engine, noCache, readOnly, _ bool, version string, basicAuthUser, basicAuthPass string) *Server {
	s := &Server{
		ws:            ws,
		eng:           eng,
		sse:           web.NewSSEBroker(),
		mcpServer:     livemcp.New(ws, eng, version),
		noCache:       noCache,
		readOnly:      readOnly,
		basicAuthUser: basicAuthUser,
		basicAuthPass: basicAuthPass,
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
	s.sse.Shutdown()
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

	s.mountShellRoutes(r)

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/app/", http.StatusFound)
	})

	r.Get("/api/export", s.exportHandler)

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
	appSettings := web.LoadSettingsFromDir(s.ws.Dir)
	r.Mount("/api/v1", apiv1.Router(apiv1.Deps{
		Dir:                s.ws.Dir,
		Workspace:          s.ws,
		Engine:             s.eng,
		SSE:                s.sse,
		Search:             idx,
		Attachments:        attachments.NewStore(s.ws.Dir),
		AttachmentMaxBytes: appSettings.AttachmentMaxBytes,
	}))
	r.Method(http.MethodGet, "/api/versions", apiv1.VersionsHandler())

	// MCP server (Streamable HTTP transport)
	r.Mount("/mcp", s.mcpServer.StreamableHTTPHandler())
}

// exportHandler streams a workspace ZIP. ?format=md returns raw markdown;
// ?format=html (default) renders a static HTML site using settings.json for
// theme, color-theme, and site-name. Pass ?attachments=false to exclude
// referenced attachment blobs from the archive.
func (s *Server) exportHandler(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	includeAttachments := r.URL.Query().Get("attachments") != "false"
	w.Header().Set("Content-Type", "application/zip")

	switch format {
	case "md", "markdown":
		w.Header().Set("Content-Disposition", `attachment; filename="liveboard-export-md.zip"`)
		if err := export.WriteMarkdownZipToOpts(w, s.ws, export.Options{IncludeAttachments: includeAttachments}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case "", "html":
		settings := web.LoadSettingsFromDir(s.ws.Dir)
		opts := export.Options{
			Theme:              settings.Theme,
			ColorTheme:         settings.ColorTheme,
			SiteName:           settings.SiteName,
			IncludeAttachments: includeAttachments,
		}
		w.Header().Set("Content-Disposition", `attachment; filename="liveboard-export.zip"`)
		if err := export.WriteZipTo(w, s.ws, opts); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		http.Error(w, "unknown format: "+format, http.StatusBadRequest)
	}
}

const liveboardConfigMarker = `/*__LIVEBOARD_CONFIG__*/ { adapter: 'local' }`
const liveboardConfigServer = `{ adapter: 'server', baseUrl: '/api/v1' }`

func injectLiveboardConfig(html []byte) []byte {
	return bytes.Replace(html, []byte(liveboardConfigMarker), []byte(liveboardConfigServer), 1)
}

func (s *Server) mountShellRoutes(r chi.Router) {
	// Dev mode: proxy to Vite dev servers instead of serving embedded bundles.
	// Enables HMR for TS/CSS without rebuild. See Makefile adapter-test.
	if shellURL, rendererURL := os.Getenv("LIVEBOARD_SHELL_DEV_URL"), os.Getenv("LIVEBOARD_RENDERER_DEV_URL"); shellURL != "" && rendererURL != "" {
		s.mountShellDevProxy(r, shellURL, rendererURL)
		return
	}

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

// newViteProxy builds a reverse proxy to a Vite dev server. HTML responses are
// patched to swap the liveboard config marker so the shell boots into the
// server adapter. Accept-Encoding is stripped so the upstream returns plain
// bytes we can rewrite. WebSocket upgrades (Vite HMR) are forwarded natively
// by httputil.ReverseProxy.
func newViteProxy(target string) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	orig := proxy.Director
	proxy.Director = func(req *http.Request) {
		orig(req)
		req.Header.Del("Accept-Encoding")
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		patched := injectLiveboardConfig(body)
		resp.Body = io.NopCloser(bytes.NewReader(patched))
		resp.ContentLength = int64(len(patched))
		resp.Header.Set("Content-Length", strconv.Itoa(len(patched)))
		return nil
	}
	return proxy, nil
}

func (s *Server) mountShellDevProxy(r chi.Router, shellURL, rendererURL string) {
	shellProxy, err := newViteProxy(shellURL)
	if err != nil {
		log.Printf("shell dev proxy: %v", err)
		return
	}
	rendererProxy, err := newViteProxy(rendererURL)
	if err != nil {
		log.Printf("renderer dev proxy: %v", err)
		return
	}

	r.Get("/app", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/app/", http.StatusMovedPermanently)
	})
	r.HandleFunc("/app/*", func(w http.ResponseWriter, req *http.Request) {
		if s.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store")
		}
		if strings.HasPrefix(req.URL.Path, "/app/renderer/default/") {
			rendererProxy.ServeHTTP(w, req)
			return
		}
		shellProxy.ServeHTTP(w, req)
	})
	log.Printf("shell proxying to Vite: shell=%s renderer=%s", shellURL, rendererURL)
}

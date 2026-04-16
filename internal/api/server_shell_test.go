package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/and1truong/liveboard/internal/workspace"
)

// fakeViteServer stands in for a Vite dev server. It returns a canned HTML
// body containing the liveboard config marker when asked for the shell index,
// and records the request path for assertion.
func fakeViteServer(t *testing.T, body string, contentType string) (*httptest.Server, *string) {
	t.Helper()
	var lastPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &lastPath
}

func setupShellTest(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.Open(dir)
	return NewServer(ws, ws.Engine, false, false, false, "test", "", "")
}

func TestShellRoute_Disabled(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "")
	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("shell route should be 404 when flag disabled; got %d", rec.Code)
	}
}

func TestShellRoute_Enabled(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "LiveBoard Shell") {
		t.Fatalf("response did not contain expected shell HTML")
	}
}

func TestShellIndex_InjectsServerConfig(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "adapter: 'server'") {
		t.Errorf("expected injected server config in body; got: %s", body)
	}
	if strings.Contains(body, "/*__LIVEBOARD_CONFIG__*/") {
		t.Errorf("placeholder marker should be replaced but is still present")
	}
}

func TestShellIndex_ExplicitIndexPath(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/index.html", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "adapter: 'server'") {
		t.Errorf("expected injection on /app/index.html; got: %s", rec.Body.String())
	}
}

func TestShellDev_ProxiesShellAndRewritesMarker(t *testing.T) {
	indexBody := `<!doctype html><html><body><script>window.__LIVEBOARD_CONFIG__ = /*__LIVEBOARD_CONFIG__*/ { adapter: 'local' };</script></body></html>`
	shellSrv, shellPath := fakeViteServer(t, indexBody, "text/html; charset=utf-8")
	rendererSrv, _ := fakeViteServer(t, `<div id="root"></div>`, "text/html; charset=utf-8")

	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	t.Setenv("LIVEBOARD_SHELL_DEV_URL", shellSrv.URL)
	t.Setenv("LIVEBOARD_RENDERER_DEV_URL", rendererSrv.URL)

	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if *shellPath != "/app/" {
		t.Errorf("shell upstream should see /app/, got %q", *shellPath)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "adapter: 'server'") {
		t.Errorf("expected marker to be rewritten to server adapter; got: %s", body)
	}
	if strings.Contains(body, "/*__LIVEBOARD_CONFIG__*/") {
		t.Errorf("placeholder marker should be replaced but is still present")
	}
}

func TestShellDev_ProxiesRendererSubpath(t *testing.T) {
	shellSrv, _ := fakeViteServer(t, `SHELL`, "text/plain")
	rendererSrv, rendererPath := fakeViteServer(t, `RENDERER_HTML`, "text/plain")

	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	t.Setenv("LIVEBOARD_SHELL_DEV_URL", shellSrv.URL)
	t.Setenv("LIVEBOARD_RENDERER_DEV_URL", rendererSrv.URL)

	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/renderer/default/src/main.tsx", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if *rendererPath != "/app/renderer/default/src/main.tsx" {
		t.Errorf("renderer upstream should see full path, got %q", *rendererPath)
	}
	if !strings.Contains(rec.Body.String(), "RENDERER_HTML") {
		t.Errorf("expected renderer body, got: %s", rec.Body.String())
	}
}

func TestShellDev_NonHTMLResponseNotRewritten(t *testing.T) {
	// TS source served by Vite: contains the marker literal somewhere but must
	// not be rewritten — only HTML should be patched.
	jsBody := `export const cfg = /*__LIVEBOARD_CONFIG__*/ { adapter: 'local' };`
	shellSrv, _ := fakeViteServer(t, jsBody, "application/javascript")
	rendererSrv, _ := fakeViteServer(t, `x`, "text/plain")

	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	t.Setenv("LIVEBOARD_SHELL_DEV_URL", shellSrv.URL)
	t.Setenv("LIVEBOARD_RENDERER_DEV_URL", rendererSrv.URL)

	s := setupShellTest(t)
	req := httptest.NewRequest(http.MethodGet, "/app/src/main.ts", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "/*__LIVEBOARD_CONFIG__*/") {
		t.Errorf("JS response should pass through untouched; got: %s", rec.Body.String())
	}
}

func TestShellRoute_Renderer(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	s := setupShellTest(t)

	req := httptest.NewRequest(http.MethodGet, "/app/renderer/default/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<div id=\"root\">") {
		t.Fatalf("response did not contain renderer root div")
	}
}

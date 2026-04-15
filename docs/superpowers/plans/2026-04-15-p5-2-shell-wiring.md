# P5.2 — Shell Wiring (LocalAdapter ↔ ServerAdapter) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the shell pick `ServerAdapter` (P5.1) when served by the Go binary with `LIVEBOARD_APP_SHELL=1`, and `LocalAdapter` everywhere else (online build, dev shell). Driven by a runtime config object the Go server injects via string-replacement into `index.html`.

**Architecture:** The shell's `index.html` carries a placeholder `<script>` that defines `window.__LIVEBOARD_CONFIG__ = /*__LIVEBOARD_CONFIG__*/ { adapter: 'local' };`. The Go shell handler intercepts `/app/` and `/app/index.html`, substitutes `{ adapter: 'server', baseUrl: '/api/v1' }` into the placeholder, and serves the modified HTML. Shell `main.ts` reads the config at boot and instantiates the right adapter.

**Tech Stack:** Go 1.24 (shell handler edit). TypeScript shell main (config read + branch). No new deps.

**Spec:** `docs/superpowers/specs/2026-04-15-p5-2-shell-wiring-design.md`

**Conventions:**
- Shell code in `web/shell/`. Go shell handler in `internal/api/server.go`.
- Tests colocated.
- Commit prefixes: `feat(shell)`, `feat(api)`, `test(api)`.
- `make lint` must pass after Go changes.
- Use bun, never npx (irrelevant here but session convention).

---

## File structure

**Modified:**
- `web/shell/index.html` — add the placeholder `<script>` tag.
- `web/shell/src/main.ts` — read `window.__LIVEBOARD_CONFIG__`, branch adapter selection.
- `internal/api/server.go` — intercept `/app/` (shell index) to do the placeholder replacement.
- `internal/api/server_shell_test.go` — assert the replacement happens with the flag and not without.

**No new files.**

---

## Task 1: Placeholder script in shell `index.html`

**Files:**
- Modify: `web/shell/index.html`

The placeholder uses a JS comment so the file remains valid JavaScript when served unmodified.

- [ ] **Step 1: Edit**

Open `web/shell/index.html`. Add the placeholder script BEFORE the existing module script. Final body:
```html
  <iframe id="renderer" title="LiveBoard renderer"></iframe>
  <script>window.__LIVEBOARD_CONFIG__ = /*__LIVEBOARD_CONFIG__*/ { adapter: 'local' };</script>
  <script type="module" src="./src/main.ts"></script>
```

The marker `/*__LIVEBOARD_CONFIG__*/ { adapter: 'local' }` is a single-line literal that the Go interceptor matches.

- [ ] **Step 2: Rebuild shell**

```bash
cd /Users/htruong/code/htruong/liveboard && make shell
```
Expected: clean build. The placeholder appears verbatim in `web/shell/dist/index.html`.

- [ ] **Step 3: Verify the placeholder survives Vite**

```bash
grep -c '__LIVEBOARD_CONFIG__' web/shell/dist/index.html
```
Expected: `1`.

If Vite mangles the script (it shouldn't — it's an inline non-module script with no imports), wrap the placeholder in a non-module `<script>` block (already the case in Step 1).

- [ ] **Step 4: Commit**

```bash
git add web/shell/index.html
git commit -m "feat(shell): add LIVEBOARD_CONFIG placeholder for adapter selection

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

(Don't commit `web/shell/dist/` — it's gitignored.)

---

## Task 2: Shell `main.ts` reads config + branches

**Files:**
- Modify: `web/shell/src/main.ts`

- [ ] **Step 1: Replace `main.ts`**

Replace the entire body with:
```ts
import { Broker } from '../../shared/src/broker.js'
import { LocalAdapter } from '../../shared/src/adapters/local.js'
import { ServerAdapter } from '../../shared/src/adapters/server.js'
import { BrowserStorage } from '../../shared/src/adapters/local-storage-driver.js'
import { shellTransport } from '../../shared/src/transports/post-message.js'
import type { BackendAdapter } from '../../shared/src/adapter.js'

interface LiveboardConfig {
  adapter: 'local' | 'server'
  baseUrl?: string
}

const SHELL_VERSION = '0.0.1'

function readConfig(): LiveboardConfig {
  const raw = (window as unknown as { __LIVEBOARD_CONFIG__?: LiveboardConfig }).__LIVEBOARD_CONFIG__
  if (raw && (raw.adapter === 'local' || raw.adapter === 'server')) return raw
  return { adapter: 'local' }
}

function makeAdapter(cfg: LiveboardConfig): BackendAdapter {
  if (cfg.adapter === 'server') {
    return new ServerAdapter({ baseUrl: cfg.baseUrl ?? '/api/v1' })
  }
  return new LocalAdapter(new BrowserStorage())
}

function bootstrap(): void {
  const iframe = document.getElementById('renderer') as HTMLIFrameElement | null
  if (!iframe) throw new Error('renderer iframe not found')

  const params = new URLSearchParams(window.location.search)
  const mode = params.get('renderer') ?? 'default'
  iframe.src = mode === 'stub' ? '/app/renderer-stub/' : '/app/renderer/default/'

  const adapter = makeAdapter(readConfig())
  const transport = shellTransport(iframe, window.location.origin)
  const broker = new Broker(transport, adapter, { shellVersion: SHELL_VERSION })

  window.addEventListener('beforeunload', () => broker.close())
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', bootstrap)
} else {
  bootstrap()
}
```

- [ ] **Step 2: Build + typecheck**

```bash
cd /Users/htruong/code/htruong/liveboard && make shell
```
Expected: clean build. Vite-emitted shell bundle now references `ServerAdapter` (lazy code-paths still bundle).

- [ ] **Step 3: Commit**

```bash
git add web/shell/src/main.ts
git commit -m "feat(shell): read LIVEBOARD_CONFIG and branch LocalAdapter vs ServerAdapter

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Go interceptor for shell `index.html`

**Files:**
- Modify: `internal/api/server.go`

The current `mountShellRoutes` serves the entire `/app/*` static tree via a single `http.FileServer`. We add an interceptor for the index page only — asset URLs (hashed Vite chunks) pass through unchanged.

- [ ] **Step 1: Add the helper**

In `internal/api/server.go`, near the top of the file (with other helpers), add:
```go
import (
    "bytes"
    // ...existing imports
)

const liveboardConfigMarker = `/*__LIVEBOARD_CONFIG__*/ { adapter: 'local' }`
const liveboardConfigServer = `{ adapter: 'server', baseUrl: '/api/v1' }`

func injectLiveboardConfig(html []byte) []byte {
    return bytes.Replace(html, []byte(liveboardConfigMarker), []byte(liveboardConfigServer), 1)
}
```

(`bytes` may already be imported; if not, add it.)

- [ ] **Step 2: Update `mountShellRoutes` to intercept the index**

Read `internal/api/server.go` around line 351. Replace the body of `mountShellRoutes` with:
```go
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

    serveIndex := func(w http.ResponseWriter, req *http.Request) {
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
        // Intercept the index for adapter-config injection.
        if path == "/app/" || path == "/app/index.html" {
            serveIndex(w, req)
            return
        }
        shellHandler.ServeHTTP(w, req)
    })
}
```

- [ ] **Step 3: Run + lint**

```bash
cd /Users/htruong/code/htruong/liveboard && make shell && go build ./... && make lint
```
Expected: clean build, no new lint errors.

- [ ] **Step 4: Don't commit yet** — wait for Task 4 to add the test, so the commit is a single coherent change.

---

## Task 4: Go test asserting replacement

**Files:**
- Modify: `internal/api/server_shell_test.go` (or create if missing).

- [ ] **Step 1: Read the existing test file**

```bash
cat internal/api/server_shell_test.go 2>/dev/null || echo "(file does not exist)"
```

- [ ] **Step 2: Write the test**

If the existing test already has helpers for the shell flag, append. Otherwise create the file with this content:
```go
package api

import (
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"

    "github.com/and1truong/liveboard/internal/board"
    "github.com/and1truong/liveboard/internal/workspace"
)

func newTestServerWithShell(t *testing.T) *httptest.Server {
    t.Helper()
    t.Setenv("LIVEBOARD_APP_SHELL", "1")
    dir := t.TempDir()
    ws := workspace.Open(dir)
    eng := board.NewEngine(ws)
    s := NewServer(ws, eng, true /* noCache */, false, false, "test", "", "")
    return httptest.NewServer(s.router)
}

func newTestServerNoShell(t *testing.T) *httptest.Server {
    t.Helper()
    os.Unsetenv("LIVEBOARD_APP_SHELL")
    dir := t.TempDir()
    ws := workspace.Open(dir)
    eng := board.NewEngine(ws)
    s := NewServer(ws, eng, true, false, false, "test", "", "")
    return httptest.NewServer(s.router)
}

func TestShellIndex_InjectsServerConfig(t *testing.T) {
    srv := newTestServerWithShell(t)
    defer srv.Close()

    res, err := http.Get(srv.URL + "/app/")
    if err != nil { t.Fatalf("get: %v", err) }
    defer res.Body.Close()
    if res.StatusCode != http.StatusOK {
        t.Fatalf("status = %d", res.StatusCode)
    }
    body := readBody(t, res)
    if !strings.Contains(body, "adapter: 'server'") {
        t.Errorf("expected 'adapter: \\'server\\'' in body; got first 500 chars: %s", truncate(body, 500))
    }
    if strings.Contains(body, "/*__LIVEBOARD_CONFIG__*/") {
        t.Errorf("placeholder marker should be replaced, but is still present")
    }
}

func TestShellIndex_ExplicitIndexPath(t *testing.T) {
    srv := newTestServerWithShell(t)
    defer srv.Close()
    res, err := http.Get(srv.URL + "/app/index.html")
    if err != nil { t.Fatalf("get: %v", err) }
    defer res.Body.Close()
    body := readBody(t, res)
    if !strings.Contains(body, "adapter: 'server'") {
        t.Errorf("expected injection on /app/index.html as well")
    }
}

func TestShell_NotMountedWithoutFlag(t *testing.T) {
    srv := newTestServerNoShell(t)
    defer srv.Close()
    res, err := http.Get(srv.URL + "/app/")
    if err != nil { t.Fatalf("get: %v", err) }
    defer res.Body.Close()
    if res.StatusCode != http.StatusNotFound {
        t.Errorf("expected 404 without flag, got %d", res.StatusCode)
    }
}

func readBody(t *testing.T, res *http.Response) string {
    t.Helper()
    var sb strings.Builder
    buf := make([]byte, 4096)
    for {
        n, err := res.Body.Read(buf)
        if n > 0 { sb.Write(buf[:n]) }
        if err != nil { break }
    }
    return sb.String()
}

func truncate(s string, n int) string {
    if len(s) <= n { return s }
    return s[:n] + "…"
}
```

If existing helpers in the file already do `newTestServer` / `readBody` / `truncate`, reuse them — drop the duplicates.

If `NewServer` signature differs, match the existing call shape used by other tests in the same file.

- [ ] **Step 3: Run**

```bash
go test ./internal/api/ -run TestShell -v
```
Expected: 3 tests pass.

If `TestShell_NotMountedWithoutFlag` fails because the test runner already inherited `LIVEBOARD_APP_SHELL=1` from a parent shell, the explicit `os.Unsetenv` in `newTestServerNoShell` should override. If still flaky, switch the test to use `t.Setenv("LIVEBOARD_APP_SHELL", "")`.

- [ ] **Step 4: Lint + commit (Tasks 3 + 4 together)**

```bash
make lint
git add internal/api/server.go internal/api/server_shell_test.go
git commit -m "feat(api): inject LIVEBOARD_CONFIG into shell index when flag is set

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Manual smoke

Not a code change.

- [ ] **Step 1: Server-backed mode**

```bash
make adapter-test
```
Open <http://localhost:7070/app/>. In Devtools:
- **Network → XHR/Fetch**: see calls to `/api/v1/boards`, `/api/v1/workspace`. EventSource to `/api/v1/events`.
- **Console**: no errors (apart from usual Radix warnings).

Verify mutations land on disk:
1. Add a card to the welcome board → modal closes after Save.
2. From another shell:
   ```bash
   cat ./demo/welcome.md
   ```
   The new card title appears in the file.

- [ ] **Step 2: Cross-tab sync**

Open a second tab against `/app/`. Mutate in tab A → tab B's sidebar updates without page reload. SSE working end-to-end.

- [ ] **Step 3: Offline / reconnect**

Stop `make adapter-test` (Ctrl+C in its terminal). In the browser:
- Mutations show error toasts ("Server error — try again" or similar).
- Console shows EventSource reconnect attempts every ~3s.

Restart the server. The renderer recovers without page reload (next mutation succeeds; SSE re-subscribes).

- [ ] **Step 4: Online / local mode**

```bash
make online && bash online/build.sh   # if not already built
# Open the resulting bundle via any static server (e.g. python3 -m http.server)
```
The shell loads via static-only hosting → `__LIVEBOARD_CONFIG__` keeps its `local` default → `LocalAdapter` is used → browser localStorage is the store.

Boards created here do NOT show up in the server-backed tab — they're separate stores. Expected.

- [ ] **Step 5: Stub harness still works under both modes**

`/app/?renderer=stub` loads the P3 integration harness and exercises the broker round-trips — confirms the harness works against either adapter.

- [ ] **Step 6: Report**

If anything fails, capture step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| Placeholder `<script>` in `web/shell/index.html` | 1 |
| Shell `main.ts` reads `__LIVEBOARD_CONFIG__` and branches | 2 |
| Go interceptor patches index when flag is on | 3 |
| Test: replacement happens with flag | 4 |
| Test: no replacement / 404 without flag | 4 |
| Manual smoke (server, cross-tab, reconnect, online, stub) | 5 |

## Notes for implementer

1. **The placeholder must survive Vite** — keep it inside a non-module `<script>` block. Vite leaves inline non-module scripts alone in `index.html`. Confirm with `grep '__LIVEBOARD_CONFIG__' web/shell/dist/index.html` after `make shell`.
2. **`bytes.Replace(..., 1)`** — replace exactly one occurrence. If the marker drifts and matches zero occurrences, the served HTML still contains `adapter: 'local'` and the test catches it.
3. **`indexPatched` is built once at handler init** — mutating shell `index.html` requires a server restart. Acceptable for a config injection.
4. **`NewServer` signature** in tests — match whatever the existing test helpers use. The 8-arg form here mirrors what's in production code. If the existing test file has a `newTestServer(t)` helper, reuse it instead of duplicating.
5. **`make lint`** runs after every Go change. Existing baseline lint errors are not yours; only fix new ones you introduce.
6. **No commit amending** — forward-only commits.

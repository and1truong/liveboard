# P5.2 — Shell Wiring (LocalAdapter ↔ ServerAdapter) — Design

## Goal

Wire the shell to choose between `LocalAdapter` (browser localStorage) and `ServerAdapter` (P5.1 HTTP+SSE) at boot time, driven by a runtime config object that the Go server injects when serving the shell. The online / browser-only build keeps `LocalAdapter` as the default.

**Shippable value:** `make adapter-test` opens `/app/` against the Go server's filesystem workspace. Mutations land as edits to `./demo/*.md`. SSE keeps two tabs in sync. The branching adapter story is closed.

## Scope

**In:**
- A `<script>` placeholder in `web/shell/index.html` defining `window.__LIVEBOARD_CONFIG__` with a `local` default.
- A small Go interceptor that string-replaces the placeholder when the shell is served, setting `adapter: 'server'`.
- `web/shell/src/main.ts` reads `window.__LIVEBOARD_CONFIG__` at bootstrap and instantiates the right adapter.
- A Go test asserting the replacement happens behind the `LIVEBOARD_APP_SHELL=1` flag and not otherwise.

**Out:**
- Adapter switch at runtime (would require teardown of broker/EventSource — boot-time only).
- Multiple workspace IDs / multi-server (single `baseUrl` for now).
- Auth token injection (no auth in P5).
- Cookie or session-based config persistence.
- Capability negotiation between adapter and renderer (renderer treats both adapters identically).

## Architecture

```
web/shell/index.html  (built by Vite)
       │
       ├─ <script>window.__LIVEBOARD_CONFIG__ = /*__LIVEBOARD_CONFIG__*/ { adapter: 'local' };</script>
       └─ Vite-emitted script (main.ts → bundle)

Serving paths:
    Online build / dev shell      → file served as-is → adapter: 'local'
    Go binary + LIVEBOARD_APP_SHELL=1 → handler intercepts
                                     → reads embedded HTML
                                     → string-replaces placeholder
                                     → emits adapter: 'server', baseUrl: '/api/v1'

main.ts at boot:
    cfg = window.__LIVEBOARD_CONFIG__ ?? { adapter: 'local' }
    adapter = cfg.adapter === 'server'
        ? new ServerAdapter({ baseUrl: cfg.baseUrl ?? '/api/v1' })
        : new LocalAdapter(new BrowserStorage())
    new Broker(transport, adapter, ...)
```

The renderer iframe still talks postMessage to the broker — no renderer changes. The broker delegates to whichever adapter the shell built.

## File structure

**Modified:**
- `web/shell/index.html` — add the placeholder script tag.
- `web/shell/src/main.ts` — read config, branch adapter selection.
- `internal/api/server.go` (or wherever the shell `/app/` route lives) — intercept the shell `index.html` to do the string replacement.
- `internal/api/server_shell_test.go` — assert the replacement occurs only when the flag is set.

**No new files.**

## Wire shape

The TS-side config object:
```ts
interface LiveboardConfig {
  adapter: 'local' | 'server'
  baseUrl?: string  // server adapter only; defaults to '/api/v1'
}
```

The placeholder marker is a JS comment so the file remains valid JavaScript when served unmodified:
```html
<script>window.__LIVEBOARD_CONFIG__ = /*__LIVEBOARD_CONFIG__*/ { adapter: 'local' };</script>
```

Go's replacement target is the literal substring:
```
/*__LIVEBOARD_CONFIG__*/ { adapter: 'local' }
```
…replaced by:
```
{ adapter: 'server', baseUrl: '/api/v1' }
```

If the marker disappears (someone edits `index.html` and removes the comment), the Go test catches the regression: it asserts the served HTML contains `adapter: 'server'` when the flag is on.

## Component contract

### Shell `main.ts`

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

### Go-side handler

The existing shell mount point (around `mountShellRoutes` in `internal/api/server.go`) currently serves `web/shell/dist/*` directly from the embedded FS. Add an interceptor for `GET /app/` and `GET /app/index.html` that:

1. Reads the embedded `index.html` once at handler init (or per request — small file, no big deal).
2. Calls `strings.Replace` once on the placeholder marker.
3. Writes the modified HTML to the response with the original Content-Type.

```go
func shellIndexHandler(htmlBytes []byte) http.HandlerFunc {
    const marker = `/*__LIVEBOARD_CONFIG__*/ { adapter: 'local' }`
    const replacement = `{ adapter: 'server', baseUrl: '/api/v1' }`
    out := bytes.Replace(htmlBytes, []byte(marker), []byte(replacement), 1)
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Write(out)
    }
}
```

Mounted only when `LIVEBOARD_APP_SHELL=1`, intercepting before the static FS handler so the modified version wins.

## Testing

- `internal/api/server_shell_test.go` (existing or new):
  - Without `LIVEBOARD_APP_SHELL` set: GET `/app/` → 404 (current behavior — shell isn't mounted).
  - With `LIVEBOARD_APP_SHELL=1`: GET `/app/` → 200, body contains `adapter: 'server'` AND does NOT contain the literal marker comment.
  - With `LIVEBOARD_APP_SHELL=1`: GET `/app/index.html` (explicit path) → same as above.

- TS-side: no automated test. The two-line config-read + adapter-make is trivial; manual smoke covers it.

## Manual smoke

1. **Server-backed mode**: `make adapter-test` → open <http://localhost:7070/app/>.
   - Devtools Network tab shows `XHR` calls to `/api/v1/boards`, `/api/v1/workspace`, and an `EventSource` connection to `/api/v1/events`.
   - Mutate (add card, rename board, etc.) — the underlying file in `./demo/*.md` is updated on disk (verify by `cat`).
   - Open a second tab against the same URL — SSE pushes propagate (sidebar refreshes when a board is created/deleted in the other tab).
   - Stop Go server → console shows EventSource reconnect attempts; mutations show error toasts.
   - Restart → app recovers without page reload.

2. **Local mode (online build)**: `make online`, serve the resulting bundle via any static server, open in a browser. Should still work using browser localStorage.

3. **Stub harness still works** under both modes (`?renderer=stub`).

## Risks

- **Stale embed**: if `web/shell/dist/index.html` isn't re-built after the placeholder change, the marker won't match and the replacement no-ops. Go test catches it. Mitigation: `make adapter-test` always runs `make shell` before serving.
- **Marker drift via reformatting**: if a formatter re-spaces the comment, the literal won't match. Mitigation: keep the marker on a single line; document in `index.html` with an HTML comment above it.
- **Future config keys**: adding more fields (e.g. auth token) requires the Go interceptor to know the new shape. For P5.2 we ship two fields; future expansions evolve the interceptor.
- **Online build accidentally serving 'server' mode**: the placeholder default is `local`; the Go interceptor only activates under `LIVEBOARD_APP_SHELL=1`. Online build doesn't go through Go at all, so the placeholder stays untouched.
- **Vite chunk URLs**: shell main bundle stays embedded under `/app/assets/*` and is hashed; only `index.html` is intercepted. Asset requests pass through unchanged.

## Open questions

None blocking. Pre-decided:
- Boot-time config only; no runtime adapter swap.
- Single `baseUrl` per config; no multi-workspace.
- Default = `local`; Go interceptor flips to `server`.
- TS branching is two lines; no automated TS test.

## Dependencies on prior work

- P5.0: `/api/v1/*` HTTP + SSE surface that `ServerAdapter` consumes.
- P5.1: `web/shared/src/adapters/server.ts` exists with full `BackendAdapter` implementation.
- Existing shell scaffold: `web/shell/index.html`, `web/shell/src/main.ts`, broker + transport plumbing.
- Existing `internal/api/server.go` shell-mount logic gated on `LIVEBOARD_APP_SHELL`.

## Dependencies on later work

None — P5.2 closes P5.

# How to Profile LiveBoard

LiveBoard embeds Go's `net/http/pprof` handlers, gated behind an environment variable.

## Enable Profiling

Set `LIVEBOARD_PPROF=1` before launching:

```bash
# CLI server
LIVEBOARD_PPROF=1 liveboard serve

# Desktop app (macOS)
LIVEBOARD_PPROF=1 open /Applications/LiveBoard.app
# or directly:
LIVEBOARD_PPROF=1 ./build/bin/LiveBoard.app/Contents/MacOS/LiveBoard
```

On startup you'll see: `pprof profiling enabled at /debug/pprof/`

The desktop app binds to a random port — check the log output for the URL.

## Profiling Endpoints

All endpoints live under `/debug/pprof/`. Replace `localhost:7070` with the actual address.

| Endpoint | What it captures |
|---|---|
| `/debug/pprof/` | Index page listing all profiles |
| `/debug/pprof/profile?seconds=30` | CPU profile |
| `/debug/pprof/heap` | Heap allocations |
| `/debug/pprof/goroutine` | Goroutine stacks |
| `/debug/pprof/trace?seconds=5` | Execution trace |
| `/debug/pprof/allocs` | Past memory allocations |
| `/debug/pprof/block` | Blocking contention |
| `/debug/pprof/mutex` | Mutex contention |

## Common Workflows

### Slow first page load (execution trace)

Best for understanding where wall-clock time goes during the initial request:

```bash
# Start capturing a 5-second trace, then load the page in the browser
curl -o trace.out http://localhost:7070/debug/pprof/trace?seconds=5
go tool trace trace.out
```

### CPU hotspots

```bash
go tool pprof http://localhost:7070/debug/pprof/profile?seconds=30
# interactive: top, list <func>, web (opens SVG flamegraph)
```

### Memory usage

```bash
go tool pprof http://localhost:7070/debug/pprof/heap
# interactive: top, list <func>
```

### Browse in browser

Open `http://localhost:7070/debug/pprof/` directly for the index page with links to all profiles.

## Security

Profiling endpoints expose internal runtime details. Only enable on trusted networks — never in production deployments facing the internet.

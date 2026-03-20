---
name: liveview
description: "Build and modify LiveView UI features using jfyne/live in Go. Use this skill whenever the user wants to add a new page, handler, event, component, template, or client-side interaction to the LiveView web UI. Also trigger when the user mentions LiveView, live events, live-click, live-submit, PubSub, real-time updates, or wants to add interactive UI behavior to the board. Trigger even for seemingly small UI tasks like adding a button or form — they all involve the LiveView event lifecycle."
---

# LiveView Development Skill

This project uses `github.com/jfyne/live` (v0.16.3), a Go implementation of the LiveView pattern. The server owns all state and communicates with the browser over WebSocket. The client sends events, the server re-renders HTML, and diffs are pushed back.

## Architecture Overview

```
internal/
  web/
    handler.go       — Handler struct, NewHandler, withAssignsRenderer, BoardListHandler/BoardViewHandler factories
    board_view.go    — BoardViewModel struct, mountBoardView, all board event handlers
    board_list.go    — BoardListModel struct, mountBoardList, board list event handlers
    events.go        — Event name constants (if used)
  templates/
    layout.html      — App shell (sidebar, scripts, CSS link)
    board_view.html  — Kanban board template
    board_list.html  — Board list template
    settings.html    — Settings page (traditional HTTP, not LiveView)
  api/
    server.go        — chi router, route registration, LiveView handler mounting
web/
  css/board.css      — All styles (CSS variables for theming)
  js/drag.js         — Client-side JS (drag-and-drop, context menus, modals, inline editing)
```

## How to Add a New LiveView Feature

Every LiveView feature touches up to 5 places. Follow this checklist:

### 1. Define the Model (State)

The model is a plain Go struct. It gets passed directly to the template via `withAssignsRenderer`. Add fields for any UI state you need.

```go
// in internal/web/your_view.go
type YourViewModel struct {
    Title    string
    Items    []Item
    Error    string
    // UI state fields (form visibility, selected item, etc.)
    ShowForm bool
}
```

The model is returned from every handler — mount, event handlers, and PubSub handlers. LiveView diffs the rendered HTML and sends only what changed.

### 2. Write the Mount Handler

Mount runs on initial page load (HTTP) and on WebSocket reconnect. Extract URL params here.

```go
func (h *Handler) mountYourView(ctx context.Context, s *live.Socket) (interface{}, error) {
    // On reconnect, reuse state from existing assigns
    if s != nil {
        if m, ok := s.Assigns().(YourViewModel); ok && m.SomeField != "" {
            return m, nil
        }
    }

    // Initial HTTP mount: extract params from URL
    req := live.Request(ctx)
    // ... parse req.URL.Path or req.URL.Query()

    return YourViewModel{
        Title: "Page Title",
        Items: loadItems(),
    }, nil
}
```

### 3. Write Event Handlers

Every event handler has the same signature. Extract params, do business logic, return updated model.

```go
func (h *Handler) handleYourEvent(_ context.Context, _ *live.Socket, p live.Params) (interface{}, error) {
    // Extract params (sent from template or JS)
    name, ok := p["name"].(string)
    if !ok || name == "" {
        return YourViewModel{Error: "Name is required"}, nil
    }

    // Do business logic
    err := h.eng.DoSomething(name)
    if err != nil {
        return YourViewModel{Error: err.Error()}, nil
    }

    // Optional: git commit
    h.commitWithHandling(path, "Do something")

    // Optional: broadcast to other clients
    h.publishBoardEvent(slug, "something_happened")

    // Return fresh model
    return h.yourViewModel()
}
```

Params come from two sources:
- Template attributes: `live-value-key="value"` sends `p["key"]`
- JavaScript: `window.Live.send("event", {key: "value"})` sends `p["key"]`

### 4. Register the Handler

Wire everything together in `handler.go`. Follow the existing pattern:

```go
func (h *Handler) YourViewHandler() http.Handler {
    handler := live.NewHandler(
        withAssignsRenderer(h.yourViewTpl),  // template parsed in NewHandler
    )
    handler.MountHandler = h.mountYourView
    handler.HandleEvent("your-event", h.handleYourEvent)
    handler.HandleEvent("another-event", h.handleAnotherEvent)

    // For real-time updates from other clients:
    handler.HandleSelf("board_update", h.handleBoardUpdate)

    return live.NewHttpHandler(context.Background(), handler,
        live.WithSocketStateStore(live.NewMemorySocketStateStore(context.Background())),
    )
}
```

Then add the route in `internal/api/server.go`:

```go
r.Handle("/your-path", s.liveHandler.YourViewHandler())
```

Also parse the template in `NewHandler()` in `handler.go`:

```go
h.yourViewTpl = template.Must(template.ParseFiles(layoutFile, filepath.Join(h.tmplDir, "your_view.html")))
```

### 5. Write the Template

Templates use Go `html/template` with LiveView data attributes. Create `internal/templates/your_view.html`:

```html
{{define "content"}}
<div class="your-view">
    {{if .Error}}
    <div class="error">{{.Error}}</div>
    {{end}}

    <!-- Button that sends an event -->
    <button live-click="your-event"
            live-value-item_id="{{.ID}}"
            live-value-name="{{.Slug}}">
        Click Me
    </button>

    <!-- Form that sends on submit -->
    <form live-submit="create-item"
          live-value-name="{{.Slug}}">
        <input type="text" name="title" placeholder="Title" />
        <button type="submit">Create</button>
    </form>

    <!-- Conditional rendering -->
    {{if .ShowForm}}
    <div class="form-container">...</div>
    {{end}}

    <!-- Loops -->
    {{range .Items}}
    <div class="item" data-id="{{.ID}}">{{.Title}}</div>
    {{end}}
</div>
{{end}}
```

**Template binding reference:**
- `live-click="event-name"` — sends event on click
- `live-submit="event-name"` — sends event on form submit (collects all named inputs)
- `live-value-key="value"` — attaches a key-value param to the event
- `live-change="event-name"` — sends event on input change (for validation)

Form inputs with `name="xyz"` attributes are automatically included in `live-submit` params as `p["xyz"]`.

### 6. Client-Side Interactivity

**Prefer Alpine.js** for all client-side interactivity (dropdowns, filtering, toggles, modals, inline editing). Use the **alpinejs skill** for Alpine.js syntax and patterns — it covers directives, magics, plugins, and idiomatic examples.

**How Alpine.js integrates with LiveView:**

Alpine manages local UI state (visibility, filtering, form input) while LiveView manages server state. They communicate through `window.Live.send()` from Alpine event handlers:

```html
{{define "content"}}
<div x-data="{ search: '' }">
    <!-- Alpine handles local filtering UI -->
    <input x-model="search" @keyup.debounce="window.Live.send('suggest', { query: search })" />

    <!-- Server-rendered results from LiveView -->
    {{range .Suggestions}}
    <div @click="window.Live.send('selected', { id: '{{.ID}}' })">
        {{.Name}}
    </div>
    {{end}}
</div>
{{end}}
```

The server handles `"suggest"` and `"selected"` events via `handler.HandleEvent()` as normal — Alpine just provides the reactive UI layer on top.

**When to use vanilla JS instead:** Only for low-level DOM manipulation that Alpine doesn't cover well, like drag-and-drop with complex pointer tracking. Legacy vanilla JS lives in `web/js/drag.js`.

**Sending events from vanilla JS:**

```javascript
window.Live.send("event-name", {
    key: "value",
    name: boardSlug
});
```

## PubSub for Real-Time Updates

Publish/subscribe pattern for syncing state across multiple browser tabs and users. Read `references/pubsub.md` for the publishing and subscribing patterns.

## Components (page.Component)

For reusable, self-contained UI pieces with their own state and events, use `github.com/jfyne/live/page`. Read `references/components.md` for the full pattern — it covers creating components, composing parent-child relationships, and scoped events.

## Error Handling

Three levels: model-based inline errors, handler-level `ErrorHandler`, and client-side JS hooks. Read `references/error-handling.md` for the full pattern with code examples and selection guidance.

## Dropdown / Autocomplete Inputs

When building chip-based inputs with dropdowns (tags, autocomplete, etc.), always bind **both `click` and `focus`** to show the dropdown:

```javascript
input.addEventListener("focus", function () { showDropdown(input.value); });
input.addEventListener("click", function () { showDropdown(input.value); });
```

**Why both?** Dropdown item selection typically uses `mousedown` with `preventDefault()` to avoid triggering blur on the input. This means the input never loses focus, so clicking it again won't re-fire `focus`. The `click` listener ensures the dropdown always reopens.

## Common Patterns in This Project

- **UI state toggles**: Use model fields like `ShowAddCard string` to control conditional rendering
- **Slug passing**: Every event includes `live-value-name="{{.BoardSlug}}"` so handlers know which board to operate on
- **Git integration**: Call `h.commitWithHandling()` or `h.commitRemoveWithHandling()` after mutations
- **Reload pattern**: After mutation, call `h.boardViewModel(slug)` to return a fresh model from disk

# LiveView Components (page.Component)

Components let you build reusable, self-contained UI pieces — each with its own state, event handlers, and render function. Think of them like React components but server-side.

This project currently uses the flat handler pattern (one big model per page). Components are useful when you need multiple independent instances of the same UI on a page, or when a piece of UI has its own lifecycle (e.g., a timer, a chat widget).

## Import

```go
import "github.com/jfyne/live/page"
```

## Creating a Component

A component has three lifecycle hooks wired via options:

```go
func NewMyComponent(ID string, h *live.Handler, s *live.Socket, someParam string) (*page.Component, error) {
    return page.NewComponent(ID, h, s,
        page.WithRegister(myRegister),          // wire up event handlers
        page.WithMount(myMount(someParam)),     // initialize state
        page.WithRender(myRender),              // render HTML
    )
}
```

Every component needs a **unique stable ID** (e.g., `"clock-1"`, `"chat-widget"`). LiveView uses this to track and diff the component independently.

## State

Just a plain struct:

```go
type MyComponentState struct {
    Label   string
    Count   int
    Active  bool
}
```

## Register (Event Handlers)

Bind events scoped to this component instance:

```go
func myRegister(c *page.Component) error {
    // Handle events from the client (clicks, form submits)
    c.HandleEvent("increment", func(_ context.Context, p live.Params) (any, error) {
        state := c.State.(*MyComponentState)
        state.Count++
        return state, nil
    })

    // Handle self-events (server-to-server, timers, PubSub)
    c.HandleSelf("tick", func(ctx context.Context, d any) (any, error) {
        state := c.State.(*MyComponentState)
        // update state from d
        return state, nil
    })

    return nil
}
```

Use `c.HandleEvent` for client-initiated events and `c.HandleSelf` for server-initiated events (timers, PubSub broadcasts).

## Mount

Initialize state when the component is created:

```go
func myMount(label string) page.MountHandler {
    return func(ctx context.Context, c *page.Component) error {
        c.State = &MyComponentState{
            Label: label,
            Count: 0,
        }

        // Optionally start a timer or background work on WebSocket connect
        if c.Socket.Connected() {
            go func() {
                time.Sleep(1 * time.Second)
                c.Self(ctx, c.Socket, "tick", time.Now())
            }()
        }

        return nil
    }
}
```

## Render

Output HTML for the component. Use `page.HTML` helper with Go templates:

```go
func myRender(w io.Writer, c *page.Component) error {
    return page.HTML(`
        <div class="my-component">
            <span>{{.Label}}: {{.Count}}</span>
            <button live-click="` + c.Event("increment") + `">+1</button>
        </div>
    `, c).Render(w)
}
```

`c.Event("increment")` scopes the event name to this component instance — so if you have 5 instances on the page, clicking one button only increments that component's counter.

## Composing: Parent + Children

A parent component holds children in its state:

```go
type PageState struct {
    Title    string
    Widgets  []*page.Component
}
```

Create children with `page.Init`:

```go
widget, err := page.Init(ctx, func() (*page.Component, error) {
    return NewMyComponent(
        fmt.Sprintf("widget-%d", len(state.Widgets)+1),
        c.Handler,
        c.Socket,
        "some param",
    )
})
if err != nil {
    return state, err
}
state.Widgets = append(state.Widgets, widget)
```

Render children in the parent's render function:

```go
// Using gomponents
g.Group(g.Map(state.Widgets, func(c *page.Component) g.Node {
    return page.Render(c)
}))

// Or using page.HTML template
// You'd need to call page.Render(child) for each child within the parent template
```

## When to Use Components vs. Flat Handlers

**Use flat handlers** (the current pattern) when:
- The page has one logical model
- Events are simple CRUD operations
- No need for multiple independent instances

**Use components** when:
- You need multiple independent instances of the same UI piece
- A UI piece has its own lifecycle (timers, polling)
- You want to encapsulate complex state + events into a reusable unit
- Different parts of the page need independent re-rendering

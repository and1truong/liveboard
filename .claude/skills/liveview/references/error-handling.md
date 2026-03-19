# Error Handling

There are three levels of error handling. Prefer the approach that fits the UX need.

## Level 1: Model-Based (inline errors in UI)

Return the model with `.Error` set; the template renders it conditionally. Best for validation errors the user can fix.

```go
func (h *Handler) handleCreate(_ context.Context, s *live.Socket, p live.Params) (interface{}, error) {
    name, _ := p["name"].(string)
    if name == "" {
        m := s.Assigns().(YourViewModel)
        m.Error = "Name is required"
        return m, nil  // nil error — keeps the socket alive
    }
    // ...
}
```

## Level 2: Handler-Level ErrorHandler (custom HTTP error responses)

Set `handler.ErrorHandler` to intercept errors returned from event handlers and write a custom HTTP response. Use `live.Writer(ctx)` to get the `http.ResponseWriter`.

```go
func (h *Handler) YourViewHandler() http.Handler {
    handler := live.NewHandler(withAssignsRenderer(h.yourViewTpl))
    handler.ErrorHandler = func(ctx context.Context, err error) {
        w := live.Writer(ctx)
        w.WriteHeader(http.StatusBadRequest)
        w.Write([]byte("bad request: " + err.Error()))
    }
    // ...
}
```

## Level 3: Client-Side Error Hook (JS alert/toast on event failure)

When an event handler returns a non-nil error, the library automatically emits an `"err"` event to the client. Catch it with a `live-hook` and `this.handleEvent`:

**Template:**
```html
<main live-hook="err">
    <button live-click="problem" live-value-msg="details">Do risky thing</button>
</main>

<script>
    window.Hooks = {
        "err": {
            mounted: function() {
                this.handleEvent("err", function(data) {
                    console.error(data);
                    window.alert(data.err);  // or show a toast
                });
            }
        }
    };
</script>
```

**Go handler — just return the error:**
```go
handler.HandleEvent("problem", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
    return nil, fmt.Errorf("something went wrong")
})
```

The `data.err` string on the client side contains the error message from the Go handler.

## Preference: Choose the right level

- **Validation / user-fixable errors** → Level 1 (model `.Error` field). Keeps socket alive, user sees inline feedback.
- **Fatal / unrecoverable errors on mount or render** → Level 2 (`ErrorHandler`). Returns an HTTP error page.
- **Event errors that should notify the user without breaking the page** → Level 3 (client-side `"err"` hook). The socket stays connected, the user gets a JS-driven notification.

Do NOT return `(nil, err)` from event handlers unless you have Level 2 or Level 3 wired up — an unhandled error will drop the WebSocket connection with no user feedback.

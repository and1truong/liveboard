# PubSub for Real-Time Updates

When multiple browser tabs or users view the same board, PubSub keeps them in sync.

## Publishing

In any event handler after a mutation:

```go
h.publishBoardEvent(slug, "descriptive_action")
```

This calls `h.pubsub.Publish()` which broadcasts to all connected sockets.

## Subscribing

In the handler factory:

```go
handler.HandleSelf("board_update", h.handleBoardUpdate)
```

The `HandleSelf` handler receives the broadcast and returns a fresh model, causing a re-render for all connected clients.

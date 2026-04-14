# Move Card Across Boards — Design

## Goal

Allow a user to move a card from one board into a column of another board, from the card modal and the right-click quick-edit menu.

## User Surfaces

- **Card modal** — a "Move to board…" row with cascading `<select>`s: destination board, then destination column. A "Move" button commits.
- **Quick-edit context menu** — a "Move to board ▸" submenu. Level 1 lists boards; level 2 lists columns for the chosen board.
- Command palette and REST API parity are in scope for the REST endpoint (below), but no command-palette entry in this iteration.

## Behavior

- The card lands at the **top** of the destination column. (Columns in LiveBoard have no persistent sort setting — sorting is an explicit user action via `SortColumn` — so there is nothing to defer to.)
- Tags and members on the moved card that are **not** present on the destination board's frontmatter lists are auto-added to those lists.
- All other card metadata (body, priority, due, assignee, custom keys, completed state) is preserved verbatim.

## Architecture

### Engine method

New method in `internal/board/board.go`:

```go
func (e *Engine) MoveCardToBoard(
    srcPath string, srcVersion int, srcColIdx, cardIdx int,
    dstPath, dstColumn string,
) error
```

Implementation uses **two sequential `MutateBoard` calls**, matching existing concurrency patterns. No cross-board locking — each board mutation is independently atomic via its own per-board mutex.

Sequence:

1. **Read source card** — a lock-free disk read via `parser.Parse` to snapshot the card. (Actual version check happens in step 3 under the source lock.)
2. **`MutateBoard(dstPath, -1, fn)`** — target-side mutation. Version bypass (`-1`) because the client never saw the target board's version. `fn` resolves the destination column by name (error if missing), inserts the card copy (top, or per column sort setting), and merges any missing tags/members into target frontmatter.
3. **`MutateBoard(srcPath, srcVersion, fn)`** — source-side mutation. Version-checked against `srcVersion` to catch drift. `fn` removes the card at `(srcColIdx, cardIdx)`.
4. SSE publish to **both** board channels on full success.

### Failure modes

- **Step 2 fails** (target column missing, target disk write error, etc.) — neither board mutated. Return wrapped error.
- **Step 3 fails after step 2 succeeded** — card is duplicated (exists on both boards). Return a wrapped error so the handler can surface a clear message; user re-runs to remove from source. No data loss.
- **Same-board / same-column move** — rejected with a clear error to prevent confusion with the existing in-board move.

### Sentinel errors

- Target column not found → `board.ErrNotFound` (existing sentinel, wrapped with context).
- Source version conflict → `board.ErrVersionConflict` (existing sentinel; web path already handles auto-retry).

## Handlers

### Web

`POST /board/{slug}/cards/move-to-board` in `internal/web/board_card_handlers.go`. Form fields: `col_idx` (source column index), `card_idx` (card index within that column), `dst_board` (slug), `dst_column` (name). On success, returns the standard `#board-content` partial for the **source** board (user stays on source). SSE handles the target board for anyone viewing it.

### REST API

`POST /api/boards/{slug}/cards/move-to-board` in `internal/api/cards.go`. JSON body: `{src_col_idx, card_idx, dst_board, dst_column}`. Returns `204 No Content` on success (no board JSON body). Status codes via existing `handleError`:
- 204 on success
- 404 on missing target board or column
- 409 on source version conflict
- 400 on malformed input

### Board-list-lite endpoint

`GET /api/boards/list-lite` — returns `[{slug, name, columns: [name…]}]` for populating the cascading selects. Used by both the modal and the quick-edit submenu. Frontend caches the result in an Alpine store for the session.

## Frontend Wiring

- `web/js/liveboard.card-modal.js` — add the "Move to board…" section with two `<select>`s; column select repopulates on board change.
- `web/js/liveboard.drag.js` (`quickEdit`) — add the "Move to board ▸" cascading submenu.
- New Alpine store entry `boardsLite` populated lazily on first open; no invalidation needed for this iteration (stale board/column lists just mean the move may return an error, which is recoverable).
- Error surface for post-step-2 failure: banner reading "Card moved to {target} but could not be removed from the source — please delete it manually."

## Testing

### Engine (`internal/board/board_test.go`)

- Happy path: card removed from source, inserted at top of target; both versions bumped.
- Missing tags/members on target → auto-merged into target frontmatter.
- Target column not found → `ErrNotFound`, neither board mutated.
- Source version conflict → `ErrVersionConflict` after target already written; test documents the duplicate state.
- Same-board same-column move → rejected.
- All card metadata preserved (body, priority, due, assignee, custom keys, completed state).

### Web handlers (`internal/web/web_test.go`)

- Returns source-board `#board-content` partial on success.
- Invalid `dst_board` → error banner.
- Invalid `dst_column` → error banner.

### REST API (`internal/api/server_test.go`)

- 200 on success, body = updated source board.
- 404 on missing target board / column.
- 409 on source version conflict.

### Manual smoke

- Move via card modal.
- Move via quick-edit submenu.
- Target board open in a second browser window → SSE update received.

## Out of Scope

- Command palette entry for cross-board move.
- Bulk move (multiple cards at once).
- Undo.
- Cross-workspace moves.

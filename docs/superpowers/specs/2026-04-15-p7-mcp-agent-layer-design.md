# P7 — MCP Agent Layer Expansion — Design

## Goal

Close the biggest gaps between LiveBoard's MCP surface and the adapter/HTTP surface so agents (Claude Desktop, Claude Code, external LLM tooling) can effectively find, reference, and mutate cards. Adds four new tools (`search`, `backlinks`, `rename_board`, `resolve_card`) and optional `card_id` addressing on the five card-mutating tools (`show_card`, `edit_card`, `delete_card`, `move_card`, `complete_card`).

**Shippable value:** agents get full-text search, cross-board backlink navigation, and stable-ID card references. An agent that stashes a `card_id` from a previous session can still find and mutate that card after board reorders.

## Scope

**In:**
- New MCP tool `search(query, limit?)` wrapping `search.Index.Search`.
- New MCP tool `backlinks(card_id)` wrapping `search.Index.Backlinks`.
- New MCP tool `rename_board(board, new_name)` wrapping `Workspace.RenameBoard`.
- New MCP tool `resolve_card(card_id)` wrapping a new `search.Index.ResolveCard`.
- Optional `card_id` field on input schemas for `show_card`, `edit_card`, `delete_card`, `move_card`, `complete_card`.
- `lookupCard(slug, cardID, col, idx)` helper in `internal/mcp/helpers.go`.
- `search.Index.ResolveCard(cardID string) (*Hit, error)` helper (wildcard query on `card_id`).
- Thread `search.Index` into `mcp.Server` via constructor.
- Tests per new tool + an addressing-by-`card_id` test on `edit_card`.

**Out:**
- MCP tools for settings, `update_board_meta`, `update_board_members`, `update_board_icon`, `tag_card`, `reorder_card` (defer; low agent-demand).
- MCP streaming (SSE/event subscription).
- Auth/scope policy beyond what the existing MCP transport provides.
- UI / renderer changes (MCP is server-side only).
- Natural-language mutation planning (agents still pick tools themselves; P7 is plumbing, not an LLM).

## Architecture

```
existing mcp.Server
  ├─ workspace *workspace.Workspace
  ├─ engine    *board.Engine
  └─ server    *mcpsdk.Server

NEW
  └─ search    *search.Index     # injected via NewServer

Tool registration (existing pattern: mcpsdk.AddTool(srv, &Tool{Name:...}, handler))
  + tools_search.go
       ├─ search        → srv.search.Search
       ├─ backlinks     → srv.search.Backlinks
       └─ resolve_card  → srv.search.ResolveCard

  tools_board.go       adds rename_board registration
  tools_card.go        each card-mutating tool's handler calls lookupCard(slug, cardID, col, idx)
                        BEFORE calling the engine — rest of handler is unchanged.
  helpers.go           adds lookupCard.

search.Index           adds ResolveCard(cardID) (*Hit, error):
                         wildcard query field:card_id = cardID; first hit or nil.
```

All new tools reuse existing helpers (`errResult`, `jsonResult`, `textResult`). No new MCP SDK primitives.

## `lookupCard`

```go
// internal/mcp/helpers.go
func lookupCard(ws *workspace.Workspace, srch *search.Index,
    slug, cardID string, col, idx int) (string, int, int, error) {

    if cardID != "" {
        if srch == nil {
            return "", 0, 0, fmt.Errorf("search index unavailable; cannot resolve card_id")
        }
        hit, err := srch.ResolveCard(cardID)
        if err != nil {
            return "", 0, 0, err
        }
        if hit == nil {
            return "", 0, 0, fmt.Errorf("%w: card_id %q", board.ErrNotFound, cardID)
        }
        return hit.BoardID, hit.ColIdx, hit.CardIdx, nil
    }
    return slug, col, idx, nil
}
```

Each card-tool handler starts with:
```go
slug, col, idx, err := lookupCard(m.workspace, m.search, in.Board, in.CardID, in.ColIdx, in.CardIdx)
if err != nil { return errResult(err) }
// existing handler body uses slug/col/idx
```

`card_id` wins when both addressing modes are present. Documented in the tool description.

## Tool schemas

### `search`

```go
type searchInput struct {
    Query string `json:"query" jsonschema:"description=Query string (full-text over card title/body/tags/links)"`
    Limit int    `json:"limit,omitempty" jsonschema:"description=Maximum hits; defaults to 20"`
}

type searchHit struct {
    Board     string `json:"board"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardID    string `json:"card_id"`
    CardTitle string `json:"card_title"`
    Snippet   string `json:"snippet"`
}

type searchOutput struct {
    Hits []searchHit `json:"hits"`
}
```

### `backlinks`

```go
type backlinksInput struct {
    CardID string `json:"card_id" jsonschema:"description=Stable card id to find inbound links for"`
}

type backlinkHit struct {
    Board     string `json:"board"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardTitle string `json:"card_title"`
}

type backlinksOutput struct {
    Hits []backlinkHit `json:"hits"`
}
```

### `rename_board`

```go
type renameBoardInput struct {
    Board   string `json:"board" jsonschema:"description=Current board slug"`
    NewName string `json:"new_name" jsonschema:"description=New board name (will be slugified)"`
}

type renameBoardOutput struct {
    Board   string `json:"board"`  // new slug
    Name    string `json:"name"`
    Version int    `json:"version"`
}
```

### `resolve_card`

```go
type resolveCardInput struct {
    CardID string `json:"card_id" jsonschema:"description=Stable card id to locate"`
}

type resolveCardOutput struct {
    Board     string `json:"board"`
    BoardName string `json:"board_name"`
    ColIdx    int    `json:"col_idx"`
    CardIdx   int    `json:"card_idx"`
    CardTitle string `json:"card_title"`
}
```

`not found` → errResult (via `board.ErrNotFound`), which the MCP client renders as a structured error.

### Extended existing tools (card-mutating)

Every card-mutating tool's input struct gets:
```go
CardID string `json:"card_id,omitempty" jsonschema:"description=Alternate addressing: stable card id. Overrides board/col_idx/card_idx when set."`
```

No change to output shape. Handlers prepend the `lookupCard` call.

## `search.Index.ResolveCard`

```go
// internal/search/index.go
func (i *Index) ResolveCard(cardID string) (*Hit, error) {
    if cardID == "" {
        return nil, nil
    }
    q := bleve.NewTermQuery(cardID)
    q.SetField("card_id")
    sr := bleve.NewSearchRequestOptions(q, 1, 0, false)
    sr.Fields = []string{"board_id", "board_name", "col_idx", "card_idx", "card_id", "title"}
    res, err := i.idx.Search(sr)
    if err != nil {
        return nil, err
    }
    if len(res.Hits) == 0 {
        return nil, nil
    }
    h := res.Hits[0]
    return &Hit{
        BoardID:   getString(h.Fields, "board_id"),
        BoardName: getString(h.Fields, "board_name"),
        ColIdx:    getInt(h.Fields, "col_idx"),
        CardIdx:   getInt(h.Fields, "card_idx"),
        CardID:    getString(h.Fields, "card_id"),
        CardTitle: getString(h.Fields, "title"),
    }, nil
}
```

`card_id` is indexed by P6.3a's keyword analyzer setup (via the mapping extension in P6.3b). TermQuery on `card_id` with the exact ID returns the one doc or nothing.

## `mcp.Server` constructor change

Current:
```go
func NewServer(ws *workspace.Workspace, eng *board.Engine) *Server { ... }
```

New signature adds one param:
```go
func NewServer(ws *workspace.Workspace, eng *board.Engine, srch *search.Index) *Server { ... }
```

All callers (likely one — `internal/api/server.go`) pass the existing index instance. Nil-tolerant: when `srch == nil`, `search` / `backlinks` / `resolve_card` / `card_id` addressing return a descriptive error; the server keeps running.

## File structure

**New:**
- `internal/mcp/tools_search.go` — registrations for `search`, `backlinks`, `resolve_card`.

**Modified:**
- `internal/mcp/server.go` — constructor + struct field.
- `internal/mcp/helpers.go` — `lookupCard`.
- `internal/mcp/tools_board.go` — `rename_board` registration.
- `internal/mcp/tools_card.go` — input structs gain `CardID`; handlers prepend `lookupCard`.
- `internal/mcp/mcp_test.go` — new test cases (5+).
- `internal/search/index.go` — `ResolveCard` method.
- `internal/api/server.go` — pass `s.search` (the P6.2 index) into `mcp.NewServer`.

## Testing

Existing `setup(t)` in `mcp_test.go` constructs a test server. Extend it to also construct a search index and register board content so search is populated. Alternative: lazy — call `idx.UpdateBoard` within the test for the boards it cares about.

Representative cases:

```go
func TestSearch_MCP(t *testing.T) {
    srv, _ := setup(t)
    callTool(t, srv, "add_card", map[string]any{
        "board": "test-board", "column": "Todo", "title": "alpha-token",
    })
    r := callTool(t, srv, "search", map[string]any{"query": "alpha-token"})
    out := parseSearchOutput(t, r)
    if len(out.Hits) == 0 {
        t.Fatal("expected 1+ hit for 'alpha-token'")
    }
}

func TestBacklinks_MCP(t *testing.T) {
    srv, _ := setup(t)
    // Target card.
    callTool(t, srv, "add_card", map[string]any{
        "board": "test-board", "column": "Todo", "title": "target",
    })
    // Find target id.
    board := parseBoard(t, callTool(t, srv, "get_board", map[string]any{"board": "test-board"}))
    tgtID := board.Columns[0].Cards[0].ID
    if tgtID == "" { t.Fatal("target id empty") }

    // Source board with link.
    callTool(t, srv, "create_board", map[string]any{"name": "src"})
    callTool(t, srv, "add_card", map[string]any{
        "board": "src", "column": "Todo", "title": "source",
    })
    callTool(t, srv, "edit_card", map[string]any{
        "board": "src", "col_idx": 0, "card_idx": 0,
        "title": "source", "links": []string{"test-board:" + tgtID},
    })

    r := callTool(t, srv, "backlinks", map[string]any{"card_id": tgtID})
    out := parseBacklinksOutput(t, r)
    if len(out.Hits) != 1 || out.Hits[0].Board != "src" {
        t.Errorf("unexpected backlinks: %+v", out)
    }
}

func TestRenameBoard_MCP(t *testing.T) {
    srv, _ := setup(t)
    r := callTool(t, srv, "rename_board", map[string]any{
        "board": "test-board", "new_name": "Renamed",
    })
    out := parseRenameBoardOutput(t, r)
    if out.Board != "renamed" { t.Errorf("board = %q", out.Board) }
    // Old slug should 404.
    r2 := callTool(t, srv, "get_board", map[string]any{"board": "test-board"})
    assertErrorResult(t, r2, "not found")
}

func TestResolveCard_MCP(t *testing.T) {
    srv, _ := setup(t)
    callTool(t, srv, "add_card", map[string]any{
        "board": "test-board", "column": "Todo", "title": "findme",
    })
    board := parseBoard(t, callTool(t, srv, "get_board", map[string]any{"board": "test-board"}))
    id := board.Columns[0].Cards[0].ID
    r := callTool(t, srv, "resolve_card", map[string]any{"card_id": id})
    out := parseResolveCardOutput(t, r)
    if out.Board != "test-board" || out.CardTitle != "findme" {
        t.Errorf("resolved = %+v", out)
    }
}

func TestEditCard_ByCardID_MCP(t *testing.T) {
    srv, _ := setup(t)
    callTool(t, srv, "add_card", map[string]any{
        "board": "test-board", "column": "Todo", "title": "orig",
    })
    board := parseBoard(t, callTool(t, srv, "get_board", map[string]any{"board": "test-board"}))
    id := board.Columns[0].Cards[0].ID
    // Reorder or move (not strictly necessary for the test, but asserts indices don't matter):
    // ...
    callTool(t, srv, "edit_card", map[string]any{
        "card_id": id, "title": "edited",
    })
    after := parseBoard(t, callTool(t, srv, "get_board", map[string]any{"board": "test-board"}))
    if after.Columns[0].Cards[0].Title != "edited" {
        t.Errorf("edit didn't apply by card_id")
    }
}
```

`parseSearchOutput`, `parseBacklinksOutput`, etc. are small helpers that `json.Unmarshal` the tool's text result into the expected output struct. Follow the existing `resultText` pattern in `mcp_test.go`.

## Risks

- **`search.Index.ResolveCard` relies on `card_id` being indexed as exact-match**: P6.3b already added a keyword-analyzer mapping for `links`. `card_id` was added as a field in P6.3a but may still use the default analyzer. If TermQuery returns no hits, switch the mapping to keyword for `card_id` too — see the `New()` mapping setup in `internal/search/index.go`.
- **`setup(t)` in `mcp_test.go`**: its current shape doesn't know about a search index. Extending it is mechanical — build a new `search.Index`, index the seed board, pass to `mcp.NewServer`. Existing tests that don't use the new tools are unaffected.
- **Nil search index**: if the server starts without a search index (pathological), the new tools return a polite error instead of panicking. Nil-guard in `lookupCard` + each handler.
- **Addressing by card_id in `add_card`**: doesn't apply — new cards don't have IDs yet. `add_card` is intentionally excluded from the extension.
- **JSON schema generation**: the MCP SDK derives schemas from struct tags. Optional `CardID` must use `omitempty` + default zero-value (empty string). Verify schemas look correct during manual smoke against Claude Desktop or the MCP inspector.

## Open questions

None blocking. Pre-decided:
- Four new tools: `search`, `backlinks`, `rename_board`, `resolve_card`.
- `card_id` addressing on `show_card`, `edit_card`, `delete_card`, `move_card`, `complete_card`.
- Addressing conflict: `card_id` wins.
- `search.Index` threaded through `mcp.NewServer`.
- Settings / meta / tags / reorder tools deferred.

## Dependencies on prior work

- P5.0: `Workspace.RenameBoard` exists.
- P6.2: `search.Index` with `Search`, `Backlinks` methods and the bleve plumbing.
- P6.3a: `card.id` populated on every mutation-touched card.
- P6.3b: `Card.Links` + backlinks semantics + `SearchHit.CardID`.
- Existing MCP server structure (tools_board, tools_card, tools_column, helpers).

## Dependencies on later work

None — P7 closes the roadmap items the branch set out to cover.

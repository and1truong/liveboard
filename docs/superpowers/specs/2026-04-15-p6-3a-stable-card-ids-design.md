# P6.3a — Stable Card IDs — Design

## Goal

Add a stable 10-char alphanumeric `id` field to every card. Lazy migration: parsers tolerate absence, writers emit only when set, mutations always assign on touch. New cards get an ID at creation. No protocol additions; existing op shapes (`(col_idx, card_idx)` addressing) unchanged. Unblocks P6.3b cross-board linking by giving link targets a stable identity.

**Shippable value:** every mutation-touched card has a stable identity. Future link-resolution code in P6.3b can rely on `card.id` as the lookup key.

## Scope

**In:**
- `Card.id string` field on `pkg/models.Card` (Go) and `web/shared/src/types.ts` `Card` (TS).
- ID generator in both stacks: `internal/util/cardid.NewID()` (Go) and `web/shared/src/util/cardid.newCardId()` (TS), both producing 10-char strings from `[A-Za-z0-9]`.
- Parser: read `  id: <value>` from card metadata; ignore if absent.
- Writer: emit `id` as the first metadata line when present; omit otherwise.
- Engine helper `ensureCardID(c)` invoked from every op that creates or edits a card.
- TS `applyOp` mirror: assign ID for the same ops on the renderer-optimistic path.
- Tests across parser/writer/engine/applyOp.
- bleve `Doc` gains an `id` field so future linking lookups can resolve by card ID.

**Out:**
- Stable IDs for boards (boards still keyed by slug).
- Stable IDs for columns.
- An eager migration command.
- New ops or protocol changes — addressing remains positional.
- Backwards-compat handling for ID-bearing files written by other tools.
- The actual cross-board linking UI (P6.3b).

## ID generation

10 chars, alphabet `A-Za-z0-9` (62 chars). Collision space ≈ 8.4 × 10¹⁷ — adequate for any realistic workspace.

### Go (`internal/util/cardid/cardid.go`)

```go
package cardid

import (
    "crypto/rand"
    "encoding/binary"
)

const Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// NewID returns a 10-character alphanumeric ID drawn from crypto/rand.
func NewID() string {
    var raw [40]byte
    _, _ = rand.Read(raw[:])
    var b [10]byte
    for i := 0; i < 10; i++ {
        n := binary.BigEndian.Uint32(raw[i*4 : i*4+4])
        b[i] = Alphabet[int(n)%len(Alphabet)]
    }
    return string(b[:])
}
```

### TypeScript (`web/shared/src/util/cardid.ts`)

```ts
const ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'

export function newCardId(): string {
  const bytes = new Uint32Array(10)
  crypto.getRandomValues(bytes)
  let out = ''
  for (let i = 0; i < 10; i++) out += ALPHABET[bytes[i] % ALPHABET.length]
  return out
}
```

`crypto.getRandomValues` is in browser, bun, and Node 19+. No polyfill needed.

## Parser

Existing card metadata is `^  (\w+): (.+)$` (2-space indent). Adding `id` requires no grammar change — just parse the field:

```go
case "id":
    card.ID = value
```

Empty IDs are simply unread. No collision detection / validation at parse time (a hand-edited shorter ID still parses; the system tolerates whatever the file says).

## Writer

Existing writer sorts metadata keys alphabetically (or per a fixed order). Decision: emit `id` as the FIRST metadata line, before the alphabetical block, when `card.ID != ""`. This makes diffs predictable and visually surfaces the identity at the top of each card's metadata.

```go
if card.ID != "" {
    fmt.Fprintf(w, "  id: %s\n", card.ID)
}
// ... existing alphabetical metadata emission ...
```

If `card.ID == ""`, omit. This preserves the file as-is for cards that haven't been touched since the migration began.

## Engine `ensureCardID`

```go
package board

import "github.com/and1truong/liveboard/internal/util/cardid"

func ensureCardID(c *models.Card) {
    if c == nil { return }
    if c.ID == "" {
        c.ID = cardid.NewID()
    }
}
```

Call sites:

| Op | Where to call |
|---|---|
| `add_card` | After constructing the new card, before append. |
| `edit_card` | After locating the card to edit, before applying changes. |
| `complete_card` | Same. |
| `tag_card` | Same. |
| `move_card` | On the moved card. |
| `reorder_card` | On the reordered card. |
| `delete_card` | Skip — no need to mint an ID for a card we're removing. |
| `add_column`, `rename_column`, `delete_column`, `move_column`, `sort_column`, `toggle_column_collapse`, `update_board_*` | Skip — no card touched. |

## TS `applyOp` mirror

`web/shared/src/boardOps.ts` is the source of truth for renderer-side optimistic apply. Mirror the engine's behavior: every card-touching op calls `newCardId()` if `card.id` is missing.

```ts
import { newCardId } from './util/cardid.js'

function ensureCardId(c: Card): void {
  if (!c.id) c.id = newCardId()
}
```

Call sites parallel the Go ones.

## Optimistic vs server ID resolution

Since both Go and TS generate independent random IDs, an optimistic add_card emits a different ID than the server's eventual response. The renderer's `useBoardMutation.onSuccess` already replaces the cached board with the server's Board — the server's ID wins. Brief inconsistency window between optimistic apply and server confirm. P6.3b's link UI must not allow link creation against an optimistic-only ID; mitigations there.

## bleve `Doc` extension

`internal/search/index.go` `Doc` struct gains:
```go
type doc struct {
    BoardID   string   `json:"board_id"`
    BoardName string   `json:"board_name"`
    ColIdx    int      `json:"col_idx"`
    CardIdx   int      `json:"card_idx"`
    CardID    string   `json:"card_id"`
    Title     string   `json:"title"`
    Body      string   `json:"body"`
    Tags      []string `json:"tags"`
}
```

`UpdateBoard` populates `CardID` from `card.ID`. P6.3b's link-resolution path queries by `card_id`. P6.2's search results don't expose `card_id` yet (no consumer); plumbing wait until P6.3b.

## Testing

### Go

- `internal/util/cardid/cardid_test.go`:
  - `NewID()` returns 10 chars.
  - All chars in `Alphabet`.
  - 10000 calls produce no duplicates (probabilistic but reliable).

- `internal/parser/parser_test.go` (extension):
  - A card with `id: aBc1234XyZ` round-trips and `card.ID == "aBc1234XyZ"`.
  - A card without `id` parses with `card.ID == ""`.

- `internal/writer/writer_test.go`:
  - A card with `ID="abc"` emits `  id: abc` as the first metadata line.
  - A card with `ID=""` emits no `id:` line.
  - Round-trip parse → write → parse preserves the ID.

- `internal/board/board_test.go`:
  - `add_card` → returned board's new card has non-empty `ID`.
  - `edit_card` on an ID-less card → after mutation, the card has an ID.
  - Multiple `add_card` → IDs are distinct.

### TS

- `web/shared/src/util/cardid.test.ts`:
  - `newCardId()` length, alphabet, no duplicates in 10k draws.

- `web/shared/src/boardOps.test.ts`:
  - `applyOp({type:'add_card',...})` → returned card has `id`.
  - `applyOp({type:'edit_card',...})` on an ID-less card → card has `id`.

- `web/shared/src/adapters/local.test.ts` (extension or new):
  - After `mutateBoard` add_card, `getBoard` returns a card with non-empty `id`.

## Risks

- **One-time diff churn on the first mutation per card**: each existing card grows an `id:` line on its first edit. Acceptable; documented.
- **Optimistic vs server ID drift**: discussed above. Server replaces; renderer's cache becomes correct. P6.3b's UI guards against linking against unconfirmed cards.
- **Manual user edits of `id:` lines**: a user could rename or duplicate an ID by hand. We don't enforce uniqueness at parse time. Acceptable for local-first single-user; document.
- **ID-bearing files from other tools**: parser tolerates anything; we don't check format. If another tool writes a 5-char ID, we keep it as-is.
- **`go.mod` / `bun.lock`**: no new deps. crypto/rand and crypto.getRandomValues are stdlib.
- **Test ordering inside metadata block**: writer ordering changes if `id` is now first. The writer test that asserts metadata-line order needs an update.

## Open questions

None blocking. Pre-decided:
- 10-char alphanumeric.
- Lazy migration via mutation-touched assignment.
- Same alphabet + length on Go and TS sides.
- Writer emits `id` first when present; omits when empty.
- bleve doc carries `card_id` for future P6.3b link lookups.

## Dependencies on prior work

- P3: parser/writer infrastructure, op set, mutation engine.
- P5.0: `internal/api/v1` and `Deps.Search` (bleve doc field addition).
- P6.2: `internal/search/index.go` (Doc struct gets a new field).

## Dependencies on later work

- P6.3b: cross-board linking UI uses `card.id` as the link target. Without this milestone, link targets would be unstable across reorder/move.

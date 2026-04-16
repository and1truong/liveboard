# P6.3a — Stable Card IDs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a lazy-migrated, 10-char alphanumeric `id` field to every card, mirrored across Go (`pkg/models.Card`) and TS (`web/shared/src/types.ts Card`), with parser/writer roundtrip, engine + `applyOp` assignment on touch, and bleve doc extension.

**Architecture:** Two small `cardid` utilities (Go + TS) produce IDs from crypto RNG. Parser reads `id: <value>` metadata; writer emits `id` as first metadata line when present. `ensureCardID` is called at every card-touching op in both stacks. Deterministic-override hook lets parity vectors and tests pin IDs. Bleve `doc` gains a `card_id` field populated in `UpdateBoard`.

**Tech Stack:** Go 1.24, crypto/rand, bleve v2, TypeScript (bun test), `crypto.getRandomValues`.

---

## File Structure

**Create:**
- `internal/util/cardid/cardid.go` — Go generator with overridable `NewID` var
- `internal/util/cardid/cardid_test.go`
- `web/shared/src/util/cardid.ts` — TS generator with overridable `_setGenerator` helper
- `web/shared/src/util/cardid.test.ts`

**Modify:**
- `pkg/models/models.go` — add `Card.ID`
- `internal/parser/parser.go` — parse `id:`
- `internal/writer/writer.go` — emit `id:` first when non-empty
- `internal/parser/parser_test.go` — add cases
- `internal/writer/writer_test.go` — add cases, adjust any metadata-order asserts
- `internal/board/board.go` — add `ensureCardID`, call from card-touching Apply* fns
- `internal/board/board_test.go` — add cases
- `internal/search/index.go` — add `CardID` to `doc`; populate from `c.ID`
- `internal/search/index_test.go` — verify new field indexed
- `web/shared/src/types.ts` — add `Card.id?`
- `web/shared/src/boardOps.ts` — import and call `ensureCardId` in card-touching branches
- `web/shared/src/boardOps.test.ts` — extend `stripNulls` to ignore `id` on comparison (parity vectors don't carry ids)
- `web/shared/src/adapters/local.test.ts` — add ID round-trip test

---

## Testing strategy for determinism

Random IDs would break shared parity vectors and any assertion that compares whole boards. Solution: the generator is a package-level variable that tests can swap for a deterministic counter. Use this in all engine/applyOp tests that check structural equality.

- Go: `var NewID = defaultNewID` in `cardid` package; tests call `cardid.NewID = func() string { ... }` and restore via `t.Cleanup`.
- TS: export `let generator = defaultGenerator`; export `_setGenerator(fn)` and `_resetGenerator()` for tests.
- `boardOps.test.ts` parity vectors: extend `stripNulls` to drop the `id` key so static vectors without `id` still match outputs where we minted one.

---

## Task 1: Go ID generator

**Files:**
- Create: `internal/util/cardid/cardid.go`
- Test: `internal/util/cardid/cardid_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/util/cardid/cardid_test.go
package cardid

import (
	"strings"
	"testing"
)

func TestNewIDLength(t *testing.T) {
	id := NewID()
	if len(id) != 10 {
		t.Fatalf("want len 10, got %d (%q)", len(id), id)
	}
}

func TestNewIDAlphabet(t *testing.T) {
	id := NewID()
	for _, r := range id {
		if !strings.ContainsRune(Alphabet, r) {
			t.Fatalf("rune %q not in alphabet", r)
		}
	}
}

func TestNewIDUnique(t *testing.T) {
	seen := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		id := NewID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %q after %d draws", id, i)
		}
		seen[id] = struct{}{}
	}
}

func TestNewIDOverride(t *testing.T) {
	orig := NewID
	t.Cleanup(func() { NewID = orig })
	NewID = func() string { return "FIXED00001" }
	if got := NewID(); got != "FIXED00001" {
		t.Fatalf("override failed: %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (package doesn't exist)**

Run: `go test ./internal/util/cardid/...`
Expected: FAIL — package not found / undefined `NewID`.

- [ ] **Step 3: Write the implementation**

```go
// internal/util/cardid/cardid.go
// Package cardid mints stable 10-char alphanumeric identifiers for cards.
package cardid

import (
	"crypto/rand"
	"encoding/binary"
)

// Alphabet is the character set IDs are drawn from (62 chars).
const Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// NewID returns a fresh card ID. Exposed as a variable so tests can inject
// a deterministic generator.
var NewID = defaultNewID

func defaultNewID() string {
	var raw [40]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic("cardid: crypto/rand failed: " + err.Error())
	}
	var b [10]byte
	for i := 0; i < 10; i++ {
		n := binary.BigEndian.Uint32(raw[i*4 : i*4+4])
		b[i] = Alphabet[int(n)%len(Alphabet)]
	}
	return string(b[:])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/util/cardid/...`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/util/cardid/
git commit -m "feat(cardid): add Go 10-char alphanumeric ID generator"
```

---

## Task 2: Go `Card.ID` field

**Files:**
- Modify: `pkg/models/models.go:42-53`

- [ ] **Step 1: Add field**

Edit `pkg/models/models.go`, add as first field of `Card`:

```go
type Card struct {
	ID         string            `json:"id,omitempty"`
	Title      string            `json:"title"`
	Completed  bool              `json:"completed"`
	NoCheckbox bool              `json:"no_checkbox,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	InlineTags []string          `json:"inline_tags,omitempty"`
	Assignee   string            `json:"assignee,omitempty"`
	Priority   string            `json:"priority,omitempty"`
	Due        string            `json:"due,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Body       string            `json:"body,omitempty"`
}
```

- [ ] **Step 2: Verify compile**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add pkg/models/models.go
git commit -m "feat(models): add Card.ID field"
```

---

## Task 3: Parser reads `id:`

**Files:**
- Modify: `internal/parser/parser.go` (metadata switch ~line 164)
- Modify: `internal/parser/parser_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/parser/parser_test.go`:

```go
func TestParseCardID(t *testing.T) {
	md := "---\nversion: 1\nname: B\n---\n\n## Todo\n\n- [ ] Card\n  id: aBc1234XyZ\n"
	b, err := Parse(md)
	if err != nil { t.Fatal(err) }
	if len(b.Columns) != 1 || len(b.Columns[0].Cards) != 1 {
		t.Fatalf("unexpected structure: %+v", b)
	}
	if got := b.Columns[0].Cards[0].ID; got != "aBc1234XyZ" {
		t.Fatalf("want id %q, got %q", "aBc1234XyZ", got)
	}
}

func TestParseCardIDAbsent(t *testing.T) {
	md := "---\nversion: 1\nname: B\n---\n\n## Todo\n\n- [ ] Card\n"
	b, err := Parse(md)
	if err != nil { t.Fatal(err) }
	if got := b.Columns[0].Cards[0].ID; got != "" {
		t.Fatalf("want empty id, got %q", got)
	}
}
```

- [ ] **Step 2: Run — fails (ID always empty because parser drops to Metadata map)**

Run: `go test ./internal/parser/ -run TestParseCardID`
Expected: FAIL on `TestParseCardID` (id lands in Metadata, not `card.ID`).

- [ ] **Step 3: Add case to metadata switch**

In `internal/parser/parser.go`, inside the `switch key {` block (around line 164), add **before** `default:`:

```go
case "id":
	currentCard.ID = val
```

- [ ] **Step 4: Run — passes**

Run: `go test ./internal/parser/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/
git commit -m "feat(parser): read card id metadata field"
```

---

## Task 4: Writer emits `id:` first

**Files:**
- Modify: `internal/writer/writer.go` (inside `writeCard`, after title line)
- Modify: `internal/writer/writer_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/writer/writer_test.go`:

```go
func TestRenderCardIDFirst(t *testing.T) {
	b := &models.Board{
		Version: 1, Name: "B",
		Columns: []models.Column{{Name: "Todo", Cards: []models.Card{
			{ID: "XYZ1234567", Title: "T", Assignee: "alice", Priority: "high"},
		}}},
	}
	out, err := Render(b)
	if err != nil { t.Fatal(err) }
	// id must appear immediately after the title line, before any other metadata.
	want := "- [ ] T\n  id: XYZ1234567\n  assignee: alice\n  priority: high\n"
	if !strings.Contains(out, want) {
		t.Fatalf("expected %q in output:\n%s", want, out)
	}
}

func TestRenderCardIDOmittedWhenEmpty(t *testing.T) {
	b := &models.Board{
		Version: 1, Name: "B",
		Columns: []models.Column{{Name: "Todo", Cards: []models.Card{{Title: "T"}}}},
	}
	out, err := Render(b)
	if err != nil { t.Fatal(err) }
	if strings.Contains(out, "  id:") {
		t.Fatalf("did not expect id line:\n%s", out)
	}
}

func TestRenderRoundTripID(t *testing.T) {
	src := "---\nversion: 1\nname: B\n---\n\n## Todo\n\n- [ ] T\n  id: ABCDEFGHIJ\n"
	parsed, err := parser.Parse(src)
	if err != nil { t.Fatal(err) }
	out, err := Render(parsed)
	if err != nil { t.Fatal(err) }
	parsed2, err := parser.Parse(out)
	if err != nil { t.Fatal(err) }
	if parsed2.Columns[0].Cards[0].ID != "ABCDEFGHIJ" {
		t.Fatalf("round-trip lost id: %+v", parsed2.Columns[0].Cards[0])
	}
}
```

Ensure the file imports `strings` and the parser package (`github.com/and1truong/liveboard/internal/parser`). Add imports if missing.

- [ ] **Step 2: Run — fails (writer ignores ID)**

Run: `go test ./internal/writer/...`
Expected: FAIL.

- [ ] **Step 3: Emit `id` first**

In `internal/writer/writer.go` `writeCard`, directly after the title-line block (after the `- [x] ...` / `- ...` print, before `metaTags`), insert:

```go
	if card.ID != "" {
		fmt.Fprintf(b, "  id: %s\n", card.ID)
	}
```

- [ ] **Step 4: Run — passes**

Run: `go test ./internal/writer/...`
Expected: PASS.

- [ ] **Step 5: Fix any other writer tests that assert exact metadata-line ordering**

Run: `go test ./internal/writer/... ./internal/parser/... ./internal/board/...`
If anything else fails because of ordering assumptions, update those tests to accept `id:` first. If none fail, skip.

- [ ] **Step 6: Commit**

```bash
git add internal/writer/
git commit -m "feat(writer): emit card id as first metadata line"
```

---

## Task 5: Engine `ensureCardID` + call sites

**Files:**
- Modify: `internal/board/board.go`
- Modify: `internal/board/board_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/board/board_test.go`:

```go
func TestApplyAddCardAssignsID(t *testing.T) {
	b := &models.Board{Columns: []models.Column{{Name: "Todo"}}}
	c, err := ApplyAddCard(b, "Todo", "hello", false)
	if err != nil { t.Fatal(err) }
	if c.ID == "" { t.Fatal("expected ID to be assigned") }
	if len(c.ID) != 10 { t.Fatalf("unexpected ID length: %q", c.ID) }
}

func TestApplyEditCardAssignsIDWhenMissing(t *testing.T) {
	b := &models.Board{Columns: []models.Column{{Name: "Todo", Cards: []models.Card{{Title: "x"}}}}}
	if err := ApplyEditCard(b, 0, 0, "x", "", nil, "", "", ""); err != nil { t.Fatal(err) }
	if b.Columns[0].Cards[0].ID == "" { t.Fatal("expected ID after edit") }
}

func TestApplyAddCardDistinctIDs(t *testing.T) {
	b := &models.Board{Columns: []models.Column{{Name: "Todo"}}}
	c1, _ := ApplyAddCard(b, "Todo", "a", false)
	c2, _ := ApplyAddCard(b, "Todo", "b", false)
	if c1.ID == c2.ID { t.Fatalf("expected distinct IDs, got %q twice", c1.ID) }
}

func TestApplyCompleteTagMovePreserveAndAssignID(t *testing.T) {
	b := &models.Board{Columns: []models.Column{
		{Name: "Todo", Cards: []models.Card{{Title: "x"}}},
		{Name: "Done"},
	}}
	if err := ApplyCompleteCard(b, 0, 0); err != nil { t.Fatal(err) }
	id1 := b.Columns[0].Cards[0].ID
	if id1 == "" { t.Fatal("complete_card must assign ID") }
	if err := ApplyTagCard(b, 0, 0, []string{"a"}); err != nil { t.Fatal(err) }
	if b.Columns[0].Cards[0].ID != id1 { t.Fatal("tag_card must preserve ID") }
	if err := ApplyMoveCard(b, 0, 0, "Done"); err != nil { t.Fatal(err) }
	if b.Columns[1].Cards[0].ID != id1 { t.Fatal("move_card must preserve ID") }
}
```

- [ ] **Step 2: Run — fails**

Run: `go test ./internal/board/ -run "TestApplyAddCardAssignsID|TestApplyEditCardAssignsIDWhenMissing|TestApplyAddCardDistinctIDs|TestApplyCompleteTagMovePreserveAndAssignID"`
Expected: FAIL.

- [ ] **Step 3: Add helper + wire into every card-touching Apply fn**

In `internal/board/board.go`, add to the imports:

```go
	"github.com/and1truong/liveboard/internal/util/cardid"
```

Add near the top of the file (below Engine definition):

```go
// ensureCardID assigns a fresh ID to c if it has none.
func ensureCardID(c *models.Card) {
	if c == nil { return }
	if c.ID == "" {
		c.ID = cardid.NewID()
	}
}
```

Wire the helper into each card-touching Apply fn. For each edit, the call must target the card as stored in the slice (use `&b.Columns[...].Cards[...]`), not a value copy.

- `ApplyAddCard`: after both append branches, call `ensureCardID` on the returned pointer (before `return`). Example for the prepend branch:

  ```go
  if prepend {
      b.Columns[i].Cards = append([]models.Card{{Title: title}}, b.Columns[i].Cards...)
      ensureCardID(&b.Columns[i].Cards[0])
      return &b.Columns[i].Cards[0], nil
  }
  b.Columns[i].Cards = append(b.Columns[i].Cards, models.Card{Title: title})
  ensureCardID(&b.Columns[i].Cards[len(b.Columns[i].Cards)-1])
  return &b.Columns[i].Cards[len(b.Columns[i].Cards)-1], nil
  ```

- `ApplyEditCard`: after `card := &b.Columns[colIdx].Cards[cardIdx]`, add `ensureCardID(card)`.
- `ApplyCompleteCard`: after `validateIndices`, add `ensureCardID(&b.Columns[colIdx].Cards[cardIdx])` (before toggling).
- `ApplyTagCard`: after `card := &b.Columns[colIdx].Cards[cardIdx]`, add `ensureCardID(card)`.
- `ApplyMoveCard`: after `validateIndices`, add `ensureCardID(&b.Columns[colIdx].Cards[cardIdx])` **before** reading `card := b.Columns[colIdx].Cards[cardIdx]` so the copy carries the ID.
- `ApplyReorderCard`: same pattern — `ensureCardID(&b.Columns[colIdx].Cards[cardIdx])` before the value read.
- `ApplyDeleteCard`: no call (skip per spec).
- Column-level and board-level `Apply*` fns: no call.

- [ ] **Step 4: Run — all pass**

Run: `go test ./internal/board/...`
Expected: PASS.

- [ ] **Step 5: Regression sweep**

Run: `go test ./...`
Expected: PASS across the repo. If any board-snapshot or string-equality test breaks because minted IDs changed the output, update that test to pin `cardid.NewID` to a deterministic stub using `t.Cleanup`:

```go
orig := cardid.NewID
cardid.NewID = func() string { return "TESTID0001" }
t.Cleanup(func() { cardid.NewID = orig })
```

- [ ] **Step 6: Commit**

```bash
git add internal/board/
git commit -m "feat(board): assign card IDs on every mutation that touches a card"
```

---

## Task 6: bleve `doc.CardID`

**Files:**
- Modify: `internal/search/index.go`
- Modify: `internal/search/index_test.go`

- [ ] **Step 1: Write failing test**

Append to `internal/search/index_test.go`:

```go
func TestUpdateBoardIndexesCardID(t *testing.T) {
	idx, err := New()
	if err != nil { t.Fatal(err) }
	b := &models.Board{Name: "B", Columns: []models.Column{{Name: "Todo", Cards: []models.Card{
		{ID: "ABCDE12345", Title: "hello world"},
	}}}}
	if err := idx.UpdateBoard("b", b); err != nil { t.Fatal(err) }
	// Search by unique title term; returned doc must round-trip card_id.
	q := bleve.NewTermQuery("hello")
	q.SetField("title")
	sr := bleve.NewSearchRequestOptions(q, 10, 0, false)
	sr.Fields = []string{"card_id"}
	res, err := idx.idx.Search(sr)
	if err != nil { t.Fatal(err) }
	if len(res.Hits) != 1 { t.Fatalf("want 1 hit, got %d", len(res.Hits)) }
	if got, _ := res.Hits[0].Fields["card_id"].(string); got != "ABCDE12345" {
		t.Fatalf("want card_id ABCDE12345, got %v", res.Hits[0].Fields["card_id"])
	}
}
```

Add `"github.com/blevesearch/bleve/v2"` and `"github.com/and1truong/liveboard/pkg/models"` imports if the file doesn't already have them.

- [ ] **Step 2: Run — fails (field absent)**

Run: `go test ./internal/search/ -run TestUpdateBoardIndexesCardID`
Expected: FAIL.

- [ ] **Step 3: Extend `doc` and populate**

In `internal/search/index.go`:

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

Inside `UpdateBoard`, extend `d`:

```go
d := doc{
	BoardID:   slug,
	BoardName: boardName,
	ColIdx:    cIdx,
	CardIdx:   kIdx,
	CardID:    c.ID,
	Title:     c.Title,
	Body:      c.Body,
	Tags:      c.Tags,
}
```

- [ ] **Step 4: Run — passes**

Run: `go test ./internal/search/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/search/
git commit -m "feat(search): index card_id field on bleve doc"
```

---

## Task 7: TS ID generator

**Files:**
- Create: `web/shared/src/util/cardid.ts`
- Test: `web/shared/src/util/cardid.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
// web/shared/src/util/cardid.test.ts
import { describe, expect, it, afterEach } from 'bun:test'
import { ALPHABET, newCardId, _setGenerator, _resetGenerator } from './cardid.js'

describe('newCardId', () => {
  afterEach(() => _resetGenerator())

  it('returns 10 chars from the alphabet', () => {
    const id = newCardId()
    expect(id).toHaveLength(10)
    for (const ch of id) expect(ALPHABET).toContain(ch)
  })

  it('produces no duplicates in 10k draws', () => {
    const seen = new Set<string>()
    for (let i = 0; i < 10000; i++) {
      const id = newCardId()
      expect(seen.has(id)).toBe(false)
      seen.add(id)
    }
  })

  it('can be overridden for tests', () => {
    _setGenerator(() => 'FIXED00001')
    expect(newCardId()).toBe('FIXED00001')
    _resetGenerator()
    expect(newCardId()).not.toBe('FIXED00001')
  })
})
```

- [ ] **Step 2: Run — fails (module missing)**

Run: `cd web/shared && bun test src/util/cardid.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement generator**

```ts
// web/shared/src/util/cardid.ts
export const ALPHABET =
  'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'

type Generator = () => string

function defaultGenerator(): string {
  const bytes = new Uint32Array(10)
  crypto.getRandomValues(bytes)
  let out = ''
  for (let i = 0; i < 10; i++) out += ALPHABET[bytes[i]! % ALPHABET.length]
  return out
}

let generator: Generator = defaultGenerator

export function newCardId(): string {
  return generator()
}

export function _setGenerator(fn: Generator): void {
  generator = fn
}

export function _resetGenerator(): void {
  generator = defaultGenerator
}
```

- [ ] **Step 4: Run — passes**

Run: `cd web/shared && bun test src/util/cardid.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/util/cardid.ts web/shared/src/util/cardid.test.ts
git commit -m "feat(cardid): add TS 10-char alphanumeric ID generator"
```

---

## Task 8: TS `Card.id` field

**Files:**
- Modify: `web/shared/src/types.ts:26-37`

- [ ] **Step 1: Add field**

Edit the `Card` interface to add `id?: string` as the first field:

```ts
export interface Card {
  id?: string
  title: string
  completed?: boolean
  no_checkbox?: boolean
  tags?: string[]
  inline_tags?: string[]
  assignee?: string
  priority?: string
  due?: string
  metadata?: Record<string, string>
  body?: string
}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd web/shared && bun run typecheck` (or `bunx tsc --noEmit`)
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/types.ts
git commit -m "feat(types): add Card.id field"
```

---

## Task 9: `applyOp` assigns IDs on card-touching ops

**Files:**
- Modify: `web/shared/src/boardOps.ts`
- Modify: `web/shared/src/boardOps.test.ts` (tweak `stripNulls` and add targeted tests)

- [ ] **Step 1: Write failing tests**

Append to `web/shared/src/boardOps.test.ts` (at the bottom, after existing vector-driven tests):

```ts
import { _setGenerator, _resetGenerator } from './util/cardid.js'

describe('applyOp card id assignment', () => {
  afterEach(() => _resetGenerator())

  it('add_card assigns id', () => {
    _setGenerator(() => 'OPTID00001')
    const b: Board = { columns: [{ name: 'Todo', cards: [] }] }
    const out = applyOp(b, { type: 'add_card', column: 'Todo', title: 'x' })
    expect(out.columns![0]!.cards[0]!.id).toBe('OPTID00001')
  })

  it('edit_card assigns id to id-less card', () => {
    _setGenerator(() => 'OPTID00002')
    const b: Board = { columns: [{ name: 'Todo', cards: [{ title: 'x' }] }] }
    const out = applyOp(b, {
      type: 'edit_card', col_idx: 0, card_idx: 0,
      title: 'x', body: '', tags: [], priority: '', due: '', assignee: '',
    })
    expect(out.columns![0]!.cards[0]!.id).toBe('OPTID00002')
  })

  it('preserves existing id on edit_card', () => {
    _setGenerator(() => 'SHOULD_NOT_USE')
    const b: Board = { columns: [{ name: 'Todo', cards: [{ id: 'KEEPME1234', title: 'x' }] }] }
    const out = applyOp(b, {
      type: 'edit_card', col_idx: 0, card_idx: 0,
      title: 'x', body: '', tags: [], priority: '', due: '', assignee: '',
    })
    expect(out.columns![0]!.cards[0]!.id).toBe('KEEPME1234')
  })
})
```

You also need to keep vector-driven equality tests green. Find the `stripNulls` function in the same test file and extend it to drop the `id` field from object entries so that vectors (which don't carry `id`) still match outputs that include one. In the `Object.entries(...)` loop, add:

```ts
if (k === 'id') continue
```

(place this next to the existing null filter).

You'll need `afterEach` in the imports from `bun:test`.

- [ ] **Step 2: Run — fails**

Run: `cd web/shared && bun test src/boardOps.test.ts`
Expected: FAIL on new tests.

- [ ] **Step 3: Wire `ensureCardId` into card-touching branches**

In `web/shared/src/boardOps.ts`, at the top add:

```ts
import { newCardId } from './util/cardid.js'

function ensureCardId(c: Card): void {
  if (!c.id) c.id = newCardId()
}
```

Call `ensureCardId` in each branch that creates or edits a card. The `structuredClone` at the top of `applyOp` means mutating `card` is safe.

- `add_card`: after `const card: Card = { title: op.title }`, call `ensureCardId(card)`.
- `move_card`: after `const card = src.cards[op.card_idx]!`, call `ensureCardId(card)`.
- `reorder_card`: same as `move_card`.
- `edit_card`: after `const card = col.cards[op.card_idx]!`, call `ensureCardId(card)` (before title assignment).
- `complete_card`: after `const card = col.cards[op.card_idx]!`, call `ensureCardId(card)`.
- `tag_card`: after `const card = col.cards[op.card_idx]!`, call `ensureCardId(card)`.
- `delete_card`: skip.
- All column-level and board-level branches: skip.

- [ ] **Step 4: Run — passes**

Run: `cd web/shared && bun test src/boardOps.test.ts`
Expected: PASS (including untouched parity vectors).

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/boardOps.ts web/shared/src/boardOps.test.ts
git commit -m "feat(boardOps): assign card id on every card-touching op"
```

---

## Task 10: Local adapter round-trip test

**Files:**
- Modify: `web/shared/src/adapters/local.test.ts`

- [ ] **Step 1: Write failing test**

Append to `web/shared/src/adapters/local.test.ts` (pattern-match existing imports and factory — look at the top of the file to reuse whatever `createLocalAdapter` / test harness is in place):

```ts
it('mutateBoard add_card → getBoard returns card with non-empty id', async () => {
  // Use whatever adapter factory the rest of this file uses; create a board
  // with one empty column named "Todo", then add a card and reload.
  const adapter = /* existing factory */
  const slug = /* existing slug-create pattern */
  await adapter.mutateBoard(slug, -1, { type: 'add_card', column: 'Todo', title: 'x' })
  const board = await adapter.getBoard(slug)
  const card = board.columns![0]!.cards[0]!
  expect(card.id).toBeDefined()
  expect(card.id).toHaveLength(10)
})
```

If the existing file already has an adapter fixture/helper, reuse it verbatim. If not, open `local.test.ts` and mimic the patterns in the nearest existing `it(...)` test — copy imports and setup exactly.

- [ ] **Step 2: Run — confirm it passes (end-to-end wiring from Task 9 should already satisfy it)**

Run: `cd web/shared && bun test src/adapters/local.test.ts`
Expected: PASS.

If it fails with an id missing, the local adapter's mutation path might snapshot before `applyOp`; inspect `web/shared/src/adapters/local.ts` and ensure the persisted board comes from `applyOp`'s return value, not the input. Fix only if broken.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/adapters/local.test.ts
git commit -m "test(local-adapter): verify card id persists through mutateBoard+getBoard"
```

---

## Task 11: Full regression + lint

- [ ] **Step 1: Run full Go test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 2: Run full TS test suite**

Run: `cd web/shared && bun test`
Expected: PASS.

- [ ] **Step 3: Run linters per repo convention**

Run: `make lint`
Expected: no errors.

- [ ] **Step 4: Inspect a freshly-mutated demo board and confirm `id:` lines appear**

Run: `make dev` briefly, edit a card via the UI, then inspect the `.md` file. Verify the `id:` line is the first metadata line for the touched card, untouched cards carry no `id:`.

If `make dev` is disruptive, skip and rely on tests.

- [ ] **Step 5: Commit any lint-only fixes**

```bash
git add -u
git commit -m "chore: lint fixes for stable card id rollout" || echo "nothing to commit"
```

---

## Self-Review Notes

- **Spec coverage:** Card.id field (T2, T8), generators (T1, T7), parser (T3), writer (T4), engine (T5), applyOp (T9), bleve doc (T6), tests across parser/writer/engine/applyOp/local adapter — all covered.
- **Placeholder scan:** Task 10 references "existing factory / slug-create pattern" — this is because `adapters/local.test.ts` wasn't read in detail; the engineer is instructed to mirror a neighboring `it()` rather than being told to TBD.
- **Type consistency:** Go field is `Card.ID string`, TS field is `Card.id?: string`. Generator names: Go `cardid.NewID` (var), TS `newCardId()` + `_setGenerator`/`_resetGenerator` for tests. bleve json key `card_id`. All referenced consistently across tasks.

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-15-p6-3a-stable-card-ids.md`. Two execution options:**

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?

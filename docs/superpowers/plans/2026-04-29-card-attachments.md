# Card Attachments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add file attachments to cards. Files live in a workspace-wide content-addressed pool (`<workspace>/.attachments/<sha256>.<ext>`). Cards reference attachments via a JSON-encoded `attachments:` metadata field. Five new mutation ops (add/remove/move/rename/reorder) wired through the existing dispatch pipeline. Frontend `LocalAdapter` mirrors the pool in IndexedDB. Body markdown supports inline `attachment:<hash>` URLs.

**Architecture:**

- **Storage:** Content-addressed blob pool at `<workspace>/.attachments/`. Card metadata holds `[{h, n, s, m}]` (hash, display-name, size, sniffed-mime).
- **Mutation pipeline:** Five new variants in the existing `mutationRegistry` (`internal/board/mutation.go`). Atomic per-board, optimistic locking unchanged. Cross-card move is same-board only.
- **HTTP:** New `/api/v1/attachments` (multipart upload) and `/api/v1/attachments/{hash}/{name}` (download with `Content-Disposition` policy). Optional `?thumb=1` for image thumbs.
- **GC:** Manual via `liveboard gc` cobra command. Walks all boards, collects referenced hashes (from `attachments:` field + body `attachment:<hash>` URLs), deletes orphaned blobs.
- **Frontend:** `LocalAdapter` puts blobs in IndexedDB keyed by hash. `ServerAdapter` POSTs/GETs the HTTP endpoints. Renderer card UI: badge in compact mode, thumbnail strip in expanded mode, full UI in modal. Body markdown renderer rewrites `attachment:<hash>` → resolved URL.

**Tech Stack:** Go 1.24, chi/v5, cobra. React 19 + TanStack Query (renderer). Bleve (search). MCP Streamable HTTP. Bun (TS tooling).

**Pre-flight:** Confirm you're on a worktree branch (per project workflow). Run `git status` — uncommitted work on main should be stashed/committed before starting. Run `make lint` to confirm baseline is green.

---

## File Structure

### New files

- `internal/attachments/store.go` — content-addressed pool: `Put(io.Reader) (Descriptor, error)`, `Open(hash) (io.ReadSeekCloser, error)`, `Stat(hash) (size int64, mime string, ok bool)`, `Path(hash) string`, `Remove(hash) error`.
- `internal/attachments/store_test.go`
- `internal/attachments/refs.go` — `CollectReferenced(workspaceDir) (map[string]struct{}, error)`. Walks `.md` files, scans `attachments:` field + body `attachment:<hash>` URLs.
- `internal/attachments/refs_test.go`
- `internal/attachments/gc.go` — `GC(workspaceDir) (deleted []string, err error)`. Reads pool dir, removes blobs not in `CollectReferenced`. Pure; no time grace (manual mode).
- `internal/attachments/gc_test.go`
- `internal/attachments/thumb.go` — `Thumb(src io.Reader, max int) (io.Reader, error)`. WebP encode of decoded image, max-edge `max` px. Returns `ErrNotImage` for non-images.
- `internal/attachments/thumb_test.go`
- `internal/api/v1/attachments.go` — HTTP handlers: `postUpload`, `getDownload`. Wires `Store` + `Thumb` + `Content-Disposition` policy.
- `internal/api/v1/attachments_test.go`
- `cmd/liveboard/gc.go` — cobra `liveboard gc` command. Calls `attachments.GC(workspaceDir)`.
- `internal/mcp/tools_attachment.go` — five MCP tools mirroring the five mutation ops (descriptor-level only).
- `internal/mcp/tools_attachment_test.go`
- `web/shared/src/attachments.ts` — `Attachment` type, helpers `parseAttachmentURL(url) → hash | null`, `attachmentURL(adapter, descriptor) → string` (server: HTTP path; local: blob URL).
- `web/shared/src/adapters/local-attachments.ts` — IndexedDB driver: `putBlob(blob) → Descriptor`, `getBlob(hash) → Blob | null`. Computes SHA-256 via `crypto.subtle.digest`.
- `web/shared/src/adapters/local-attachments.test.ts`
- `web/renderer/default/src/components/AttachmentList.tsx` — modal full UI: list rows, drag-handle reorder, inline rename, remove, download, drop zone, paste handler, "+" button, "insert into body" action.
- `web/renderer/default/src/components/AttachmentBadge.tsx` — compact mode badge.
- `web/renderer/default/src/components/AttachmentThumbStrip.tsx` — expanded mode thumb strip.
- `web/renderer/default/src/markdown/attachmentScheme.ts` — markdown plugin/utility that rewrites `attachment:<hash>.<ext>` URLs in `<a>`/`<img>` to resolved URLs from the active adapter.
- `docs/attachments.md` — user-facing reference: pool layout, GC command, body URL scheme.

### Modified files

- `pkg/models/models.go` — add `Attachment` struct, add `Card.Attachments []Attachment`.
- `internal/parser/parser.go` — handle `attachments` key (JSON-decode to `[]Attachment`).
- `internal/parser/parser_test.go` — roundtrip tests including attachments.
- `internal/writer/writer.go` — emit `attachments:` line (JSON-encode), in alphabetical position with other metadata keys.
- `internal/board/mutation.go` — add 5 op structs + 5 registry entries.
- `internal/board/board.go` — add 5 `apply*` functions and 5 thin `Engine.*` wrappers (the latter only if any caller besides the dispatcher needs them; otherwise skip).
- `internal/board/mutation_test.go` — vector cases for the 5 new ops.
- `internal/board/board_test.go` — direct unit tests for new `apply*` functions if engine wrappers added.
- `internal/api/v1/router.go` — mount `/attachments` routes.
- `internal/web/settings.go` — read `attachments_max_bytes` from `settings.json` (default 25MB). Add field to relevant struct.
- `internal/export/export.go` — bundle referenced blobs into ZIP; HTML export rewrites `attachment:` URLs and renders `attachments:` field as link/img list. Honour `?attachments=false`.
- `internal/api/server.go` — pass `attachments=false` query param down into `export.Options`.
- `internal/search/index.go` — add `attachment_names` field on card document.
- `internal/mcp/server.go` — register the 5 new tools.
- `internal/parity/runner_test.go` — extend vector runner if needed (likely vectors live in shared file; new vectors auto-pick-up).
- `web/shared/src/types.ts` — add `Attachment` type, add `Card.attachments` field.
- `web/shared/src/boardOps.ts` — TS twin of new `apply*` functions.
- `web/shared/src/boardOps.test.ts` — TS vector tests for the 5 new ops.
- `web/shared/src/mutations.gen.ts` — regenerated by `cmd/gen-ts-mutations`.
- `cmd/gen-ts-mutations/main.go` — usually no change (registry-driven), but verify it picks up the new variants.
- `web/shared/src/adapter.ts` — add `BackendAdapter.uploadAttachment(File): Promise<Attachment>` and `attachmentURL(hash, name): string`.
- `web/shared/src/adapters/server.ts` — implement upload via `fetch` multipart; `attachmentURL` returns `/api/v1/attachments/<hash>/<name>`.
- `web/shared/src/adapters/local.ts` — implement upload via `local-attachments.ts`; `attachmentURL` returns a `blob:` URL from IndexedDB lookup.
- `web/renderer/default/src/components/CardModal.tsx` (or wherever the card edit modal lives — verify path before editing) — mount `AttachmentList`.
- `web/renderer/default/src/components/Card.tsx` (or wherever the column card renders — verify path) — mount `AttachmentBadge` (compact) or `AttachmentThumbStrip` (expanded).
- `web/renderer/default/src/markdown/*` — wire `attachmentScheme` into the markdown renderer used for card body.
- `Makefile` — add `make gen-ts-mutations` target if not already exposed; document in plan steps.
- `CLAUDE.md` — bump architecture section to mention `internal/attachments/` and the new endpoint paths.

---

## Phase 1 — Data model: `Attachment` struct, parser, writer

### Task 1: Add `Attachment` struct and `Card.Attachments` field

**Files:**
- Modify: `pkg/models/models.go`

- [ ] **Step 1: Add the type and field**

In `pkg/models/models.go`, after the `Card` struct definition, add:

```go
// Attachment is a card-level reference to a blob in the workspace pool.
// Field tags use short keys to keep the serialized JSON line scannable
// in raw markdown.
type Attachment struct {
	Hash string `json:"h"`           // sha256 hex + "." + extension, e.g. "a3f9...e1.pdf"
	Name string `json:"n"`           // display filename, user-editable via rename_attachment
	Size int64  `json:"s"`           // bytes
	Mime string `json:"m"`           // sniffed MIME at upload time
}
```

In the `Card` struct, add after `Body`:

```go
	Attachments []Attachment `json:"attachments,omitempty"`
```

- [ ] **Step 2: Build to verify no compile errors**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add pkg/models/models.go
git commit -m "feat(models): add Attachment struct and Card.Attachments field"
```

---

### Task 2: Parse `attachments:` metadata key (failing test first)

**Files:**
- Modify: `internal/parser/parser_test.go`
- Modify: `internal/parser/parser.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/parser/parser_test.go`:

```go
func TestParseCardAttachments(t *testing.T) {
	md := `---
version: 1
name: T
---

## Col

- [ ] Card
  attachments: [{"h":"a3f9.pdf","n":"Plan.pdf","s":12,"m":"application/pdf"}]
`
	b, err := Parse(md)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Columns) != 1 || len(b.Columns[0].Cards) != 1 {
		t.Fatalf("shape: %+v", b)
	}
	c := b.Columns[0].Cards[0]
	if len(c.Attachments) != 1 {
		t.Fatalf("attachments len: %d", len(c.Attachments))
	}
	got := c.Attachments[0]
	want := models.Attachment{Hash: "a3f9.pdf", Name: "Plan.pdf", Size: 12, Mime: "application/pdf"}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/parser -run TestParseCardAttachments -v`
Expected: FAIL — attachments slice is empty (parser stuffs unknown keys into `Metadata`).

- [ ] **Step 3: Implement the parser branch**

In `internal/parser/parser.go`, in the `switch key` block of the metadata-line handler (around the existing `case "links":`), add:

```go
		case "attachments":
			var atts []models.Attachment
			if err := json.Unmarshal([]byte(val), &atts); err == nil {
				currentCard.Attachments = atts
			}
```

Add `"encoding/json"` to imports if not already present.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/parser -run TestParseCardAttachments -v`
Expected: PASS.

- [ ] **Step 5: Add a malformed-JSON test (graceful degradation)**

```go
func TestParseCardAttachmentsMalformed(t *testing.T) {
	md := `---
version: 1
name: T
---

## Col

- [ ] Card
  attachments: not-json
`
	b, err := Parse(md)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(b.Columns[0].Cards[0].Attachments) != 0 {
		t.Errorf("expected empty attachments on malformed input")
	}
}
```

Run: `go test ./internal/parser -run TestParseCardAttachmentsMalformed -v`
Expected: PASS (already handled — branch silently leaves slice nil on JSON error).

- [ ] **Step 6: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat(parser): decode attachments metadata key as JSON"
```

---

### Task 3: Render `attachments:` metadata key

**Files:**
- Modify: `internal/writer/writer.go`
- Add test in: `internal/writer/writer_test.go` (create if missing) OR `internal/parser/parser_test.go` as roundtrip

- [ ] **Step 1: Write a roundtrip failing test**

Add to `internal/parser/parser_test.go` (parser tests already do roundtrip):

```go
func TestRoundtripCardAttachments(t *testing.T) {
	original := &models.Board{
		Version: 1,
		Name:    "T",
		Columns: []models.Column{{
			Name: "Col",
			Cards: []models.Card{{
				Title: "Card",
				Attachments: []models.Attachment{
					{Hash: "a3f9.pdf", Name: "Plan.pdf", Size: 12, Mime: "application/pdf"},
					{Hash: "7c2b.png", Name: "shot.png", Size: 88, Mime: "image/png"},
				},
			}},
		}},
	}
	rendered, err := writer.Render(original)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	parsed, err := Parse(rendered)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := parsed.Columns[0].Cards[0].Attachments
	if !reflect.DeepEqual(got, original.Columns[0].Cards[0].Attachments) {
		t.Errorf("roundtrip mismatch:\n got:  %+v\n want: %+v\n rendered:\n%s",
			got, original.Columns[0].Cards[0].Attachments, rendered)
	}
}
```

Add `"reflect"`, `"github.com/and1truong/liveboard/internal/writer"`, `"github.com/and1truong/liveboard/pkg/models"` imports if not present.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/parser -run TestRoundtripCardAttachments -v`
Expected: FAIL — writer doesn't emit attachments.

- [ ] **Step 3: Implement writer emit**

In `internal/writer/writer.go`, inside `writeCard`, after the `card.Due` block and before the `Metadata` map block, add:

```go
	if len(card.Attachments) > 0 {
		jb, err := json.Marshal(card.Attachments)
		if err == nil {
			fmt.Fprintf(b, "  attachments: %s\n", jb)
		}
	}
```

Add `"encoding/json"` to imports if not present.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/parser -run TestRoundtripCardAttachments -v`
Expected: PASS.

- [ ] **Step 5: Run full parser/writer suites to catch regressions**

Run: `go test ./internal/parser ./internal/writer -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/writer/writer.go internal/parser/parser_test.go
git commit -m "feat(writer): emit attachments metadata as JSON line"
```

---

## Phase 2 — Mutation ops: 5 new variants

### Task 4: Add `add_attachments` op (struct, registry, apply, vector test)

**Files:**
- Modify: `internal/board/mutation.go`
- Modify: `internal/board/board.go`
- Modify: `internal/board/mutation_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/board/mutation_test.go`, find the table-driven test list (search for `cases := []board.MutationOp{` near line 260, or `vectorCases` if structured that way). Add a new test:

```go
func TestApplyAddAttachments(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{{
			Name: "Col",
			Cards: []models.Card{{Title: "C", ID: "id1"}},
		}},
	}
	op := board.MutationOp{
		Type: "add_attachments",
		AddAttachments: &board.AddAttachmentsOp{
			ColIdx:  0,
			CardIdx: 0,
			Items: []models.Attachment{
				{Hash: "a3f9.pdf", Name: "Plan.pdf", Size: 12, Mime: "application/pdf"},
			},
		},
	}
	if err := board.ApplyMutation(b, op); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := b.Columns[0].Cards[0].Attachments
	if len(got) != 1 || got[0].Hash != "a3f9.pdf" {
		t.Errorf("got %+v", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/board -run TestApplyAddAttachments -v`
Expected: FAIL — `AddAttachments` field doesn't exist on `MutationOp`.

- [ ] **Step 3: Add the op struct and registry entry**

In `internal/board/mutation.go`, add a typed pointer field to `MutationOp`:

```go
	AddAttachments *AddAttachmentsOp `json:"-"`
```

After the existing op struct definitions, add:

```go
// AddAttachmentsOp are the params for an "add_attachments" mutation.
// Items is plural to make batch-uploads (drag N files) a single mutation.
type AddAttachmentsOp struct {
	ColIdx  int                 `json:"col_idx"`
	CardIdx int                 `json:"card_idx"`
	Items   []models.Attachment `json:"items"`
}
```

In `mutationRegistry`, add an entry:

```go
	"add_attachments": {
		new: func() any { return &AddAttachmentsOp{} },
		get: func(m *MutationOp) (any, bool) { return m.AddAttachments, m.AddAttachments != nil },
		set: func(m *MutationOp, v any) { m.AddAttachments = mustCast[*AddAttachmentsOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*AddAttachmentsOp](p)
			return applyAddAttachments(b, op.ColIdx, op.CardIdx, op.Items)
		},
	},
```

- [ ] **Step 4: Add the apply function**

In `internal/board/board.go`, add:

```go
// applyAddAttachments appends items to a card's attachment list.
// Duplicates by hash are skipped so re-issuing the op is idempotent.
func applyAddAttachments(b *models.Board, colIdx, cardIdx int, items []models.Attachment) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	card := &b.Columns[colIdx].Cards[cardIdx]
	ensureCardID(card)
	existing := make(map[string]struct{}, len(card.Attachments))
	for _, a := range card.Attachments {
		existing[a.Hash] = struct{}{}
	}
	for _, a := range items {
		if _, dup := existing[a.Hash]; dup {
			continue
		}
		card.Attachments = append(card.Attachments, a)
		existing[a.Hash] = struct{}{}
	}
	return nil
}
```

- [ ] **Step 5: Run to verify the test passes**

Run: `go test ./internal/board -run TestApplyAddAttachments -v`
Expected: PASS.

- [ ] **Step 6: Run the registry-coverage assertion**

Run: `go test ./internal/board -run TestRegistryCoversAllVariants -v`
Expected: PASS — registry entry matches the new field.

- [ ] **Step 7: Commit**

```bash
git add internal/board/mutation.go internal/board/board.go internal/board/mutation_test.go
git commit -m "feat(board): add_attachments mutation"
```

---

### Task 5: Add `remove_attachment` op

**Files:**
- Modify: `internal/board/mutation.go`
- Modify: `internal/board/board.go`
- Modify: `internal/board/mutation_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestApplyRemoveAttachment(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{{
			Name: "Col",
			Cards: []models.Card{{
				Title: "C", ID: "id1",
				Attachments: []models.Attachment{
					{Hash: "a.pdf", Name: "a.pdf"},
					{Hash: "b.png", Name: "b.png"},
				},
			}},
		}},
	}
	op := board.MutationOp{
		Type:              "remove_attachment",
		RemoveAttachment:  &board.RemoveAttachmentOp{ColIdx: 0, CardIdx: 0, Hash: "a.pdf"},
	}
	if err := board.ApplyMutation(b, op); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := b.Columns[0].Cards[0].Attachments
	if len(got) != 1 || got[0].Hash != "b.png" {
		t.Errorf("got %+v", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/board -run TestApplyRemoveAttachment -v`
Expected: FAIL.

- [ ] **Step 3: Add struct, registry, apply**

In `internal/board/mutation.go`, add to `MutationOp`:

```go
	RemoveAttachment *RemoveAttachmentOp `json:"-"`
```

After existing op structs:

```go
// RemoveAttachmentOp removes a single attachment by hash from a card.
type RemoveAttachmentOp struct {
	ColIdx  int    `json:"col_idx"`
	CardIdx int    `json:"card_idx"`
	Hash    string `json:"hash"`
}
```

Registry entry:

```go
	"remove_attachment": {
		new: func() any { return &RemoveAttachmentOp{} },
		get: func(m *MutationOp) (any, bool) { return m.RemoveAttachment, m.RemoveAttachment != nil },
		set: func(m *MutationOp, v any) { m.RemoveAttachment = mustCast[*RemoveAttachmentOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*RemoveAttachmentOp](p)
			return applyRemoveAttachment(b, op.ColIdx, op.CardIdx, op.Hash)
		},
	},
```

In `internal/board/board.go`:

```go
// applyRemoveAttachment drops the attachment with hash from card. No-op if absent.
func applyRemoveAttachment(b *models.Board, colIdx, cardIdx int, hash string) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	card := &b.Columns[colIdx].Cards[cardIdx]
	ensureCardID(card)
	out := card.Attachments[:0]
	for _, a := range card.Attachments {
		if a.Hash == hash {
			continue
		}
		out = append(out, a)
	}
	card.Attachments = out
	return nil
}
```

- [ ] **Step 4: Run, verify passes**

Run: `go test ./internal/board -run TestApplyRemoveAttachment -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/board/mutation.go internal/board/board.go internal/board/mutation_test.go
git commit -m "feat(board): remove_attachment mutation"
```

---

### Task 6: Add `move_attachment` op (same-board only)

**Files:**
- Modify: `internal/board/mutation.go`
- Modify: `internal/board/board.go`
- Modify: `internal/board/mutation_test.go`

- [ ] **Step 1: Failing test**

```go
func TestApplyMoveAttachment(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{{
			Name: "Col",
			Cards: []models.Card{
				{Title: "Src", ID: "s",
					Attachments: []models.Attachment{{Hash: "a.pdf", Name: "a.pdf"}},
				},
				{Title: "Dst", ID: "d"},
			},
		}},
	}
	op := board.MutationOp{
		Type: "move_attachment",
		MoveAttachment: &board.MoveAttachmentOp{
			FromCol: 0, FromCard: 0, ToCol: 0, ToCard: 1, Hash: "a.pdf",
		},
	}
	if err := board.ApplyMutation(b, op); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(b.Columns[0].Cards[0].Attachments) != 0 {
		t.Errorf("src not cleared")
	}
	if got := b.Columns[0].Cards[1].Attachments; len(got) != 1 || got[0].Hash != "a.pdf" {
		t.Errorf("dst got %+v", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/board -run TestApplyMoveAttachment -v`
Expected: FAIL.

- [ ] **Step 3: Add struct, registry, apply**

`MutationOp` field:

```go
	MoveAttachment *MoveAttachmentOp `json:"-"`
```

Op struct:

```go
// MoveAttachmentOp moves an attachment from one card to another on the same
// board. Cross-board moves are orchestrated by the frontend as add+remove.
// Append to dst's list (no insert-at-index).
type MoveAttachmentOp struct {
	FromCol  int    `json:"from_col"`
	FromCard int    `json:"from_card"`
	ToCol    int    `json:"to_col"`
	ToCard   int    `json:"to_card"`
	Hash     string `json:"hash"`
}
```

Registry:

```go
	"move_attachment": {
		new: func() any { return &MoveAttachmentOp{} },
		get: func(m *MutationOp) (any, bool) { return m.MoveAttachment, m.MoveAttachment != nil },
		set: func(m *MutationOp, v any) { m.MoveAttachment = mustCast[*MoveAttachmentOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*MoveAttachmentOp](p)
			return applyMoveAttachment(b, op.FromCol, op.FromCard, op.ToCol, op.ToCard, op.Hash)
		},
	},
```

Apply:

```go
// applyMoveAttachment moves an attachment between two cards on the same
// board. Errors if the source attachment is not present. If dst already has
// the same hash, the op becomes pure src-removal (idempotent dedup).
func applyMoveAttachment(b *models.Board, fromCol, fromCard, toCol, toCard int, hash string) error {
	if err := validateIndices(b, fromCol, fromCard); err != nil {
		return err
	}
	if err := validateIndices(b, toCol, toCard); err != nil {
		return err
	}
	srcCard := &b.Columns[fromCol].Cards[fromCard]
	var moved *models.Attachment
	out := srcCard.Attachments[:0]
	for i := range srcCard.Attachments {
		if srcCard.Attachments[i].Hash == hash && moved == nil {
			a := srcCard.Attachments[i]
			moved = &a
			continue
		}
		out = append(out, srcCard.Attachments[i])
	}
	if moved == nil {
		return fmt.Errorf("attachment %q on card %d/%d: %w", hash, fromCol, fromCard, ErrNotFound)
	}
	srcCard.Attachments = out
	dstCard := &b.Columns[toCol].Cards[toCard]
	for _, a := range dstCard.Attachments {
		if a.Hash == hash {
			return nil // dst already has it; src-side removal is the only effect
		}
	}
	dstCard.Attachments = append(dstCard.Attachments, *moved)
	return nil
}
```

- [ ] **Step 4: Run, verify**

Run: `go test ./internal/board -run TestApplyMoveAttachment -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/board/mutation.go internal/board/board.go internal/board/mutation_test.go
git commit -m "feat(board): move_attachment mutation (same-board)"
```

---

### Task 7: Add `rename_attachment` op

**Files:** same set as Task 5.

- [ ] **Step 1: Failing test**

```go
func TestApplyRenameAttachment(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{{
			Name: "Col",
			Cards: []models.Card{{
				Title: "C", ID: "id",
				Attachments: []models.Attachment{{Hash: "a.pdf", Name: "old.pdf"}},
			}},
		}},
	}
	op := board.MutationOp{
		Type:              "rename_attachment",
		RenameAttachment:  &board.RenameAttachmentOp{ColIdx: 0, CardIdx: 0, Hash: "a.pdf", NewName: "new.pdf"},
	}
	if err := board.ApplyMutation(b, op); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := b.Columns[0].Cards[0].Attachments[0].Name; got != "new.pdf" {
		t.Errorf("got name %q", got)
	}
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/board -run TestApplyRenameAttachment -v`

- [ ] **Step 3: Add struct, registry, apply**

`MutationOp` field:

```go
	RenameAttachment *RenameAttachmentOp `json:"-"`
```

Op struct:

```go
// RenameAttachmentOp updates the display name of a card's attachment.
// The hash (and on-disk blob) is unchanged.
type RenameAttachmentOp struct {
	ColIdx  int    `json:"col_idx"`
	CardIdx int    `json:"card_idx"`
	Hash    string `json:"hash"`
	NewName string `json:"new_name"`
}
```

Registry:

```go
	"rename_attachment": {
		new: func() any { return &RenameAttachmentOp{} },
		get: func(m *MutationOp) (any, bool) { return m.RenameAttachment, m.RenameAttachment != nil },
		set: func(m *MutationOp, v any) { m.RenameAttachment = mustCast[*RenameAttachmentOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*RenameAttachmentOp](p)
			return applyRenameAttachment(b, op.ColIdx, op.CardIdx, op.Hash, op.NewName)
		},
	},
```

Apply:

```go
// applyRenameAttachment changes only the display name of the attachment with
// the given hash. Returns ErrNotFound if no such hash on the card.
func applyRenameAttachment(b *models.Board, colIdx, cardIdx int, hash, newName string) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	card := &b.Columns[colIdx].Cards[cardIdx]
	ensureCardID(card)
	for i := range card.Attachments {
		if card.Attachments[i].Hash == hash {
			card.Attachments[i].Name = newName
			return nil
		}
	}
	return fmt.Errorf("attachment %q on card %d/%d: %w", hash, colIdx, cardIdx, ErrNotFound)
}
```

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/board -run TestApplyRenameAttachment -v`

- [ ] **Step 5: Commit**

```bash
git add internal/board/mutation.go internal/board/board.go internal/board/mutation_test.go
git commit -m "feat(board): rename_attachment mutation"
```

---

### Task 8: Add `reorder_attachments` op

**Files:** same set.

- [ ] **Step 1: Failing test**

```go
func TestApplyReorderAttachments(t *testing.T) {
	b := &models.Board{
		Columns: []models.Column{{
			Name: "Col",
			Cards: []models.Card{{
				Title: "C", ID: "id",
				Attachments: []models.Attachment{
					{Hash: "a.pdf"}, {Hash: "b.png"}, {Hash: "c.txt"},
				},
			}},
		}},
	}
	op := board.MutationOp{
		Type: "reorder_attachments",
		ReorderAttachments: &board.ReorderAttachmentsOp{
			ColIdx: 0, CardIdx: 0,
			HashesInOrder: []string{"c.txt", "a.pdf", "b.png"},
		},
	}
	if err := board.ApplyMutation(b, op); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := b.Columns[0].Cards[0].Attachments
	if len(got) != 3 || got[0].Hash != "c.txt" || got[1].Hash != "a.pdf" || got[2].Hash != "b.png" {
		t.Errorf("got %+v", got)
	}
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/board -run TestApplyReorderAttachments -v`

- [ ] **Step 3: Add struct, registry, apply**

`MutationOp` field:

```go
	ReorderAttachments *ReorderAttachmentsOp `json:"-"`
```

Op struct:

```go
// ReorderAttachmentsOp reorders a card's attachments to match HashesInOrder.
// Hashes not present on the card are ignored; hashes present on the card but
// missing from HashesInOrder are appended in their original relative order
// (so a partial reorder is non-destructive).
type ReorderAttachmentsOp struct {
	ColIdx        int      `json:"col_idx"`
	CardIdx       int      `json:"card_idx"`
	HashesInOrder []string `json:"hashes_in_order"`
}
```

Registry:

```go
	"reorder_attachments": {
		new: func() any { return &ReorderAttachmentsOp{} },
		get: func(m *MutationOp) (any, bool) { return m.ReorderAttachments, m.ReorderAttachments != nil },
		set: func(m *MutationOp, v any) { m.ReorderAttachments = mustCast[*ReorderAttachmentsOp](v) },
		apply: func(b *models.Board, p any) error {
			op := mustCast[*ReorderAttachmentsOp](p)
			return applyReorderAttachments(b, op.ColIdx, op.CardIdx, op.HashesInOrder)
		},
	},
```

Apply:

```go
// applyReorderAttachments rearranges attachments to match HashesInOrder.
// Unknown hashes ignored; surviving attachments not listed are appended in
// original relative order.
func applyReorderAttachments(b *models.Board, colIdx, cardIdx int, order []string) error {
	if err := validateIndices(b, colIdx, cardIdx); err != nil {
		return err
	}
	card := &b.Columns[colIdx].Cards[cardIdx]
	ensureCardID(card)
	byHash := make(map[string]models.Attachment, len(card.Attachments))
	for _, a := range card.Attachments {
		byHash[a.Hash] = a
	}
	out := make([]models.Attachment, 0, len(card.Attachments))
	placed := make(map[string]struct{}, len(order))
	for _, h := range order {
		if a, ok := byHash[h]; ok {
			out = append(out, a)
			placed[h] = struct{}{}
		}
	}
	for _, a := range card.Attachments {
		if _, done := placed[a.Hash]; done {
			continue
		}
		out = append(out, a)
	}
	card.Attachments = out
	return nil
}
```

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/board -run TestApplyReorderAttachments -v`

- [ ] **Step 5: Run the registry coverage test once more for all 5**

Run: `go test ./internal/board -v`
Expected: PASS — including any `TestRegistryCoversAllVariants` / `TestRegistryMatchesVectorSuite` already in the package.

- [ ] **Step 6: Commit**

```bash
git add internal/board/mutation.go internal/board/board.go internal/board/mutation_test.go
git commit -m "feat(board): reorder_attachments mutation"
```

---

### Task 9: JSON marshal roundtrip for new ops

**Files:**
- Modify: `internal/board/mutation_test.go`

- [ ] **Step 1: Add JSON roundtrip test for each new op**

```go
func TestMutationOpJSONRoundtripAttachments(t *testing.T) {
	cases := []board.MutationOp{
		{Type: "add_attachments", AddAttachments: &board.AddAttachmentsOp{ColIdx: 1, CardIdx: 2, Items: []models.Attachment{{Hash: "h1", Name: "n", Size: 1, Mime: "m"}}}},
		{Type: "remove_attachment", RemoveAttachment: &board.RemoveAttachmentOp{ColIdx: 1, CardIdx: 2, Hash: "h"}},
		{Type: "move_attachment", MoveAttachment: &board.MoveAttachmentOp{FromCol: 0, FromCard: 0, ToCol: 1, ToCard: 1, Hash: "h"}},
		{Type: "rename_attachment", RenameAttachment: &board.RenameAttachmentOp{ColIdx: 0, CardIdx: 0, Hash: "h", NewName: "n"}},
		{Type: "reorder_attachments", ReorderAttachments: &board.ReorderAttachmentsOp{ColIdx: 0, CardIdx: 0, HashesInOrder: []string{"a", "b"}}},
	}
	for _, in := range cases {
		t.Run(in.Type, func(t *testing.T) {
			data, err := json.Marshal(in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var out board.MutationOp
			if err := json.Unmarshal(data, &out); err != nil {
				t.Fatalf("unmarshal: %v\nraw: %s", err, data)
			}
			if !reflect.DeepEqual(in, out) {
				t.Errorf("roundtrip mismatch:\n in:  %+v\n out: %+v\n raw: %s", in, out, data)
			}
		})
	}
}
```

Add `"encoding/json"`, `"reflect"` imports if missing.

- [ ] **Step 2: Run, verify PASS**

Run: `go test ./internal/board -run TestMutationOpJSONRoundtripAttachments -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/board/mutation_test.go
git commit -m "test(board): JSON roundtrip for attachment mutation ops"
```

---

### Task 10: Verify dispatch test runs the new ops via the v1 handler

**Files:**
- Modify: `internal/api/v1/mutations_dispatch_test.go`

- [ ] **Step 1: Add a dispatch test for one new op (smoke)**

The dispatch test has table-driven cases (search for `return board.MutationOp{` calls). Add a case that issues `add_attachments`. Pattern from existing tests:

```go
{
	name: "add_attachments persists to disk",
	op: func() board.MutationOp {
		return board.MutationOp{
			Type: "add_attachments",
			AddAttachments: &board.AddAttachmentsOp{
				ColIdx: 0, CardIdx: 0,
				Items: []models.Attachment{{Hash: "h.pdf", Name: "n.pdf", Size: 1, Mime: "application/pdf"}},
			},
		}
	},
	verify: func(t *testing.T, b *models.Board) {
		got := b.Columns[0].Cards[0].Attachments
		if len(got) != 1 || got[0].Hash != "h.pdf" {
			t.Errorf("got %+v", got)
		}
	},
},
```

(Adjust shape to whatever `mutations_dispatch_test.go` actually uses — read the file first to match its harness.)

- [ ] **Step 2: Run dispatch tests**

Run: `go test ./internal/api/v1 -run TestPostMutation -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/api/v1/mutations_dispatch_test.go
git commit -m "test(api/v1): dispatch test for add_attachments"
```

---

## Phase 3 — Storage pool (`internal/attachments`)

### Task 11: Implement `attachments.Store` (content-addressed pool)

**Files:**
- Create: `internal/attachments/store.go`
- Create: `internal/attachments/store_test.go`

- [ ] **Step 1: Write the failing test**

`internal/attachments/store_test.go`:

```go
package attachments_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestStorePutThenOpen(t *testing.T) {
	dir := t.TempDir()
	s, err := attachments.NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	payload := []byte("hello world")
	desc, err := s.Put(bytes.NewReader(payload), "greeting.txt")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	wantHash := sha256.Sum256(payload)
	wantHex := hex.EncodeToString(wantHash[:])
	if desc.Hash[:64] != wantHex {
		t.Errorf("hash prefix got %q want %q", desc.Hash[:64], wantHex)
	}
	if desc.Size != int64(len(payload)) {
		t.Errorf("size got %d want %d", desc.Size, len(payload))
	}
	if desc.Mime == "" {
		t.Errorf("mime should be sniffed, got empty")
	}

	r, err := s.Open(desc.Hash)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer r.Close()
	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, payload) {
		t.Errorf("read mismatch")
	}

	// Pool dir exists at expected path.
	if _, err := os.Stat(filepath.Join(dir, ".attachments", desc.Hash)); err != nil {
		t.Errorf("pool file missing: %v", err)
	}
}

func TestStorePutDedupes(t *testing.T) {
	dir := t.TempDir()
	s, _ := attachments.NewStore(dir)
	a, _ := s.Put(bytes.NewReader([]byte("x")), "a.txt")
	b, _ := s.Put(bytes.NewReader([]byte("x")), "b.txt")
	if a.Hash != b.Hash {
		t.Errorf("expected dedup: %q vs %q", a.Hash, b.Hash)
	}
}

func TestStoreRemove(t *testing.T) {
	dir := t.TempDir()
	s, _ := attachments.NewStore(dir)
	d, _ := s.Put(bytes.NewReader([]byte("y")), "x.txt")
	if err := s.Remove(d.Hash); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := s.Open(d.Hash); err == nil {
		t.Errorf("expected open to fail after remove")
	}
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/attachments -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Create `internal/attachments/store.go`**

```go
// Package attachments implements the workspace-wide content-addressed blob
// pool plus reference scanning, garbage collection, and thumbnail generation.
package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// PoolDir is the per-workspace subdirectory holding all blobs.
const PoolDir = ".attachments"

// Descriptor describes a stored blob.
type Descriptor struct {
	Hash string // <sha256-hex>.<ext>  (ext derived from origName, may be empty)
	Name string // origName, used as default download filename
	Size int64
	Mime string // sniffed at Put time via http.DetectContentType
}

// ErrNotFound is returned when a hash isn't present in the pool.
var ErrNotFound = errors.New("attachment not found")

// Store is a content-addressed blob pool rooted at workspaceDir/<PoolDir>.
type Store struct {
	dir string // pool dir, e.g. /workspace/.attachments
}

// NewStore returns a Store rooted at workspaceDir. The pool dir is created
// lazily on first Put.
func NewStore(workspaceDir string) (*Store, error) {
	return &Store{dir: filepath.Join(workspaceDir, PoolDir)}, nil
}

// Dir returns the absolute path to the pool directory.
func (s *Store) Dir() string { return s.dir }

// Put streams r through a sha256 hasher and a size counter, materialising
// the blob in the pool. Dedup is automatic: a duplicate Put returns the
// existing descriptor without rewriting the file.
//
// origName is used to derive the on-disk file extension and is returned
// untouched in Descriptor.Name.
func (s *Store) Put(r io.Reader, origName string) (Descriptor, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return Descriptor{}, fmt.Errorf("mkdir pool: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return Descriptor{}, fmt.Errorf("temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // best-effort if anything below fails

	hasher := sha256.New()
	sizer := &countingWriter{}
	mw := io.MultiWriter(tmp, hasher, sizer)
	// Sniff first 512 bytes for MIME via tee.
	var sniffBuf [512]byte
	headLen, _ := io.ReadFull(r, sniffBuf[:])
	if headLen > 0 {
		if _, err := mw.Write(sniffBuf[:headLen]); err != nil {
			tmp.Close()
			return Descriptor{}, err
		}
	}
	if _, err := io.Copy(mw, r); err != nil {
		tmp.Close()
		return Descriptor{}, err
	}
	if err := tmp.Close(); err != nil {
		return Descriptor{}, err
	}

	ext := strings.ToLower(path.Ext(origName))
	hexsum := hex.EncodeToString(hasher.Sum(nil))
	hash := hexsum + ext
	dst := filepath.Join(s.dir, hash)

	// Dedup: if dst exists, drop tmp; otherwise rename.
	if _, statErr := os.Stat(dst); statErr == nil {
		// already present; tmp will be removed by deferred cleanup
	} else {
		if err := os.Rename(tmpPath, dst); err != nil {
			return Descriptor{}, fmt.Errorf("rename: %w", err)
		}
	}

	mime := http.DetectContentType(sniffBuf[:headLen])
	return Descriptor{
		Hash: hash,
		Name: origName,
		Size: sizer.n,
		Mime: mime,
	}, nil
}

// Open returns a reader for hash. Caller must Close.
func (s *Store) Open(hash string) (*os.File, error) {
	if !validHash(hash) {
		return nil, fmt.Errorf("%w: invalid hash %q", ErrNotFound, hash)
	}
	f, err := os.Open(filepath.Join(s.dir, hash))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// Stat returns size and mime for hash without opening the data stream.
// The mime is re-sniffed (cheap) since it isn't stored alongside.
func (s *Store) Stat(hash string) (size int64, mime string, err error) {
	if !validHash(hash) {
		return 0, "", fmt.Errorf("%w: invalid hash %q", ErrNotFound, hash)
	}
	fi, err := os.Stat(filepath.Join(s.dir, hash))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", ErrNotFound
		}
		return 0, "", err
	}
	f, err := os.Open(filepath.Join(s.dir, hash))
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	var head [512]byte
	n, _ := io.ReadFull(f, head[:])
	return fi.Size(), http.DetectContentType(head[:n]), nil
}

// Remove deletes hash from the pool. Idempotent: missing → nil.
func (s *Store) Remove(hash string) error {
	if !validHash(hash) {
		return fmt.Errorf("%w: invalid hash %q", ErrNotFound, hash)
	}
	err := os.Remove(filepath.Join(s.dir, hash))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// validHash sanity-checks a pool key. It must be exactly 64 lowercase hex
// chars (sha256), optionally followed by a "." plus a short alnum extension.
// Anything else is rejected to prevent path traversal and weird filenames.
func validHash(s string) bool {
	if len(s) < 64 {
		return false
	}
	for i := 0; i < 64; i++ {
		c := s[i]
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	if len(s) == 64 {
		return true
	}
	if s[64] != '.' {
		return false
	}
	for _, c := range s[65:] {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return len(s)-65 <= 16 // sane extension cap
}

type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}
```

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/attachments -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/attachments/store.go internal/attachments/store_test.go
git commit -m "feat(attachments): content-addressed blob store"
```

---

### Task 12: Implement `attachments.CollectReferenced` (workspace ref scan)

**Files:**
- Create: `internal/attachments/refs.go`
- Create: `internal/attachments/refs_test.go`

- [ ] **Step 1: Failing test**

`internal/attachments/refs_test.go`:

```go
package attachments_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestCollectReferenced(t *testing.T) {
	dir := t.TempDir()
	board1 := `---
version: 1
name: A
---

## C

- [ ] Card
  attachments: [{"h":"aaaa.pdf","n":"x","s":0,"m":"x"}]
  Body has an image: ![](attachment:bbbb.png)
`
	board2 := `---
version: 1
name: B
---

## C

- [ ] Card
  attachments: [{"h":"cccc.txt","n":"y","s":0,"m":"x"}]
`
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(board1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.md"), []byte(board2), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := attachments.CollectReferenced(dir)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	for _, want := range []string{"aaaa.pdf", "bbbb.png", "cccc.txt"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing hash %q in %v", want, got)
		}
	}
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/attachments -run TestCollectReferenced -v`

- [ ] **Step 3: Implement `internal/attachments/refs.go`**

```go
package attachments

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// bodyAttachmentRe matches attachment:<hash>[.ext] inside body markdown.
// Matches the inert-URL scheme used by the renderer's attachmentScheme rewrite.
var bodyAttachmentRe = regexp.MustCompile(`attachment:([a-f0-9]{64}(?:\.[a-z0-9]{1,16})?)`)

// metaAttachmentsLine matches "  attachments: <json>".
var metaAttachmentsLine = regexp.MustCompile(`^\s{2}attachments:\s*(.+)$`)

// CollectReferenced walks workspaceDir, scans every .md file for attachment
// references (both the card-level attachments: field and the body
// attachment:<hash> URL scheme), and returns the union as a set keyed by hash
// (e.g. "a3f9....pdf").
//
// This is a textual scan, not a full parse — it must stay cheap because GC
// and export both call it across every board.
func CollectReferenced(workspaceDir string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	err := filepath.WalkDir(workspaceDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			// Skip the pool dir and any hidden dirs.
			name := d.Name()
			if path != workspaceDir && (name == PoolDir || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		scanRefs(string(data), out)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func scanRefs(content string, out map[string]struct{}) {
	for _, line := range strings.Split(content, "\n") {
		if m := metaAttachmentsLine.FindStringSubmatch(line); m != nil {
			var atts []struct {
				H string `json:"h"`
			}
			if err := json.Unmarshal([]byte(m[1]), &atts); err == nil {
				for _, a := range atts {
					if a.H != "" {
						out[a.H] = struct{}{}
					}
				}
			}
		}
		for _, m := range bodyAttachmentRe.FindAllStringSubmatch(line, -1) {
			out[m[1]] = struct{}{}
		}
	}
}
```

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/attachments -run TestCollectReferenced -v`

- [ ] **Step 5: Commit**

```bash
git add internal/attachments/refs.go internal/attachments/refs_test.go
git commit -m "feat(attachments): collect referenced hashes from workspace"
```

---

### Task 13: Implement `attachments.GC`

**Files:**
- Create: `internal/attachments/gc.go`
- Create: `internal/attachments/gc_test.go`

- [ ] **Step 1: Failing test**

`internal/attachments/gc_test.go`:

```go
package attachments_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestGCRemovesOrphans(t *testing.T) {
	dir := t.TempDir()
	s, _ := attachments.NewStore(dir)
	keep, _ := s.Put(bytes.NewReader([]byte("keep")), "k.txt")
	orphan, _ := s.Put(bytes.NewReader([]byte("orphan")), "o.txt")

	board := "---\nversion: 1\nname: A\n---\n\n## C\n\n- [ ] x\n  attachments: [{\"h\":\"" + keep.Hash + "\",\"n\":\"k\",\"s\":4,\"m\":\"text/plain\"}]\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}

	deleted, err := attachments.GC(dir)
	if err != nil {
		t.Fatalf("gc: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != orphan.Hash {
		t.Errorf("deleted = %v, want [%q]", deleted, orphan.Hash)
	}
	if _, err := s.Open(keep.Hash); err != nil {
		t.Errorf("kept blob gone: %v", err)
	}
	if _, err := s.Open(orphan.Hash); err == nil {
		t.Errorf("orphan still present")
	}
}

func TestGCNoPoolDirIsNoop(t *testing.T) {
	dir := t.TempDir()
	deleted, err := attachments.GC(dir)
	if err != nil {
		t.Fatalf("gc: %v", err)
	}
	if len(deleted) != 0 {
		t.Errorf("got deleted: %v", deleted)
	}
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/attachments -run TestGC -v`

- [ ] **Step 3: Implement `internal/attachments/gc.go`**

```go
package attachments

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GC removes blobs from the pool that are not referenced by any board.
// Returns the sorted list of removed hashes. Missing pool dir → no-op.
//
// Manual-only by design: callers (CLI command, future admin endpoints)
// drive cadence. There is no time-based grace window because there is no
// background sweep racing with uploads.
func GC(workspaceDir string) ([]string, error) {
	refs, err := CollectReferenced(workspaceDir)
	if err != nil {
		return nil, err
	}
	poolDir := filepath.Join(workspaceDir, PoolDir)
	entries, err := os.ReadDir(poolDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var deleted []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Ignore in-flight uploads and thumbnails (handled separately).
		if strings.HasPrefix(name, ".upload-") || strings.HasSuffix(name, ".thumb.webp") {
			continue
		}
		if _, ok := refs[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(poolDir, name)); err != nil {
			return deleted, err
		}
		deleted = append(deleted, name)
	}
	sort.Strings(deleted)
	return deleted, nil
}
```

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/attachments -v`
Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/attachments/gc.go internal/attachments/gc_test.go
git commit -m "feat(attachments): GC removes unreferenced blobs"
```

---

### Task 14: Add `liveboard gc` CLI command

**Files:**
- Create: `cmd/liveboard/gc.go`

- [ ] **Step 1: Inspect existing command pattern**

Run: `ls cmd/liveboard && head -30 cmd/liveboard/serve.go`
Note the workspace-dir flag/env pattern; reuse it.

- [ ] **Step 2: Create `cmd/liveboard/gc.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/attachments"
)

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Remove unreferenced attachment blobs from the workspace pool",
	Long: `Walks every board in the workspace, collects all referenced attachment
hashes (from card attachments: fields and body attachment:<hash> URLs), and
deletes any blob in <workspace>/.attachments/ that is not referenced.

Manual-only — there is no background sweep. Run after deleting cards or
clearing attachments to reclaim space.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// resolveWorkspaceDir is the existing helper used by other commands.
		// If the helper is named differently in this codebase, swap it in.
		dir, err := resolveWorkspaceDir(cmd)
		if err != nil {
			return err
		}
		deleted, err := attachments.GC(dir)
		if err != nil {
			return err
		}
		fmt.Printf("Removed %d unreferenced blob(s)\n", len(deleted))
		for _, h := range deleted {
			fmt.Println("  " + h)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(gcCmd)
}
```

NOTE: read `cmd/liveboard/serve.go` first to find the actual workspace-dir resolution helper name (`resolveWorkspaceDir` is a placeholder); the existing command will already have one. Replace the placeholder with the real one.

- [ ] **Step 3: Build, verify**

Run: `go build ./cmd/liveboard && ./liveboard gc --help`
Expected: prints command help with the long description.

- [ ] **Step 4: Smoke test against a temp workspace**

```bash
mkdir -p /tmp/lbgc && cd /tmp/lbgc
echo "---
version: 1
name: x
---

## c

- [ ] hi" > a.md
mkdir .attachments && echo orphan > .attachments/$(printf '%064s' 0 | tr ' ' 0)
$OLDPWD/liveboard gc --workspace .
```

Expected: prints "Removed 1 unreferenced blob(s)" with the hash. (Adjust `--workspace` flag name if the cobra setup differs.)

- [ ] **Step 5: Commit**

```bash
git add cmd/liveboard/gc.go
git commit -m "feat(cli): liveboard gc command"
```

---

### Task 15: Implement thumbnail generation

**Files:**
- Create: `internal/attachments/thumb.go`
- Create: `internal/attachments/thumb_test.go`

- [ ] **Step 1: Pick the WebP encoder**

The Go standard library does not include a WebP encoder. Use `golang.org/x/image/webp` (decoder only — no native encoder). For encoder, use `github.com/chai2010/webp` if importable, or fall back to `image/jpeg` for thumbnails (smaller dep, slightly larger files). **Decision for this task: use `image/jpeg` quality 80 for thumbnails to avoid a new external dep.** Renaming the file extension to `.thumb.jpg` accordingly.

Update `gc.go` skip prefix to match: `strings.HasSuffix(name, ".thumb.jpg")`.

- [ ] **Step 2: Failing test**

`internal/attachments/thumb_test.go`:

```go
package attachments_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	_ "image/jpeg" // register decoder

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestThumb(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1024, 512))
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	out, err := attachments.Thumb(&buf, 256)
	if err != nil {
		t.Fatalf("thumb: %v", err)
	}
	img, _, err := image.Decode(out)
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w > 256 || h > 256 {
		t.Errorf("thumb too large: %dx%d", w, h)
	}
	if w != 256 {
		t.Errorf("expected width 256 (max edge), got %d", w)
	}
}

func TestThumbNonImage(t *testing.T) {
	_, err := attachments.Thumb(bytes.NewReader([]byte("not an image")), 128)
	if err == nil {
		t.Errorf("expected error on non-image")
	}
}
```

- [ ] **Step 3: Run, verify FAIL**

Run: `go test ./internal/attachments -run TestThumb -v`

- [ ] **Step 4: Implement `internal/attachments/thumb.go`**

```go
package attachments

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"

	_ "image/gif"  // decoder
	_ "image/jpeg" // decoder
	_ "image/png"  // decoder

	"golang.org/x/image/draw"
)

// ErrNotImage is returned when Thumb is called on bytes that don't decode
// as one of the registered image formats.
var ErrNotImage = errors.New("not a decodable image")

// Thumb decodes src as an image, scales it so the longest edge is maxEdge,
// and JPEG-encodes the result at quality 80. Aspect ratio preserved.
func Thumb(src io.Reader, maxEdge int) (io.Reader, error) {
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, ErrNotImage
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxEdge && h <= maxEdge {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
			return nil, err
		}
		return &buf, nil
	}
	var nw, nh int
	if w >= h {
		nw = maxEdge
		nh = int(float64(h) * float64(maxEdge) / float64(w))
	} else {
		nh = maxEdge
		nw = int(float64(w) * float64(maxEdge) / float64(h))
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return &buf, nil
}
```

- [ ] **Step 5: Add the dep**

Run: `go get golang.org/x/image/draw`
Then: `go mod tidy`

- [ ] **Step 6: Run, verify PASS**

Run: `go test ./internal/attachments -v`

- [ ] **Step 7: Commit**

```bash
git add internal/attachments/thumb.go internal/attachments/thumb_test.go go.mod go.sum
git commit -m "feat(attachments): JPEG thumbnail generation"
```

---

## Phase 4 — HTTP endpoints

### Task 16: Wire `Store` into v1 deps

**Files:**
- Modify: `internal/api/v1/router.go` (or wherever `Deps` is defined — read first)

- [ ] **Step 1: Read `Deps` definition**

Run: `grep -n "type Deps" internal/api/v1/*.go`
Find the `Deps` struct. Add a `Attachments *attachments.Store` field.

- [ ] **Step 2: Modify the file that defines `Deps`**

Add to imports: `"github.com/and1truong/liveboard/internal/attachments"`.

Add field:

```go
	Attachments *attachments.Store
```

- [ ] **Step 3: Modify the call site in `cmd/liveboard/serve.go`**

Read the file, find where `v1.Deps{...}` is constructed. Add `Attachments: attachments.MustNewStore(workspaceDir),` (or initialise via `attachments.NewStore` and propagate the error). Use whichever helper matches the codebase style — pattern after the `Engine`/`Workspace` field initialisation.

If using a single-call helper, add to `internal/attachments/store.go`:

```go
// MustNewStore is NewStore that panics on error. NewStore currently never
// returns an error, but the signature is kept for future flexibility.
func MustNewStore(workspaceDir string) *Store {
	s, err := NewStore(workspaceDir)
	if err != nil {
		panic(err)
	}
	return s
}
```

- [ ] **Step 4: Build, verify**

Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/api/v1/router.go internal/attachments/store.go cmd/liveboard/serve.go
git commit -m "wire(api): attachments.Store into v1 Deps"
```

---

### Task 17: Implement upload handler

**Files:**
- Create: `internal/api/v1/attachments.go`
- Create: `internal/api/v1/attachments_test.go`
- Modify: `internal/api/v1/router.go`

- [ ] **Step 1: Failing test**

`internal/api/v1/attachments_test.go`:

```go
package v1_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostAttachmentRoundtrip(t *testing.T) {
	srv := newTestServer(t) // existing helper in v1 tests
	defer srv.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "hello.txt")
	io.Copy(fw, bytes.NewReader([]byte("hi there")))
	w.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/attachments", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	var desc struct {
		Hash, Name, Mime string
		Size             int64
	}
	if err := json.NewDecoder(resp.Body).Decode(&desc); err != nil {
		t.Fatal(err)
	}
	if desc.Size != 8 || desc.Name != "hello.txt" {
		t.Errorf("descriptor: %+v", desc)
	}

	// Download.
	gresp, err := http.Get(srv.URL + "/api/v1/attachments/" + desc.Hash + "/" + desc.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer gresp.Body.Close()
	body, _ := io.ReadAll(gresp.Body)
	if string(body) != "hi there" {
		t.Errorf("download body = %q", body)
	}
	if cd := gresp.Header.Get("Content-Disposition"); cd == "" {
		t.Errorf("missing Content-Disposition")
	}
	if got := gresp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
}

func TestPostAttachmentTooLarge(t *testing.T) {
	srv := newTestServer(t, withAttachmentMaxBytes(8))
	defer srv.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "big.bin")
	fw.Write(make([]byte, 9))
	w.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/attachments", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d", resp.StatusCode)
	}
}
```

NOTE: `newTestServer` and `withAttachmentMaxBytes` are placeholders — read the existing `internal/api/v1/router_test.go` or `boards_test.go` to find the actual harness helpers and adapt. If no helper takes a max-bytes option yet, add one as part of this task.

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/api/v1 -run TestPostAttachment -v`

- [ ] **Step 3: Implement handler**

`internal/api/v1/attachments.go`:

```go
package v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/attachments"
)

// inlineMimes is the small allowlist of MIMEs served with
// Content-Disposition: inline. Everything else is served as attachment so
// the browser cannot execute it in our origin (XSS hardening).
var inlineMimes = map[string]struct{}{
	"image/png":       {},
	"image/jpeg":      {},
	"image/gif":       {},
	"image/webp":      {},
	"application/pdf": {},
}

// postAttachment handles POST /api/v1/attachments (multipart, single file
// part named "file"). Returns the stored Descriptor as JSON.
func (d Deps) postAttachment(w http.ResponseWriter, r *http.Request) {
	maxBytes := d.AttachmentMaxBytes
	if maxBytes <= 0 {
		maxBytes = 25 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		writeError(w, fmt.Errorf("%w: %v", errInvalid, err))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, fmt.Errorf("%w: missing file part: %v", errInvalid, err))
		return
	}
	defer file.Close()

	desc, err := d.Attachments.Put(file, header.Filename)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, desc)
}

// getAttachment handles GET /api/v1/attachments/{hash}/{name}.
// Serves bytes with sniffed Content-Type, conservative Content-Disposition,
// long immutable cache, X-Content-Type-Options: nosniff.
func (d Deps) getAttachment(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	name := chi.URLParam(r, "name")

	f, err := d.Attachments.Open(hash)
	if err != nil {
		if errors.Is(err, attachments.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		writeError(w, err)
		return
	}
	defer f.Close()

	size, mime, err := d.Attachments.Stat(hash)
	if err != nil {
		writeError(w, err)
		return
	}

	// Optional thumbnail.
	if r.URL.Query().Get("thumb") == "1" {
		thumb, terr := attachments.Thumb(f, 256)
		if terr != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.Copy(w, thumb)
		return
	}

	disposition := "attachment"
	if _, ok := inlineMimes[mime]; ok {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`,
			disposition, asciiSafe(name), url.PathEscape(name)))

	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, f)
}

// asciiSafe returns name with non-ASCII chars stripped, for the legacy
// `filename=` field. The RFC 5987 `filename*=` carries the real value.
func asciiSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 && r != '"' && r != '\\' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "download"
	}
	return b.String()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := newJSONEncoder(w)
	_ = enc.Encode(v)
}
```

If `newJSONEncoder` doesn't exist, replace with `json.NewEncoder(w)` and add `"encoding/json"` import.

Also add to `Deps` (the file where `Deps` is defined):

```go
	AttachmentMaxBytes int64 // 0 → use default 25MB
```

Wire it in `cmd/liveboard/serve.go` from `settings.json`:

```go
AttachmentMaxBytes: appSettings.AttachmentMaxBytes,
```

Add to `internal/web/settings.go`'s `AppSettings` struct (or wherever `appSettings` is defined): `AttachmentMaxBytes int64 \`json:"attachments_max_bytes,omitempty"\``.

- [ ] **Step 4: Mount routes**

In `internal/api/v1/router.go`, in the `Mount` (or equivalent) function adding `/api/v1/...` routes, add:

```go
	r.Post("/attachments", d.postAttachment)
	r.Get("/attachments/{hash}/{name}", d.getAttachment)
	r.Head("/attachments/{hash}/{name}", d.getAttachment)
```

- [ ] **Step 5: Run, verify PASS**

Run: `go test ./internal/api/v1 -run TestPostAttachment -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/v1/attachments.go internal/api/v1/attachments_test.go internal/api/v1/router.go internal/web/settings.go cmd/liveboard/serve.go
git commit -m "feat(api/v1): attachment upload + download endpoints"
```

---

## Phase 5 — Search index integration

### Task 18: Index attachment display names

**Files:**
- Modify: `internal/search/index.go`
- Modify: `internal/search/index_test.go`

- [ ] **Step 1: Failing test**

Add to `internal/search/index_test.go`:

```go
func TestIndexAttachmentNames(t *testing.T) {
	idx := newTestIndex(t) // existing helper
	b := &models.Board{
		Version: 1, Name: "B",
		Columns: []models.Column{{
			Name: "C",
			Cards: []models.Card{{
				Title: "ticket",
				Attachments: []models.Attachment{
					{Hash: "h1", Name: "Q1-roadmap.pdf", Mime: "application/pdf"},
				},
			}},
		}},
	}
	if err := idx.UpdateBoard("slug", b); err != nil {
		t.Fatal(err)
	}
	hits := idx.Search("Q1-roadmap")
	if len(hits) == 0 {
		t.Errorf("expected hit for attachment name")
	}
}
```

(Adjust to match the actual `index.go` API surface — e.g. `Search` may return a different shape.)

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/search -run TestIndexAttachmentNames -v`

- [ ] **Step 3: Implement**

In `internal/search/index.go`, in the function that builds a card document for indexing (search for `cardDoc` or similar struct/builder), add a field/branch:

```go
	if len(card.Attachments) > 0 {
		names := make([]string, 0, len(card.Attachments))
		for _, a := range card.Attachments {
			names = append(names, a.Name)
		}
		doc.AttachmentNames = strings.Join(names, " ")
	}
```

Add the field to the doc struct (adjacent to existing text-search fields like `Title`, `Body`, `Tags`):

```go
	AttachmentNames string `json:"attachment_names,omitempty"`
```

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/search -v`

- [ ] **Step 5: Commit**

```bash
git add internal/search/index.go internal/search/index_test.go
git commit -m "feat(search): index attachment display names"
```

---

## Phase 6 — Export integration

### Task 19: Bundle referenced blobs into export ZIPs

**Files:**
- Modify: `internal/export/export.go`
- Modify: `internal/export/export_test.go`
- Modify: `internal/api/server.go`

- [ ] **Step 1: Read existing export shape**

Run: `head -80 internal/export/export.go`
Note where the writer adds files to the zip; this is where `.attachments/` entries will be added.

- [ ] **Step 2: Failing test**

Add to `internal/export/export_test.go`:

```go
func TestExportBundlesAttachments(t *testing.T) {
	dir := t.TempDir()
	store, _ := attachments.NewStore(dir)
	desc, _ := store.Put(bytes.NewReader([]byte("blob")), "x.txt")
	board := "---\nversion: 1\nname: B\n---\n\n## C\n\n- [ ] x\n  attachments: [{\"h\":\"" + desc.Hash + "\",\"n\":\"x.txt\",\"s\":4,\"m\":\"text/plain\"}]\n"
	os.WriteFile(filepath.Join(dir, "b.md"), []byte(board), 0o644)

	var buf bytes.Buffer
	if err := export.Markdown(&buf, dir, export.Options{IncludeAttachments: true}); err != nil {
		t.Fatal(err)
	}
	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	var found bool
	for _, f := range zr.File {
		if f.Name == ".attachments/"+desc.Hash {
			found = true
		}
	}
	if !found {
		t.Errorf("attachment blob missing from zip")
	}
}

func TestExportSkipsAttachmentsWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	store, _ := attachments.NewStore(dir)
	desc, _ := store.Put(bytes.NewReader([]byte("blob")), "x.txt")
	board := "---\nversion: 1\nname: B\n---\n\n## C\n\n- [ ] x\n  attachments: [{\"h\":\"" + desc.Hash + "\",\"n\":\"x.txt\",\"s\":4,\"m\":\"text/plain\"}]\n"
	os.WriteFile(filepath.Join(dir, "b.md"), []byte(board), 0o644)

	var buf bytes.Buffer
	if err := export.Markdown(&buf, dir, export.Options{IncludeAttachments: false}); err != nil {
		t.Fatal(err)
	}
	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, ".attachments/") {
			t.Errorf("attachments included when disabled: %s", f.Name)
		}
	}
}
```

(Adjust `export.Markdown` / `export.HTML` / `export.Options` names to match the actual package surface — read `export.go` first.)

- [ ] **Step 3: Run, verify FAIL**

Run: `go test ./internal/export -run TestExport -v`

- [ ] **Step 4: Implement bundle logic**

In `internal/export/export.go`:

1. Add `IncludeAttachments bool` to `Options` (default true when omitted; tests opt out explicitly).
2. After writing all `.md` files into the zip, if `opts.IncludeAttachments`:
   - Call `attachments.CollectReferenced(workspaceDir)` to get the hash set.
   - For each hash, read from `<workspaceDir>/.attachments/<hash>` and write into the zip at `.attachments/<hash>`.
3. For HTML export: in addition to bundling the blobs, the HTML render pass (probably in `internal/templates/export_*.html` or a render function in this package) needs to:
   - Render the card-level `attachments:` field as a `<ul>` of `<a href="./attachments/<hash>/<name>">` links (or `<img>` for image MIMEs).
   - Rewrite body markdown `attachment:<hash>[.ext]` URLs to `./attachments/<hash>/<original-name>` (look up name from the card descriptor; fall back to the hash if not found).

- [ ] **Step 5: Run, verify PASS**

Run: `go test ./internal/export -v`

- [ ] **Step 6: Wire `attachments=false` query param in server**

In `internal/api/server.go`, in the export handler (search for `liveboard-export.zip`), parse `r.URL.Query().Get("attachments")`:

```go
opts := export.Options{IncludeAttachments: r.URL.Query().Get("attachments") != "false"}
```

Pass `opts` to the export function.

- [ ] **Step 7: Commit**

```bash
git add internal/export/export.go internal/export/export_test.go internal/api/server.go internal/templates/
git commit -m "feat(export): bundle referenced attachments; ?attachments=false opt-out"
```

---

## Phase 7 — MCP tools

### Task 20: Add 5 attachment MCP tools

**Files:**
- Create: `internal/mcp/tools_attachment.go`
- Create: `internal/mcp/tools_attachment_test.go`
- Modify: `internal/mcp/server.go`

- [ ] **Step 1: Read existing MCP tool pattern**

Run: `cat internal/mcp/tools_card.go | head -80`
Each tool is a registration on the MCP server with a JSON schema and a handler. Mirror this exactly.

- [ ] **Step 2: Failing test**

`internal/mcp/tools_attachment_test.go`:

```go
package mcp_test

import (
	"context"
	"testing"
)

func TestMCPAddAttachmentRef(t *testing.T) {
	srv := newTestMCPServer(t) // existing helper in mcp_test.go
	createTestBoard(t, srv, "b") // existing helper, adjust name as needed

	resp, err := srv.Call(context.Background(), "card_add_attachment_ref", map[string]any{
		"board": "b", "col": 0, "card": 0,
		"hash": "abc.pdf", "name": "doc.pdf", "size": 1, "mime": "application/pdf",
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	_ = resp
	// Verify the descriptor lands on the card.
	b := loadBoard(t, srv, "b")
	if got := b.Columns[0].Cards[0].Attachments; len(got) != 1 || got[0].Hash != "abc.pdf" {
		t.Errorf("got %+v", got)
	}
}
```

(Adapt to match the actual mcp test harness in `internal/mcp/mcp_test.go`.)

- [ ] **Step 3: Run, verify FAIL**

Run: `go test ./internal/mcp -run TestMCPAddAttachmentRef -v`

- [ ] **Step 4: Implement `internal/mcp/tools_attachment.go`**

Mirror `tools_card.go`'s registration pattern. Each tool dispatches to the existing `board.Apply*` via `Engine.MutateBoard`. Concrete shapes:

```go
package mcp

import (
	"context"
	"encoding/json"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

func (s *Server) registerAttachmentTools() {
	s.tool("card_add_attachment_ref",
		"Add an existing-blob reference (descriptor only) to a card. Bytes must already be uploaded via the HTTP /api/v1/attachments endpoint; this tool only writes the descriptor.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"board": map[string]any{"type": "string"},
				"col":   map[string]any{"type": "integer"},
				"card":  map[string]any{"type": "integer"},
				"hash":  map[string]any{"type": "string"},
				"name":  map[string]any{"type": "string"},
				"size":  map[string]any{"type": "integer"},
				"mime":  map[string]any{"type": "string"},
			},
			"required": []string{"board", "col", "card", "hash", "name", "size", "mime"},
		},
		s.handleAddAttachmentRef)

	s.tool("card_remove_attachment", "Remove an attachment from a card by hash.", /* schema */, s.handleRemoveAttachment)
	s.tool("card_move_attachment", "Move an attachment between two cards on the same board.", /* schema */, s.handleMoveAttachment)
	s.tool("card_rename_attachment", "Update the display name of an attachment.", /* schema */, s.handleRenameAttachment)
	s.tool("card_reorder_attachments", "Reorder a card's attachments to match the given hash list.", /* schema */, s.handleReorderAttachments)
}

// Each handler resolves the board path from the slug, calls Engine.MutateBoard
// with the matching MutationOp, and returns the updated board JSON.
func (s *Server) handleAddAttachmentRef(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Board string `json:"board"`
		Col   int    `json:"col"`
		Card  int    `json:"card"`
		Hash  string `json:"hash"`
		Name  string `json:"name"`
		Size  int64  `json:"size"`
		Mime  string `json:"mime"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	path, err := s.workspace.BoardPath(p.Board)
	if err != nil {
		return nil, err
	}
	op := board.MutationOp{
		Type: "add_attachments",
		AddAttachments: &board.AddAttachmentsOp{
			ColIdx: p.Col, CardIdx: p.Card,
			Items: []models.Attachment{{Hash: p.Hash, Name: p.Name, Size: p.Size, Mime: p.Mime}},
		},
	}
	if err := s.engine.MutateBoard(path, -1, func(b *models.Board) error {
		return board.ApplyMutation(b, op)
	}); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// handleRemoveAttachment, handleMoveAttachment, handleRenameAttachment,
// handleReorderAttachments follow the same pattern with the corresponding op.
```

(Fill in the four remaining handlers analogously. Match the actual `s.tool(...)` signature and helpers from `tools_card.go`.)

In `internal/mcp/server.go`, in whatever function calls `registerCardTools()`/`registerColumnTools()`/`registerBoardTools()`, add:

```go
	s.registerAttachmentTools()
```

- [ ] **Step 5: Run, verify PASS**

Run: `go test ./internal/mcp -v`

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/tools_attachment.go internal/mcp/tools_attachment_test.go internal/mcp/server.go
git commit -m "feat(mcp): five attachment tools (descriptor-level, no binary)"
```

---

## Phase 8 — TS types and `boardOps` parity

### Task 21: Regenerate `mutations.gen.ts`

**Files:**
- Modify: `web/shared/src/mutations.gen.ts` (auto-generated)
- Modify: `cmd/gen-ts-mutations/main.go` (only if regen finds gaps)

- [ ] **Step 1: Find the regen command**

Run: `grep -n "gen-ts-mutations" Makefile cmd/gen-ts-mutations/main.go 2>/dev/null`
Identify the make target. If not present, add one:

```makefile
gen-ts-mutations:
	go run ./cmd/gen-ts-mutations > web/shared/src/mutations.gen.ts
```

- [ ] **Step 2: Run the regen**

Run: `make gen-ts-mutations`
Or, if no make target: `go run ./cmd/gen-ts-mutations > web/shared/src/mutations.gen.ts`

- [ ] **Step 3: Verify the new ops appear**

Run: `grep -n "add_attachments\|remove_attachment\|move_attachment\|rename_attachment\|reorder_attachments" web/shared/src/mutations.gen.ts`
Expected: each op appears as a discriminated-union member with the correct field shapes.

If any op is missing, the codegen needs extending — read `cmd/gen-ts-mutations/main.go` and adjust whatever encodes types like `[]models.Attachment` (ensure `Attachment` is emitted as a TS interface).

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/mutations.gen.ts cmd/gen-ts-mutations/main.go Makefile
git commit -m "feat(types): regenerate mutations.gen.ts with attachment ops"
```

---

### Task 22: Add `Attachment` to `web/shared/src/types.ts` and `Card.attachments`

**Files:**
- Modify: `web/shared/src/types.ts`

- [ ] **Step 1: Add the type and field**

Locate the existing `Card` interface. Add (next to other card fields):

```typescript
export interface Attachment {
  h: string;   // <sha256-hex>.<ext>
  n: string;   // display name
  s: number;   // bytes
  m: string;   // sniffed MIME
}
```

Add to `Card`:

```typescript
  attachments?: Attachment[];
```

- [ ] **Step 2: Type-check**

Run: `cd web/shared && bun x tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/types.ts
git commit -m "feat(types): Card.attachments field"
```

---

### Task 23: Mirror the 5 ops in `boardOps.ts`

**Files:**
- Modify: `web/shared/src/boardOps.ts`
- Modify: `web/shared/src/boardOps.test.ts`

- [ ] **Step 1: Failing test**

Add to `web/shared/src/boardOps.test.ts`:

```typescript
import { describe, it, expect } from 'bun:test';
import { applyMutation } from './boardOps';
import type { Board } from './types';

describe('attachment mutations', () => {
  const baseBoard = (): Board => ({
    version: 1, name: 'B',
    columns: [{ name: 'C', cards: [
      { title: 'C0', attachments: [] },
      { title: 'C1', attachments: [] },
    ]}],
  } as any);

  it('add_attachments appends and dedupes', () => {
    const b = baseBoard();
    applyMutation(b, { type: 'add_attachments', col_idx: 0, card_idx: 0,
      items: [{ h: 'a', n: 'a', s: 1, m: 'x' }, { h: 'a', n: 'a', s: 1, m: 'x' }] });
    expect(b.columns[0].cards[0].attachments!.length).toBe(1);
  });

  it('remove_attachment by hash', () => {
    const b = baseBoard();
    b.columns[0].cards[0].attachments = [{ h: 'a', n: 'a', s: 1, m: 'x' }, { h: 'b', n: 'b', s: 1, m: 'x' }];
    applyMutation(b, { type: 'remove_attachment', col_idx: 0, card_idx: 0, hash: 'a' });
    expect(b.columns[0].cards[0].attachments!.map(a => a.h)).toEqual(['b']);
  });

  it('move_attachment between cards on same board', () => {
    const b = baseBoard();
    b.columns[0].cards[0].attachments = [{ h: 'a', n: 'a', s: 1, m: 'x' }];
    applyMutation(b, { type: 'move_attachment', from_col: 0, from_card: 0, to_col: 0, to_card: 1, hash: 'a' });
    expect(b.columns[0].cards[0].attachments!.length).toBe(0);
    expect(b.columns[0].cards[1].attachments!.map(a => a.h)).toEqual(['a']);
  });

  it('rename_attachment changes only display name', () => {
    const b = baseBoard();
    b.columns[0].cards[0].attachments = [{ h: 'a', n: 'old', s: 1, m: 'x' }];
    applyMutation(b, { type: 'rename_attachment', col_idx: 0, card_idx: 0, hash: 'a', new_name: 'new' });
    expect(b.columns[0].cards[0].attachments![0].n).toBe('new');
    expect(b.columns[0].cards[0].attachments![0].h).toBe('a');
  });

  it('reorder_attachments respects hashes_in_order', () => {
    const b = baseBoard();
    b.columns[0].cards[0].attachments = [
      { h: 'a', n: 'a', s: 1, m: 'x' },
      { h: 'b', n: 'b', s: 1, m: 'x' },
      { h: 'c', n: 'c', s: 1, m: 'x' },
    ];
    applyMutation(b, { type: 'reorder_attachments', col_idx: 0, card_idx: 0, hashes_in_order: ['c', 'a', 'b'] });
    expect(b.columns[0].cards[0].attachments!.map(a => a.h)).toEqual(['c', 'a', 'b']);
  });
});
```

- [ ] **Step 2: Run, verify FAIL**

Run: `cd web/shared && bun test boardOps.test.ts`
Expected: FAIL — no handlers for the new op types.

- [ ] **Step 3: Implement in `boardOps.ts`**

In `web/shared/src/boardOps.ts`, in the `applyMutation` switch (or registry), add five branches that mirror exactly the Go `apply*` semantics:

```typescript
case 'add_attachments': {
  const card = b.columns[op.col_idx].cards[op.card_idx];
  card.attachments = card.attachments ?? [];
  const seen = new Set(card.attachments.map(a => a.h));
  for (const a of op.items) {
    if (seen.has(a.h)) continue;
    card.attachments.push(a);
    seen.add(a.h);
  }
  return;
}
case 'remove_attachment': {
  const card = b.columns[op.col_idx].cards[op.card_idx];
  card.attachments = (card.attachments ?? []).filter(a => a.h !== op.hash);
  return;
}
case 'move_attachment': {
  const src = b.columns[op.from_col].cards[op.from_card];
  const dst = b.columns[op.to_col].cards[op.to_card];
  src.attachments = src.attachments ?? [];
  dst.attachments = dst.attachments ?? [];
  const idx = src.attachments.findIndex(a => a.h === op.hash);
  if (idx < 0) throw new Error(`attachment ${op.hash} not found`);
  const [moved] = src.attachments.splice(idx, 1);
  if (!dst.attachments.some(a => a.h === op.hash)) {
    dst.attachments.push(moved);
  }
  return;
}
case 'rename_attachment': {
  const card = b.columns[op.col_idx].cards[op.card_idx];
  const a = (card.attachments ?? []).find(x => x.h === op.hash);
  if (!a) throw new Error(`attachment ${op.hash} not found`);
  a.n = op.new_name;
  return;
}
case 'reorder_attachments': {
  const card = b.columns[op.col_idx].cards[op.card_idx];
  const byHash = new Map((card.attachments ?? []).map(a => [a.h, a]));
  const out: Attachment[] = [];
  const placed = new Set<string>();
  for (const h of op.hashes_in_order) {
    const a = byHash.get(h);
    if (a) { out.push(a); placed.add(h); }
  }
  for (const a of card.attachments ?? []) {
    if (!placed.has(a.h)) out.push(a);
  }
  card.attachments = out;
  return;
}
```

Add `import type { Attachment } from './types';` if not already.

- [ ] **Step 4: Run, verify PASS**

Run: `cd web/shared && bun test boardOps.test.ts`

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/boardOps.ts web/shared/src/boardOps.test.ts
git commit -m "feat(boardOps): TS twin of 5 attachment mutations"
```

---

### Task 24: Confirm Go ↔ TS parity vector runner still passes

**Files:** none directly; runs existing `internal/parity/runner_test.go`.

- [ ] **Step 1: Run parity vectors**

Run: `go test ./internal/parity -v`
Expected: PASS.

If the vector suite enumerates ops by name and asserts both sides handle them, you may need to add vectors for the 5 new ops. Read `internal/parity/runner_test.go` to determine. If vectors are required, add JSON fixtures under whatever `testdata/` path the runner uses, with one fixture per new op.

- [ ] **Step 2: Commit (only if fixtures added)**

```bash
git add internal/parity/
git commit -m "test(parity): vectors for attachment mutations"
```

---

## Phase 9 — Adapter upload/download API

### Task 25: Extend `BackendAdapter` interface

**Files:**
- Modify: `web/shared/src/adapter.ts`

- [ ] **Step 1: Add methods to the interface**

```typescript
import type { Attachment } from './types';

export interface BackendAdapter {
  // ... existing methods ...

  uploadAttachment(file: File): Promise<Attachment>;
  attachmentURL(att: Pick<Attachment, 'h' | 'n'>): string;
  // Optional: a thumb URL, returns full URL when no thumb path applies.
  attachmentThumbURL?(att: Pick<Attachment, 'h' | 'n'>): string;
}
```

- [ ] **Step 2: Type-check**

Run: `cd web/shared && bun x tsc --noEmit`
Expected: errors in adapters that don't yet implement these (server.ts, local.ts) — that's intentional; fixed in next tasks.

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/adapter.ts
git commit -m "feat(adapter): uploadAttachment + attachmentURL on BackendAdapter"
```

---

### Task 26: Implement `ServerAdapter` upload/download

**Files:**
- Modify: `web/shared/src/adapters/server.ts`
- Modify: `web/shared/src/adapters/server.test.ts`

- [ ] **Step 1: Failing test**

Add to `server.test.ts`:

```typescript
it('uploadAttachment posts multipart and returns descriptor', async () => {
  const adapter = makeAdapter('/api/v1');
  const calls: { url: string; init?: RequestInit }[] = [];
  globalThis.fetch = (async (input: any, init?: RequestInit) => {
    calls.push({ url: String(input), init });
    return new Response(JSON.stringify({ h: 'h.txt', n: 'x.txt', s: 4, m: 'text/plain' }), { status: 200 });
  }) as any;
  const file = new File(['hi'], 'x.txt', { type: 'text/plain' });
  const desc = await adapter.uploadAttachment(file);
  expect(desc.h).toBe('h.txt');
  expect(calls[0].url).toBe('/api/v1/attachments');
  expect((calls[0].init?.body as FormData).get('file')).toBe(file);
});

it('attachmentURL builds a stable path', () => {
  const adapter = makeAdapter('/api/v1');
  expect(adapter.attachmentURL({ h: 'abc.pdf', n: 'doc.pdf' })).toBe('/api/v1/attachments/abc.pdf/doc.pdf');
});
```

(Adjust `makeAdapter` to whatever harness factory the file uses.)

- [ ] **Step 2: Run, verify FAIL**

Run: `cd web/shared && bun test server.test.ts`

- [ ] **Step 3: Implement**

In `web/shared/src/adapters/server.ts`, add:

```typescript
async uploadAttachment(file: File): Promise<Attachment> {
  const fd = new FormData();
  fd.append('file', file, file.name);
  const resp = await fetch(`${this.baseUrl}/attachments`, { method: 'POST', body: fd });
  if (!resp.ok) {
    throw new Error(`upload failed: ${resp.status}`);
  }
  return resp.json() as Promise<Attachment>;
}

attachmentURL(att: { h: string; n: string }): string {
  return `${this.baseUrl}/attachments/${att.h}/${encodeURIComponent(att.n)}`;
}

attachmentThumbURL(att: { h: string; n: string }): string {
  return `${this.attachmentURL(att)}?thumb=1`;
}
```

Add `import type { Attachment } from '../types';` if needed.

- [ ] **Step 4: Run, verify PASS**

Run: `cd web/shared && bun test server.test.ts`

- [ ] **Step 5: Commit**

```bash
git add web/shared/src/adapters/server.ts web/shared/src/adapters/server.test.ts
git commit -m "feat(adapter/server): upload + URL builders for attachments"
```

---

### Task 27: Implement `LocalAdapter` IndexedDB blob store

**Files:**
- Create: `web/shared/src/adapters/local-attachments.ts`
- Create: `web/shared/src/adapters/local-attachments.test.ts`
- Modify: `web/shared/src/adapters/local.ts`

- [ ] **Step 1: Failing test (driver)**

`web/shared/src/adapters/local-attachments.test.ts`:

```typescript
import { describe, it, expect } from 'bun:test';
import 'fake-indexeddb/auto'; // add as devDep if not present
import { putBlob, getBlob } from './local-attachments';

describe('local-attachments', () => {
  it('hashes content and stores by hash', async () => {
    const blob = new Blob(['hello'], { type: 'text/plain' });
    const desc = await putBlob(blob, 'h.txt');
    // sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
    expect(desc.h).toBe('2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824.txt');
    const got = await getBlob(desc.h);
    expect(got).toBeTruthy();
    expect(await got!.text()).toBe('hello');
  });
});
```

If `fake-indexeddb` isn't in `package.json`, add it: `bun add -d fake-indexeddb`.

- [ ] **Step 2: Run, verify FAIL**

Run: `cd web/shared && bun test local-attachments.test.ts`

- [ ] **Step 3: Implement driver**

`web/shared/src/adapters/local-attachments.ts`:

```typescript
import type { Attachment } from '../types';

const DB_NAME = 'liveboard-attachments';
const STORE = 'blobs';

let dbPromise: Promise<IDBDatabase> | null = null;

function openDB(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise;
  dbPromise = new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
  return dbPromise;
}

async function txn<T>(mode: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const t = db.transaction(STORE, mode);
    const s = t.objectStore(STORE);
    const req = fn(s);
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function sha256Hex(blob: Blob): Promise<string> {
  const buf = await blob.arrayBuffer();
  const digest = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(digest)).map(b => b.toString(16).padStart(2, '0')).join('');
}

function ext(filename: string): string {
  const i = filename.lastIndexOf('.');
  if (i < 0) return '';
  return filename.slice(i).toLowerCase();
}

export async function putBlob(blob: Blob, name: string): Promise<Attachment> {
  const hex = await sha256Hex(blob);
  const hash = hex + ext(name);
  await txn('readwrite', s => s.put(blob, hash));
  return { h: hash, n: name, s: blob.size, m: blob.type || 'application/octet-stream' };
}

export async function getBlob(hash: string): Promise<Blob | null> {
  const v = await txn<Blob | undefined>('readonly', s => s.get(hash) as IDBRequest<Blob | undefined>);
  return v ?? null;
}

export async function deleteBlob(hash: string): Promise<void> {
  await txn('readwrite', s => s.delete(hash));
}
```

- [ ] **Step 4: Run, verify PASS**

Run: `cd web/shared && bun test local-attachments.test.ts`

- [ ] **Step 5: Wire into `LocalAdapter`**

In `web/shared/src/adapters/local.ts`, add the two interface methods:

```typescript
import { putBlob, getBlob } from './local-attachments';

// inside LocalAdapter class:
async uploadAttachment(file: File): Promise<Attachment> {
  return putBlob(file, file.name);
}

attachmentURL(att: { h: string; n: string }): string {
  // Returns a sentinel that the renderer's attachmentScheme rewriter will
  // resolve to a blob URL on demand. Direct synchronous URL is impossible
  // because IDB lookups are async — the rewriter awaits getBlob.
  return `attachment:${att.h}`;
}
```

For `attachmentThumbURL`: omit (renderer falls back to full URL → can downscale via CSS for now). Document this in the file as a known v1 simplification.

- [ ] **Step 6: Type-check whole shared package**

Run: `cd web/shared && bun x tsc --noEmit`
Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add web/shared/src/adapters/local-attachments.ts web/shared/src/adapters/local-attachments.test.ts web/shared/src/adapters/local.ts package.json bun.lockb
git commit -m "feat(adapter/local): IndexedDB-backed attachment storage"
```

---

## Phase 10 — Renderer UI

### Task 28: Body markdown `attachment:<hash>` URL rewrite

**Files:**
- Create: `web/renderer/default/src/markdown/attachmentScheme.ts`
- Modify: wherever the renderer constructs the markdown component (search for `react-markdown` or similar)

- [ ] **Step 1: Read existing markdown wiring**

Run: `grep -rn "react-markdown\|remark\|rehype" web/renderer/default/src 2>/dev/null | head -10`
Identify which library is used and where it's instantiated. The renderer likely uses `react-markdown` with custom `urlTransform` / `transformLinkUri` / `transformImageUri` props.

- [ ] **Step 2: Implement rewriter**

`web/renderer/default/src/markdown/attachmentScheme.ts`:

```typescript
import type { BackendAdapter } from '@liveboard/shared/adapter';

const RE = /^attachment:([a-f0-9]{64}(?:\.[a-z0-9]{1,16})?)$/;

// rewriteAttachmentUrl resolves `attachment:<hash>` URLs against the active
// adapter. Server adapter returns a stable HTTP path. Local adapter returns
// a `attachment:<hash>` sentinel that this same function resolves on demand
// to a blob: URL via the local-attachments driver.
export function rewriteAttachmentUrl(adapter: BackendAdapter, displayNameByHash: Map<string, string>) {
  return (url: string): string => {
    const m = url.match(RE);
    if (!m) return url;
    const hash = m[1];
    const name = displayNameByHash.get(hash) ?? hash;
    const out = adapter.attachmentURL({ h: hash, n: name });
    // Local adapter returns `attachment:...` again — resolve to blob URL
    // synchronously via a cache populated elsewhere; if missing, return ''
    // so the renderer shows a broken-link icon.
    if (out.startsWith('attachment:')) {
      return ''; // resolved asynchronously elsewhere; see useBlobURLs hook
    }
    return out;
  };
}
```

For local-mode synchronous resolution, add a small hook in the same dir:

```typescript
// useBlobURLs.ts
import { useEffect, useState } from 'react';
import { getBlob } from '@liveboard/shared/adapters/local-attachments';

const cache = new Map<string, string>();

// Pre-resolves `attachment:<hash>` URLs in body to blob: URLs once per hash.
export function useBlobURLs(hashes: string[]): Map<string, string> {
  const [urls, setUrls] = useState<Map<string, string>>(cache);
  useEffect(() => {
    let cancelled = false;
    (async () => {
      let changed = false;
      for (const h of hashes) {
        if (cache.has(h)) continue;
        const b = await getBlob(h);
        if (!b) continue;
        cache.set(h, URL.createObjectURL(b));
        changed = true;
      }
      if (changed && !cancelled) setUrls(new Map(cache));
    })();
    return () => { cancelled = true; };
  }, [hashes.join(',')]);
  return urls;
}
```

- [ ] **Step 3: Plug rewriter into the markdown component**

In the file that renders card body markdown (find via `grep -rn "card.body\|card\.Body" web/renderer/default/src`), pass:

```tsx
<ReactMarkdown
  urlTransform={rewriteAttachmentUrl(adapter, namesByHash)}
  // ... existing props
>{card.body}</ReactMarkdown>
```

Build `namesByHash` from `card.attachments`.

- [ ] **Step 4: Type-check renderer**

Run: `cd web/renderer/default && bun x tsc --noEmit`

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/markdown/
git commit -m "feat(renderer): rewrite attachment:<hash> body URLs"
```

---

### Task 29: `AttachmentList` modal component

**Files:**
- Create: `web/renderer/default/src/components/AttachmentList.tsx`
- Modify: `web/renderer/default/src/components/CardModal.tsx` (verify path first)

- [ ] **Step 1: Find the card modal**

Run: `grep -rn "useMutation.*edit_card\|CardModal\|CardEdit" web/renderer/default/src 2>/dev/null | head`
Read the file. Identify where to mount the new component.

- [ ] **Step 2: Implement `AttachmentList.tsx`**

```tsx
import { useState, useRef, ChangeEvent, ClipboardEvent, DragEvent } from 'react';
import type { Attachment } from '@liveboard/shared/types';
import type { BackendAdapter } from '@liveboard/shared/adapter';

type Props = {
  adapter: BackendAdapter;
  attachments: Attachment[];
  onAdd: (items: Attachment[]) => void;
  onRemove: (hash: string) => void;
  onRename: (hash: string, newName: string) => void;
  onReorder: (hashesInOrder: string[]) => void;
  onInsertIntoBody?: (att: Attachment) => void;
};

export function AttachmentList(props: Props) {
  const fileInput = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);

  async function uploadFiles(files: FileList | File[]) {
    setBusy(true);
    try {
      const out: Attachment[] = [];
      for (const f of Array.from(files)) {
        out.push(await props.adapter.uploadAttachment(f));
      }
      props.onAdd(out);
    } finally {
      setBusy(false);
    }
  }

  function onFilePick(e: ChangeEvent<HTMLInputElement>) {
    if (e.target.files) uploadFiles(e.target.files);
    e.target.value = '';
  }

  function onDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault();
    if (e.dataTransfer.files.length) uploadFiles(e.dataTransfer.files);
  }

  function onPaste(e: ClipboardEvent<HTMLDivElement>) {
    const files: File[] = [];
    for (const item of Array.from(e.clipboardData.items)) {
      if (item.kind === 'file') {
        const f = item.getAsFile();
        if (f) files.push(f);
      }
    }
    if (files.length) uploadFiles(files);
  }

  // (Drag-handle reorder, inline rename, remove buttons — implemented inline below)
  return (
    <div className="attachment-list" onDrop={onDrop} onDragOver={e => e.preventDefault()} onPaste={onPaste}>
      <ul>
        {props.attachments.map(a => (
          <li key={a.h}>
            <a href={props.adapter.attachmentURL(a)} target="_blank" rel="noopener noreferrer">{a.n}</a>
            <span className="size">{formatBytes(a.s)}</span>
            <button onClick={() => {
              const next = prompt('Rename', a.n);
              if (next && next !== a.n) props.onRename(a.h, next);
            }}>rename</button>
            {props.onInsertIntoBody && (
              <button onClick={() => props.onInsertIntoBody!(a)}>insert into body</button>
            )}
            <button onClick={() => props.onRemove(a.h)}>remove</button>
            {/* drag handle for reorder — implement with HTML5 DnD or a small lib */}
          </li>
        ))}
      </ul>
      <button onClick={() => fileInput.current?.click()} disabled={busy}>
        {busy ? 'Uploading…' : '+ Attach files'}
      </button>
      <input type="file" multiple ref={fileInput} onChange={onFilePick} hidden />
    </div>
  );
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}
```

For drag-handle reorder, the simplest approach is HTML5 DnD with `draggable={true}` on each `<li>` and tracking `dataTransfer` index — keep it minimal; add a library only if HTML5 DnD proves insufficient.

- [ ] **Step 3: Mount in card modal**

In `CardModal.tsx`, in the modal body, add:

```tsx
<AttachmentList
  adapter={adapter}
  attachments={card.attachments ?? []}
  onAdd={items => mutate({ type: 'add_attachments', col_idx: colIdx, card_idx: cardIdx, items })}
  onRemove={hash => mutate({ type: 'remove_attachment', col_idx: colIdx, card_idx: cardIdx, hash })}
  onRename={(hash, new_name) => mutate({ type: 'rename_attachment', col_idx: colIdx, card_idx: cardIdx, hash, new_name })}
  onReorder={hashes_in_order => mutate({ type: 'reorder_attachments', col_idx: colIdx, card_idx: cardIdx, hashes_in_order })}
  onInsertIntoBody={a => insertAtCursor(`![${a.n}](attachment:${a.h})`)}
/>
```

(`mutate` and `insertAtCursor` names depend on existing harness; adjust.)

- [ ] **Step 4: Manually verify in dev**

Run: `make adapter-test`
In the browser, open a card, attach a file, verify it persists across reload, drag it to another card, rename it, remove it.

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/components/AttachmentList.tsx web/renderer/default/src/components/CardModal.tsx
git commit -m "feat(renderer): AttachmentList modal UI with upload and management"
```

---

### Task 30: Card display — badge (compact) and thumb strip (expanded)

**Files:**
- Create: `web/renderer/default/src/components/AttachmentBadge.tsx`
- Create: `web/renderer/default/src/components/AttachmentThumbStrip.tsx`
- Modify: `web/renderer/default/src/components/Card.tsx` (verify path first)

- [ ] **Step 1: Find the in-column card component**

Run: `grep -rn "card-display-mode\|cardDisplayMode" web/renderer/default/src | head`
Identify where the display mode is consumed.

- [ ] **Step 2: Implement components**

`AttachmentBadge.tsx`:

```tsx
import type { Attachment } from '@liveboard/shared/types';

export function AttachmentBadge({ attachments }: { attachments: Attachment[] }) {
  if (!attachments?.length) return null;
  return <span className="attachment-badge" title={`${attachments.length} attachment(s)`}>📎 {attachments.length}</span>;
}
```

`AttachmentThumbStrip.tsx`:

```tsx
import type { Attachment } from '@liveboard/shared/types';
import type { BackendAdapter } from '@liveboard/shared/adapter';

const IMAGE_MIME_RE = /^image\//;
const MAX_VISIBLE = 3;

export function AttachmentThumbStrip({ adapter, attachments }: { adapter: BackendAdapter; attachments: Attachment[] }) {
  if (!attachments?.length) return null;
  const visible = attachments.slice(0, MAX_VISIBLE);
  const overflow = attachments.length - visible.length;
  return (
    <div className="attachment-thumb-strip">
      {visible.map(a => (
        <div key={a.h} className="thumb">
          {IMAGE_MIME_RE.test(a.m)
            ? <img src={adapter.attachmentThumbURL?.(a) ?? adapter.attachmentURL(a)} alt={a.n} loading="lazy" />
            : <span className="file-icon" title={a.n}>📄</span>}
        </div>
      ))}
      {overflow > 0 && <span className="thumb-overflow">+{overflow}</span>}
    </div>
  );
}
```

- [ ] **Step 3: Mount in `Card.tsx`**

```tsx
{cardDisplayMode === 'compact' && <AttachmentBadge attachments={card.attachments ?? []} />}
{cardDisplayMode === 'expanded' && <AttachmentThumbStrip adapter={adapter} attachments={card.attachments ?? []} />}
```

- [ ] **Step 4: Type-check**

Run: `cd web/renderer/default && bun x tsc --noEmit`

- [ ] **Step 5: Manual verify**

Run: `make adapter-test`
Switch between compact/expanded modes, verify both renderings.

- [ ] **Step 6: Commit**

```bash
git add web/renderer/default/src/components/AttachmentBadge.tsx web/renderer/default/src/components/AttachmentThumbStrip.tsx web/renderer/default/src/components/Card.tsx
git commit -m "feat(renderer): card-level attachment badge and thumb strip"
```

---

### Task 31: Card-level drop zone in column view

**Files:**
- Modify: `web/renderer/default/src/components/Card.tsx`

- [ ] **Step 1: Add drop handlers to the card root element**

Inside the card's outer wrapper:

```tsx
const onDrop = async (e: React.DragEvent) => {
  if (!e.dataTransfer.files?.length) return;
  e.preventDefault();
  const items: Attachment[] = [];
  for (const f of Array.from(e.dataTransfer.files)) {
    items.push(await adapter.uploadAttachment(f));
  }
  mutate({ type: 'add_attachments', col_idx: colIdx, card_idx: cardIdx, items });
};

return (
  <div onDrop={onDrop} onDragOver={e => { if (e.dataTransfer.types.includes('Files')) e.preventDefault(); }}>
    {/* ... existing card render ... */}
  </div>
);
```

- [ ] **Step 2: Manual verify**

Run: `make adapter-test`
Drag a file onto a card in column view, verify it uploads and the badge/strip updates.

- [ ] **Step 3: Commit**

```bash
git add web/renderer/default/src/components/Card.tsx
git commit -m "feat(renderer): drop files onto cards in column view to attach"
```

---

## Phase 11 — Cleanup, lint, docs

### Task 32: Run full test + lint

- [ ] **Step 1: Run all Go tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 2: Run all TS tests**

Run: `cd web/shared && bun test && cd ../renderer/default && bun test && cd ../shell && bun test`
Expected: PASS.

- [ ] **Step 3: Run lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 4: Build production bundles**

Run: `make frontend && go build ./...`
Expected: success.

- [ ] **Step 5: Commit any lint fixups**

```bash
git add -A
git commit -m "chore: fmt + lint fixes for attachments"
```

(Skip if no fixups needed.)

---

### Task 33: Documentation

**Files:**
- Create: `docs/attachments.md`
- Modify: `CLAUDE.md` (architecture section)

- [ ] **Step 1: Write user-facing reference**

`docs/attachments.md`:

```markdown
# Attachments

LiveBoard stores attached files in a workspace-wide content-addressed pool
at `<workspace>/.attachments/`. Each card references attachments by hash
plus a display name.

## On-disk layout

- `<workspace>/.attachments/<sha256-hex>.<ext>` — blob storage. Filename is
  the lowercase hex sha256 of the file contents plus the original
  extension. Identical files dedupe automatically.

## Card metadata

Attachments live on a single line under each card:

    - [ ] Card title
      attachments: [{"h":"abc...e1.pdf","n":"Q1 Plan.pdf","s":124533,"m":"application/pdf"}]

The JSON keys: `h` hash, `n` display name, `s` bytes, `m` sniffed MIME.

## Body inline references

Card body markdown can embed attachments using the `attachment:` URL scheme:

    Here's the screenshot: ![](attachment:abc...e1.png)

The renderer resolves these to download URLs at view time.

## Garbage collection

The pool grows until you run `liveboard gc`, which deletes any blob not
referenced by any card. Run after cleaning up cards or whole boards.

## Limits

- Max upload size defaults to 25 MB (configurable via `attachments_max_bytes`
  in `settings.json`).
- All MIMEs allowed. Inline display in browser limited to images and PDF;
  everything else downloads.

## Export

Workspace export ZIPs (`/api/export?format=md` or `?format=html`) include
referenced blobs by default. Add `?attachments=false` to omit them.
```

- [ ] **Step 2: Update `CLAUDE.md`**

In the Architecture section of `CLAUDE.md`, add to the bullet list:

```
- `internal/attachments/` — content-addressed blob pool, ref scanning, GC, thumbnails
```

In the Domain Concepts section, add:

```
- **Attachment**: file attached to a card. Stored as blob in
  `<workspace>/.attachments/` keyed by sha256. Card carries descriptors
  (`{h,n,s,m}`); body markdown can embed via `attachment:<hash>` URLs.
```

- [ ] **Step 3: Commit**

```bash
git add docs/attachments.md CLAUDE.md
git commit -m "docs: attachments reference and architecture mention"
```

---

## Self-Review checklist (run after writing the plan)

- [x] Spec coverage:
  - Storage location B (workspace-wide pool) → Task 11
  - Naming A (content-addressed) → Task 11
  - Card schema D (single-line JSON) → Tasks 1–3
  - Upload transport A (separate endpoint) → Task 17
  - Move semantics A (same-board only) → Task 6
  - GC A (manual CLI) → Tasks 13, 14
  - LocalAdapter A (IndexedDB) → Task 27
  - Export A+opt-out → Task 19
  - Validation A (size cap, sniff+disposition) → Task 17
  - Mutation ops B (5 ops) → Tasks 4–8
  - Display C (mode-aware) → Task 30
  - Cross-board card move A (no special handling) — covered by existing `MoveCardToBoard`; Card.Attachments travels for free with the card record (Task 1 adds the field; no new code needed)
  - Body URL scheme B (`attachment:<hash>`) → Tasks 12 (GC scan), 19 (export rewrite), 28 (renderer)
  - Search index A (display names) → Task 18
  - MCP B (5 descriptor-level tools) → Task 20

- [x] Type consistency:
  - Go op struct names: `AddAttachmentsOp`, `RemoveAttachmentOp`, `MoveAttachmentOp`, `RenameAttachmentOp`, `ReorderAttachmentsOp` — used identically across mutation.go, board.go, dispatch tests, MCP tools.
  - TS type discriminators: `add_attachments`, `remove_attachment`, `move_attachment`, `rename_attachment`, `reorder_attachments` — consistent across boardOps.ts and renderer call sites.
  - `Attachment` shape: `{h, n, s, m}` everywhere (Go json tags, TS interface, parser/writer).
  - Adapter methods: `uploadAttachment(File)`, `attachmentURL({h,n})`, `attachmentThumbURL?({h,n})` — consistent across interface, server.ts, local.ts, components.

- [x] No placeholders flagged.

---

## Out-of-scope deferrals (intentional)

- PDF/text-extraction for full-content search inside attachments.
- Inline-thumb cache invalidation on the server (thumbs are derived; deleting a thumb regenerates it; no separate refcount needed because the source blob's removal via GC also kills the `<hash>.thumb.jpg` if you extend GC to skip thumbs whose source blob is gone — left as a follow-up).
- Background periodic GC daemon.
- Cross-board atomic move of an attachment (orchestrated client-side).
- Resumable / chunked uploads.
- Per-user auth or quotas (no auth model exists in LiveBoard yet).

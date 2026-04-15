# P6.3b — Cross-Board Card Linking — Design

## Goal

Add forward links (stored) and backlinks (derived) between cards across boards. A link target is `boardSlug:cardId` — readable in markdown, rename-safe (resolved by `cardId`). The picker reuses `useSearch`. Backlinks render as a "Linked from" section in `<CardDetailModal>`.

**Shippable value:** users can navigate from a card to related cards anywhere in the workspace and see what links to the current card. Closes the cross-board linking item from the README roadmap.

## Scope

**In:**
- `Card.links: string[]` field on Go and TS models.
- `links: a, b, c` line in markdown metadata (parser + writer).
- `edit_card` op carries `links: string[]`.
- `<CardDetailModal>` Links section: chips + remove + "+ Add link" picker.
- `<LinkPicker>` component reusing `useSearch`.
- `<CardDetailModal>` backlinks section showing reverse references (read-only).
- `BackendAdapter.backlinks(cardId)` on both adapters.
- `GET /api/v1/cards/{cardId}/backlinks` endpoint.
- `SearchHit` extended with `cardId` so the picker can build the link target.
- bleve `Doc` adds `links` (keyword-tokenized).

**Out:**
- Inline link chips on the card body (clutter; modal-only for P6.3b).
- Auto-cleanup of broken links (user removes manually via ✕).
- Slug-rename sweep that updates referrers (resolution is by cardId, so no sweep needed).
- Drag-to-link UX.
- Link types / labels (just a flat `links` list).
- Two-way sync (edit a backlink → no auto-edit on the source).

## Architecture

```
Markdown
  - [ ] Card title
    id: aBc1234XyZ
    links: welcome:Q9rT5pZ2nM, planning:Lm0pNq8sBv

Card.links (Go + TS): []string
  | each entry: "<boardSlug>:<cardId>"

Forward links: stored, edited via edit_card op.
Backlinks: derived on demand.
  Server: bleve query (links field, keyword tokenizer, suffix match for cardId).
  Local: in-process scan over loaded boards.

CardDetailModal
  ├─ existing fields
  ├─ NEW: Links section
  │   └─ chips (resolveLink labels) + ✕ remove + "+ Add link" → <LinkPicker>
  └─ NEW: Linked from section
      └─ chips from useBacklinks(card.id), click navigates
```

Adding/removing a link = `edit_card` mutation that includes the modified `links` array. No new ops.

## Wire shapes

### Markdown

`links` line under a card uses comma-separated entries, identical pattern to `tags`:
```
- [ ] Card title
  id: aBc1234XyZ
  links: welcome:Q9rT5pZ2nM, planning:Lm0pNq8sBv
  tags: backend
```

Empty / missing → `card.links == nil`.

### `edit_card` op

```ts
| {
    type: 'edit_card'
    col_idx: number
    card_idx: number
    title: string
    body: string
    tags: string[]
    links: string[]      // NEW
    priority: string
    due: string
    assignee: string
  }
```

Go side mirror:
```go
type EditCardOp struct {
    ColIdx   int      `json:"col_idx"`
    CardIdx  int      `json:"card_idx"`
    Title    string   `json:"title"`
    Body     string   `json:"body"`
    Tags     []string `json:"tags"`
    Links    []string `json:"links"`     // NEW
    Priority string   `json:"priority"`
    Due      string   `json:"due"`
    Assignee string   `json:"assignee"`
}
```

### `SearchHit` extension

```ts
export interface SearchHit {
  boardId: string
  boardName: string
  colIdx: number
  cardIdx: number
  cardId: string       // NEW
  cardTitle: string
  snippet: string
}
```

bleve doc was updated in P6.3a to include `card_id`. Search handler surfaces it now.

ServerAdapter and LocalAdapter both populate `cardId` in their search results.

### `BacklinkHit`

```ts
export interface BacklinkHit {
  boardId: string
  boardName: string
  colIdx: number
  cardIdx: number
  cardTitle: string
}
```

No snippet (backlinks aren't search results).

### `BackendAdapter.backlinks`

```ts
interface BackendAdapter {
  // ...
  backlinks(cardId: string): Promise<BacklinkHit[]>
}
```

### HTTP endpoint

`GET /api/v1/cards/{cardId}/backlinks` → 200 + JSON array of:
```json
{
  "board_id": "welcome",
  "board_name": "Welcome",
  "col_idx": 0,
  "card_idx": 3,
  "card_title": "Read the docs"
}
```

Errors: `INVALID` if cardId is empty.

## File structure

**Modified (Go):**
- `pkg/models/models.go` — `Card.Links []string`.
- `pkg/models/mutation.go` — `EditCardOp.Links`.
- `internal/parser/parser.go` — read `links: a, b, c` (split on comma, trim).
- `internal/parser/parser_test.go` — round-trip a card with links.
- `internal/writer/writer.go` — emit `links: …` (alphabetical position; comma-joined like tags).
- `internal/writer/writer_test.go` — assert links are emitted.
- `internal/board/board.go` — `edit_card` op apply path writes `Links`.
- `internal/board/board_test.go` — assert links are persisted.
- `internal/search/index.go` — `Doc.Links []string`; index as keyword-tokenized; `Backlinks(cardId)` method; `SearchHit` Hit struct gains `CardID`.
- `internal/search/index_test.go` — backlinks fixture.
- `internal/api/v1/cards.go` (new) — `GET /cards/{cardId}/backlinks` handler.
- `internal/api/v1/cards_test.go` (new).
- `internal/api/v1/router.go` — register route.
- `internal/api/v1/search.go` — surface `card_id` in DTO.

**Modified (TS shared):**
- `web/shared/src/types.ts` — `Card.links?`, `MutationOp.edit_card.links`.
- `web/shared/src/boardOps.ts` — `applyOp` for `edit_card` writes `card.links`.
- `web/shared/src/adapter.ts` — `SearchHit.cardId`, `BacklinkHit`, `backlinks` method.
- `web/shared/src/protocol.ts` — `backlinks` Request variant.
- `web/shared/src/broker.ts` — route `backlinks` → adapter.
- `web/shared/src/client.ts` — passthrough.
- `web/shared/src/adapters/server.ts` — implement `backlinks` via HTTP; surface `cardId` in `search`.
- `web/shared/src/adapters/local.ts` — implement scan-based `backlinks`; surface `cardId` in `search`.

**Modified (TS renderer):**
- `web/renderer/default/src/components/CardDetailModal.tsx` — add Links + Linked-from sections.
- `web/renderer/default/src/components/LinkPicker.tsx` (new).
- `web/renderer/default/src/queries/useBacklinks.ts` (new).
- `web/renderer/default/src/queries/useResolveLink.ts` (new).

## Component contracts

### `<LinkPicker open onPick onClose />`

Props:
```ts
{
  open: boolean
  onPick: (target: string) => void  // "boardSlug:cardId"
  onClose: () => void
  excludeBoardId?: string
  excludeCardId?: string
}
```

Inline `<Dialog>` (or popover) with a `<Command>` from cmdk. Uses `useSearch(query)` to populate hits. Each hit row is a `Command.Item` whose `onSelect` builds `${hit.boardId}:${hit.cardId}` and calls `onPick`. `excludeBoardId` + `excludeCardId` filter the current card out of its own picker.

### Forward links section in `CardDetailModal`

```tsx
const [pickerOpen, setPickerOpen] = useState(false)
const [links, setLinks] = useState<string[]>(card.links ?? [])

const addLink = (target: string) => {
  if (links.includes(target)) return
  setLinks([...links, target])
  setPickerOpen(false)
}
const removeLink = (target: string) => setLinks(links.filter(l => l !== target))

// On Save (existing submit), include links:
mutation.mutate({
  type: 'edit_card',
  // ...other existing fields...
  links,
})
```

UI:
```tsx
<section>
  <header className="text-xs font-medium text-slate-600 dark:text-slate-300">Links</header>
  <ul className="mt-1 flex flex-col gap-1">
    {links.map(target => <LinkChip key={target} target={target} onRemove={() => removeLink(target)} />)}
  </ul>
  <button type="button" onClick={() => setPickerOpen(true)}>+ Add link</button>
</section>
{pickerOpen && (
  <LinkPicker open onPick={addLink} onClose={() => setPickerOpen(false)}
    excludeBoardId={boardId} excludeCardId={card.id} />
)}
```

### `<LinkChip target onRemove>`

Resolves the target via `useResolveLink(target)`. Shows `<Board name> · <Card title>` or "(missing)" if resolution returns null. Click navigates: `setActive(boardId)` + `setFocused({ colIdx, cardIdx })`. ✕ button calls `onRemove`.

### Backlinks section

```tsx
const backlinks = useBacklinks(card.id)
{backlinks.length > 0 && (
  <section>
    <header className="text-xs font-medium text-slate-600 dark:text-slate-300">Linked from</header>
    <ul className="mt-1 flex flex-col gap-1">
      {backlinks.map(b => (
        <li key={`${b.boardId}:${b.cardIdx}`}>
          <button type="button" onClick={() => navigateTo(b)}>
            {b.boardName} · {b.cardTitle}
          </button>
        </li>
      ))}
    </ul>
  </section>
)}
```

### `useBacklinks(cardId: string)`

```ts
export function useBacklinks(cardId: string | undefined): BacklinkHit[] {
  const client = useClient()
  const q = useQuery({
    queryKey: ['backlinks', cardId],
    queryFn: () => client.backlinks(cardId!),
    enabled: !!cardId,
  })
  return q.data ?? []
}
```

### `useResolveLink(target: string)`

```ts
export function useResolveLink(target: string) {
  const client = useClient()
  return useQuery({
    queryKey: ['resolve', target],
    queryFn: async () => {
      const [boardSlug, cardId] = target.split(':')
      if (!boardSlug || !cardId) return null
      const board = await client.getBoard(boardSlug)
      const cols = board.columns ?? []
      for (let c = 0; c < cols.length; c++) {
        const cards = cols[c]?.cards ?? []
        for (let k = 0; k < cards.length; k++) {
          if (cards[k].id === cardId) {
            return {
              boardName: board.name ?? boardSlug,
              cardTitle: cards[k].title ?? '',
              colIdx: c,
              cardIdx: k,
            }
          }
        }
      }
      return null
    },
  })
}
```

## bleve indexing change

`Doc.Links []string`. Index `links` with a keyword analyzer so each entry stays whole-token (otherwise `boardSlug:cardId` tokenizes to two terms, breaking suffix queries):

```go
mapping := bleve.NewIndexMapping()
keywordField := bleve.NewTextFieldMapping()
keywordField.Analyzer = "keyword"
docMapping := bleve.NewDocumentMapping()
docMapping.AddFieldMappingsAt("links", keywordField)
mapping.AddDocumentMapping("_default", docMapping)
```

`Backlinks(cardId string) ([]Hit, error)`:
```go
q := bleve.NewWildcardQuery("*:" + cardId)
q.SetField("links")
sr := bleve.NewSearchRequestOptions(q, 100, 0, false)
sr.Fields = []string{"board_id", "board_name", "col_idx", "card_idx", "title"}
res, err := i.idx.Search(sr)
// map to Hit shape (no snippet)
```

`Search`'s response Hit struct gains `CardID string` and surfaces `card_id` field.

## HTTP handler

`internal/api/v1/cards.go`:
```go
func (d Deps) getBacklinks(w http.ResponseWriter, r *http.Request) {
    cardID := chi.URLParam(r, "cardId")
    if cardID == "" {
        writeError(w, fmt.Errorf("%w: cardId required", errInvalid))
        return
    }
    if d.Search == nil {
        _ = json.NewEncoder(w).Encode([]struct{}{})
        return
    }
    hits, err := d.Search.Backlinks(cardID)
    if err != nil {
        writeError(w, err)
        return
    }
    out := make([]map[string]interface{}, 0, len(hits))
    for _, h := range hits {
        out = append(out, map[string]interface{}{
            "board_id":   h.BoardID,
            "board_name": h.BoardName,
            "col_idx":    h.ColIdx,
            "card_idx":   h.CardIdx,
            "card_title": h.CardTitle,
        })
    }
    _ = json.NewEncoder(w).Encode(out)
}
```

Router: `r.Get("/cards/{cardId}/backlinks", d.getBacklinks)`.

## Edit-card call-site updates

Every place that constructs an `edit_card` op now must include `links`. Audit:
- `<CardEditable>` inline title edit — preserves `card.links`.
- `<CardDetailModal>` Save — uses the local `links` state.
- Anywhere else? grep.

Engine `EditCardOp.Apply` sets `card.Links = op.Links`.

Optimistic apply mirror: `applyOp` for `edit_card` writes `card.links = op.links`.

## Manual smoke

1. Open a card detail modal. New "Links" section appears empty.
2. Click "+ Add link" → picker opens.
3. Type a query → matching cards from search appear.
4. Pick one → returns to modal; chip appears with target's title.
5. Save → modal closes; markdown file shows `links: <board>:<id>` line.
6. Open the same card again → chip persists.
7. Open the linked target card's modal → "Linked from" section shows the source card.
8. Click chip on either card → navigates to the target board, focuses the target card.
9. Delete the target card → reopen the source's modal → chip shows "(missing)".
10. Click ✕ on a chip → chip removed; Save persists.

## Risks

- **bleve keyword tokenizer**: must be applied at index-mapping time. Existing index from P6.2 uses default analyzer; a mapping change requires reindexing. Index is in-memory and rebuilt on startup, so the change takes effect on next start.
- **`SearchHit.cardId` propagation**: bleve doc field exists; Hit struct extension is mechanical; but every consumer of `SearchHit` must read it. Likely just `CommandPalette` (for navigation) and `LinkPicker`. Renderer compiles via the type system.
- **Optimistic ID drift**: link picker must not allow picking optimistic-only cards. Mitigation: hits come from search → search comes from server-confirmed state → safe.
- **Self-linking**: a card linking to itself is allowed but useless. UI excludes via `excludeCardId` prop on `LinkPicker`.
- **Duplicate links**: `addLink` short-circuits if the target is already in the list.
- **Backlinks performance**: bleve query is fast; LocalAdapter scan is O(N cards). Both fine for typical workspaces.
- **Stale resolveLink cache**: when a target card is renamed, the chip label drifts until cache invalidation. TanStack Query invalidates `['resolve', target]` when a board mutation hits that board id. Add this invalidation to `useBoardMutation.onSuccess`: `qc.invalidateQueries({ queryKey: ['resolve'] })` on every board mutation.
- **`edit_card.links` propagation in optimistic apply**: easy to miss — `applyOp` mirror must include the field. Test catches.

## Open questions

None blocking. Pre-decided:
- Format: `boardSlug:cardId`, comma-separated in markdown.
- Resolution by cardId only.
- Forward (stored) + backlinks (derived).
- Picker via `useSearch`.
- Adding `cardId` to `SearchHit`.
- bleve `links` indexed as keyword.

## Dependencies on prior work

- P6.3a: `card.id` exists and is set on every mutation-touched card.
- P6.2: bleve index, `<CommandPalette>`, `useSearch`.
- P4b.3: `<CardDetailModal>` exists.
- P4c.3: `BoardFocusContext.setFocused` for navigation after click.
- P4c.1: `useActiveBoard` for `setActive`.
- P5: ServerAdapter + Broker + Client passthrough patterns.

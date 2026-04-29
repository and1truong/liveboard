# Attachments

LiveBoard stores files attached to cards in a workspace-wide
content-addressed pool at `<workspace>/.attachments/`. Cards reference
attachments by hash plus a display name.

## On-disk layout

- `<workspace>/.attachments/<sha256-hex>.<ext>` — blob storage. The filename
  is the lowercase hex SHA-256 of the file contents plus the original
  extension. Identical files dedupe automatically.

The pool dir is created lazily on first upload. Hidden directories and the
pool itself are skipped by reference scans, so dropping a `.md` file inside
the pool dir won't be parsed as a board.

## Card metadata

Each card carries its attachments as a single JSON-encoded metadata line:

```markdown
- [ ] Card title
  attachments: [{"h":"a3f9...e1.pdf","n":"Q1 Plan.pdf","s":124533,"m":"application/pdf"}]
```

Fields:

| Key | Meaning                                              |
|-----|------------------------------------------------------|
| `h` | hash + extension; the on-disk pool key               |
| `n` | display filename (user-editable via rename op)       |
| `s` | size in bytes                                        |
| `m` | sniffed MIME type at upload time                     |

The line stays inside the existing parser metadata regex (`^  (\w+): (.+)$`)
so no markdown parsing rules change.

## Body inline references

Card body markdown can embed an attachment via the `attachment:` URL scheme:

```markdown
Here's the spec: ![](attachment:a3f9...e1.png)
And the doc: [download](attachment:b7c4...12.pdf)
```

The renderer rewrites these URLs at view time:

- **Server mode**: rewritten to `/api/v1/attachments/<hash>/<encoded-name>`.
- **Local mode**: resolved via IndexedDB lookup into `blob:` URLs.

If the body references a hash not in the card's `attachments:` list, the
renderer falls back to the hash as the filename when constructing the URL.

## Mutations

Five operations on the standard mutation pipeline:

| Op                   | Effect                                                 |
|----------------------|--------------------------------------------------------|
| `add_attachments`    | Append items to a card; dedup by hash (idempotent)     |
| `remove_attachment`  | Remove an attachment by hash; missing hash is no-op    |
| `move_attachment`    | Move between two cards on the **same board**           |
| `rename_attachment`  | Update display name only; hash and bytes unchanged     |
| `reorder_attachments`| Reorder by hash list; survivors not listed appended    |

Cross-board attachment moves are orchestrated client-side as
`add_attachments` on the destination + `remove_attachment` on the source.

## HTTP API

```
POST   /api/v1/attachments              multipart/form-data, field: "file"
GET    /api/v1/attachments/{hash}/{name}
HEAD   /api/v1/attachments/{hash}/{name}
GET    /api/v1/attachments/{hash}/{name}?thumb=1   (image MIMEs only)
```

Upload returns the attachment descriptor as JSON (`{h,n,s,m}`).

Download response headers:

- `Content-Type` — sniffed at request time (not the client-supplied value).
- `Content-Disposition: inline` for `image/{png,jpeg,gif,webp}` and
  `application/pdf`. Everything else: `attachment` (browser downloads
  rather than renders, as XSS hardening).
- `X-Content-Type-Options: nosniff`
- `Cache-Control: public, max-age=31536000, immutable`
- `Content-Length: <bytes>`
- `Content-Disposition` filename uses RFC 5987 encoding for unicode names.

## Upload limits

Default 25 MB per file. Configurable via `attachments_max_bytes` in
workspace `settings.json`. Enforced server-side via `http.MaxBytesReader`;
oversized uploads return `413 Request Entity Too Large`.

No MIME allowlist — any file type is accepted. The XSS angle is handled by
universal `Content-Disposition: attachment` for non-inline-safe MIMEs.

## Garbage collection

The pool grows until you run:

```bash
liveboard gc --dir /path/to/workspace
```

The command walks every `.md` file, collects the union of referenced hashes
(card metadata + body `attachment:` URLs), and deletes any pool blob not
in that set. Idempotent. No background sweep — manual only.

Output:

```
Removed 3 unreferenced blob(s)
  a3f9...e1.pdf
  bbbb...22.png
  cccc...c3.txt
```

Cached image thumbnails (`<hash>.thumb.jpg`) and in-flight upload temp files
(`.upload-*`) are skipped by GC.

## Export

Workspace export ZIPs include referenced blobs by default:

```
GET /api/export?format=md     # raw .md files + .attachments/
GET /api/export?format=html   # rendered HTML site + .attachments/
```

Add `?attachments=false` to omit blobs (the JSON descriptors stay in the
markdown, but downloads won't resolve outside LiveBoard).

The HTML export bundles blobs but does **not** yet rewrite body
`attachment:` URLs to relative `./attachments/<hash>/<name>` paths — body
images in the exported HTML will not load until that pass is added (tracked
as a TODO in `internal/export/export.go`). The card-level `attachments:`
field still ships in the page.

## Frontend behaviour

- **Card display** (in column):
  - `card-display-mode: compact` → small badge `📎 N` next to the title.
  - Other modes → thumbnail strip below the tags (max 3 visible, then
    `+N` overflow). Image MIMEs render thumbnails; other files show a
    generic icon.
- **Card-level drop**: dropping a file onto a card in the column view
  uploads it and appends a `add_attachments` mutation. Visual feedback
  via a brief accent ring while uploading.
- **Card detail modal**: full attachment management surface — list,
  download, rename inline, remove (with confirm), drag-handle reorder,
  paste-from-clipboard for images, "+ Attach files" button, and an
  "Insert into body" action that splices `![name](attachment:hash)` at
  the textarea cursor.
- **Body markdown preview**: `attachment:` URLs in `<img>` and `<a>` are
  resolved before render via `buildAttachmentResolver` — server-mode is
  synchronous; local-mode awaits IndexedDB and produces `blob:` URLs.

## Local mode (`LocalAdapter`)

The standalone offline build (no Go server) stores blobs in IndexedDB
(`liveboard-attachments` DB, `blobs` object store, keyed by hash). Same
descriptor format, same mutations — feature parity with server mode for
single-user workflows. The browser's per-origin storage quota applies.

## MCP tools

Five descriptor-level tools exposed to MCP clients (Claude Desktop, etc.):

- `card_add_attachment_ref` — add a descriptor for an already-uploaded blob.
- `card_remove_attachment`
- `card_move_attachment` (same board)
- `card_rename_attachment`
- `card_reorder_attachments`

No binary upload over MCP — agents that need to attach a file should POST
to `/api/v1/attachments` directly, then call `card_add_attachment_ref`
with the returned descriptor.

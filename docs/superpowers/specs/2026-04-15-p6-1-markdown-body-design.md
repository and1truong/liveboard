# P6.1 — Markdown Body Rendering — Design

## Goal

Add a Preview tab to `<CardDetailModal>`'s body field that renders the textarea content as HTML via `marked` (lazy-loaded). The Edit tab remains the default; users toggle to Preview to see the rendered output. Closes the rough edge where the body field is plain text only.

**Shippable value:** card body becomes a real markdown surface, matching the HTMX UI's experience for the modal.

## Scope

**In:**
- `Edit | Preview` tab strip in `<CardDetailModal>`.
- Lazy-loaded `marked` (~12 KB gz, separate chunk) — fetched only when the user first switches to Preview.
- Tiny regex sanitizer stripping `<script>`, `on*=` event-handler attributes, and `<iframe>` tags.
- Textarea stays mounted across tab switches (CSS hidden) so content is never lost.
- Save behavior unchanged — Save reads `bodyRef.current?.value`.
- One new dep (`marked`) and a budget bump in `scripts/check-bundle-size.sh`.

**Out:**
- DOMPurify or other heavy sanitizer.
- Markdown rendering anywhere except the modal Preview tab (cards, sidebar, etc. — defer).
- Code-block syntax highlighting.
- Tables / footnotes / extended GFM features beyond marked defaults.
- Live split-pane.
- Per-board markdown configuration.

## Architecture

```
CardDetailModal
 ├─ Title input
 ├─ Tab strip:  [Edit | Preview]
 ├─ Body region:
 │    ├─ <textarea hidden={tab==='preview'}>          ← always mounted
 │    └─ if tab==='preview':
 │           previewHtml === null  → "Rendering…"
 │           else                   → <div dangerouslySetInnerHTML={{__html: previewHtml}}>
 ├─ Tags / Priority / Due / Assignee
 └─ Cancel | Save
```

Tab switch to Preview: read `bodyRef.current?.value`, call `renderMarkdown(value)`, set `previewHtml` on resolution. Tab switch back to Edit: clear `previewHtml`. Generation counter prevents racing renders.

## File structure

**New:**
- `web/renderer/default/src/components/markdownPreview.ts`
- `web/renderer/default/src/components/markdownPreview.test.ts`

**Modified:**
- `web/renderer/default/package.json` — add `marked: "^12.0.0"`.
- `web/renderer/default/src/components/CardDetailModal.tsx` — tab state, Preview branch, render trigger.
- `web/renderer/default/src/components/CardDetailModal.test.tsx` — Preview render test.
- `scripts/check-bundle-size.sh` — bump `MAX_BYTES` post-measurement.

## `markdownPreview.ts`

```ts
let markedPromise: Promise<typeof import('marked')> | null = null

export async function renderMarkdown(src: string): Promise<string> {
  markedPromise ??= import('marked')
  const { marked } = await markedPromise
  const html = (await marked.parse(src, { async: false })) as string
  return sanitize(html)
}

const SCRIPT_RE = /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi
const ON_ATTR_RE = /\son[a-z]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)/gi
const IFRAME_RE = /<\/?iframe\b[^>]*>/gi

export function sanitize(html: string): string {
  return html.replace(SCRIPT_RE, '').replace(ON_ATTR_RE, '').replace(IFRAME_RE, '')
}
```

`sanitize` is exported for unit testing.

## `<CardDetailModal>` changes

State additions:
```tsx
const [tab, setTab] = useState<'edit' | 'preview'>('edit')
const [previewHtml, setPreviewHtml] = useState<string | null>(null)
const renderGenRef = useRef(0)

const onPickPreview = (): void => {
  setTab('preview')
  setPreviewHtml(null)
  const gen = ++renderGenRef.current
  void renderMarkdown(bodyRef.current?.value ?? '').then((html) => {
    if (renderGenRef.current === gen) setPreviewHtml(html)
  })
}

const onPickEdit = (): void => {
  setTab('edit')
  setPreviewHtml(null)
}
```

Reset on every modal open (alongside the existing `key={String(open)}` reseed on `Dialog.Content`):
- `tab` defaults to `'edit'` via `useState` initializer; the `key` reseed remounts the component, so no extra reset is needed.

Body region JSX:
```tsx
<div role="tablist" className="flex gap-2 border-b border-slate-200 dark:border-slate-700">
  <button type="button" role="tab" aria-selected={tab === 'edit'}
    onClick={onPickEdit}
    className={`px-2 py-1 text-xs ${tab === 'edit' ? 'border-b-2 border-[color:var(--accent-500)] font-semibold' : 'text-slate-500'}`}>
    Edit
  </button>
  <button type="button" role="tab" aria-selected={tab === 'preview'}
    onClick={onPickPreview}
    className={`px-2 py-1 text-xs ${tab === 'preview' ? 'border-b-2 border-[color:var(--accent-500)] font-semibold' : 'text-slate-500'}`}>
    Preview
  </button>
</div>

<textarea
  ref={bodyRef}
  aria-label="card body"
  rows={6}
  defaultValue={card.body ?? ''}
  hidden={tab === 'preview'}
  className="mt-2 w-full rounded border border-slate-300 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)] dark:border-slate-600"
/>
{tab === 'preview' && (
  <div className="mt-2 min-h-32 rounded border border-slate-200 px-2 py-1 text-sm dark:border-slate-700"
       aria-label="card body preview"
       dangerouslySetInnerHTML={previewHtml === null ? undefined : { __html: previewHtml }}>
    {previewHtml === null ? 'Rendering…' : null}
  </div>
)}
```

Note: `dangerouslySetInnerHTML` and `children` are mutually exclusive in React. The cleanest pattern:
```tsx
{tab === 'preview' && (
  previewHtml === null
    ? <div className="mt-2 min-h-32 ...">Rendering…</div>
    : <div className="mt-2 min-h-32 ..." aria-label="card body preview"
           dangerouslySetInnerHTML={{ __html: previewHtml }} />
)}
```

The textarea uses `hidden` (not conditional rendering) so its uncontrolled `defaultValue` + any user typing survive the tab switch.

## Save

No changes — `submit` already reads `bodyRef.current?.value`. Tab state is render-only.

## Testing

`markdownPreview.test.ts`:

```ts
import { describe, expect, it } from 'bun:test'
import { renderMarkdown, sanitize } from './markdownPreview.js'

describe('renderMarkdown', () => {
  it('renders headings and bold', async () => {
    const html = await renderMarkdown('# Hello\n\n**bold**')
    expect(html).toContain('<h1>Hello</h1>')
    expect(html).toContain('<strong>bold</strong>')
  })

  it('strips script tags', async () => {
    const html = await renderMarkdown('<script>alert(1)</script>safe')
    expect(html).not.toContain('<script>')
    expect(html).toContain('safe')
  })

  it('strips on* attributes', async () => {
    const html = await renderMarkdown('<a href="x" onclick="bad()">link</a>')
    expect(html).not.toContain('onclick')
    expect(html).toContain('href="x"')
  })

  it('strips iframe tags', async () => {
    const html = await renderMarkdown('<iframe src="x"></iframe>after')
    expect(html).not.toContain('iframe')
    expect(html).toContain('after')
  })
})

describe('sanitize', () => {
  it('handles double-quoted attrs', () => {
    expect(sanitize('<a onmouseover="x">a</a>')).not.toContain('onmouseover')
  })
  it('handles single-quoted attrs', () => {
    expect(sanitize("<a onmouseover='x'>a</a>")).not.toContain('onmouseover')
  })
})
```

`CardDetailModal.test.tsx` (new case):

```ts
it('Preview tab renders the body as HTML', async () => {
  const { client, qc } = await setup()
  const seedWithBody = { ...seed, body: '# Hi\n\n**bold**' }
  const { findByLabelText, findByRole, findByText } = renderWithQuery(
    <ClientProvider client={client}>
      <CardDetailModal card={seedWithBody} colIdx={0} cardIdx={0}
        boardId="welcome" open={true} onOpenChange={() => {}} />
    </ClientProvider>,
    { queryClient: qc },
  )
  await findByLabelText('card body')                  // mount, tabs visible
  const previewTab = await findByRole('tab', { name: /preview/i })
  fireEvent.click(previewTab)
  await findByText((_, el) => el?.tagName === 'H1' && el.textContent === 'Hi')
  // 'bold' wrapped in <strong>
  await findByText((content) => content === 'bold')
})
```

## Bundle gate

After implementation:
1. `make renderer` and measure: `gzip -c web/renderer/default/dist/assets/*.js | wc -c`.
2. The new chunk for `marked` is a separate file under `dist/assets/`. Total grows by ~12 KB.
3. Update `MAX_BYTES` in `scripts/check-bundle-size.sh` to `previous + 15 KB` rounded to next 5 KB. Comment the date + raw value.

## Visual

- Tab strip: 28 px tall, button-ghost style; active tab has bottom border in current accent color + bold weight.
- Preview area: same width as textarea, `min-height: 8rem`, light background under light mode, dark slate-800 under dark.
- Loading text: small italic "Rendering…" centered.

No animation on tab switch.

## Risks

- **`marked.parse(src, { async: false })`**: returns `string` synchronously; the `as string` cast is for TS narrowing. If marked v13+ changes the API, pin to `^12.0.0`.
- **Regex sanitizer false negatives**: `javascript:` URLs in `href`, base64-encoded `data:` URIs, etc. are not covered. Acceptable for local-first single-user; documented above.
- **Async race when toggling tabs fast**: the `renderGenRef` counter ensures only the latest render's HTML wins.
- **Bundle inflation**: ~12 KB gz lazy chunk. Total budget grows in `MAX_BYTES`.
- **`hidden` attribute and CSS**: every modern browser respects `hidden`. Tailwind doesn't override it. No risk.

## Open questions

None blocking. Pre-decided:
- Two tabs (Edit / Preview).
- `marked` lazy-loaded; not bundled into main chunk.
- Regex sanitizer; no DOMPurify.
- Textarea stays mounted across tab switches.
- Save reads textarea content, ignores Preview state.

## Dependencies on prior work

- P4b.3: `<CardDetailModal>` exists with the body textarea.
- P4d.3: `scripts/check-bundle-size.sh` and budget mechanism.
- P5: no dependency — modal uses the existing `useBoardMutation` pipeline regardless of adapter.

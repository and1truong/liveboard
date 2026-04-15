# P6.1 — Markdown Body Rendering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an Edit / Preview tab strip to `<CardDetailModal>`'s body field. Edit (default) is the existing textarea. Preview lazy-loads `marked`, renders + sanitizes the textarea content, and shows it as HTML. Save behavior unchanged.

**Architecture:** A small `markdownPreview.ts` module wraps a lazy `import('marked')` and a regex sanitizer. `<CardDetailModal>` adds `tab` + `previewHtml` state and a generation-counter ref to handle async render races. The textarea stays mounted across tab switches via the `hidden` attribute so its value is never lost. Bundle gate's `MAX_BYTES` bumps once after measurement.

**Tech Stack:** `marked` ^12.0.0 lazy-loaded (separate Vite chunk). No other new deps.

**Spec:** `docs/superpowers/specs/2026-04-15-p6-1-markdown-body-design.md`

**Conventions:**
- New code under `web/renderer/default/src/components/`.
- Tests colocated.
- Commit prefixes: `chore(build)`, `feat(renderer)`, `test(renderer)`.
- Use bun, never npx.

---

## File structure

**New:**
- `web/renderer/default/src/components/markdownPreview.ts`
- `web/renderer/default/src/components/markdownPreview.test.ts`

**Modified:**
- `web/renderer/default/package.json` — add `marked: "^12.0.0"`.
- `web/renderer/default/src/components/CardDetailModal.tsx` — tab state + Preview branch.
- `web/renderer/default/src/components/CardDetailModal.test.tsx` — Preview render test.
- `scripts/check-bundle-size.sh` — bump `MAX_BYTES` after measurement.

---

## Task 1: Install `marked`

**Files:**
- Modify: `web/renderer/default/package.json`

- [ ] **Step 1: Add dep**

In `web/renderer/default/package.json`, under `"dependencies"` (alphabetical position — likely between `cmdk` and `react`), add:
```json
    "marked": "^12.0.0",
```

- [ ] **Step 2: Install**

```bash
cd web/renderer/default && bun install
```
Expected: clean install; bun.lock updated.

- [ ] **Step 3: Smoke import**

```bash
cd web/renderer/default && bun -e "import('marked').then(m => console.log(typeof m.marked.parse))"
```
Expected: prints `function`.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/package.json web/renderer/default/bun.lock
git commit -m "chore(build): add marked for lazy markdown preview

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `markdownPreview.ts` + tests

**Files:**
- Create: `web/renderer/default/src/components/markdownPreview.ts`
- Create: `web/renderer/default/src/components/markdownPreview.test.ts`

TDD: write the tests first, fail, then implement.

- [ ] **Step 1: Tests**

Create `web/renderer/default/src/components/markdownPreview.test.ts`:
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

- [ ] **Step 2: Run, expect fail (module missing)**

```bash
cd /Users/htruong/code/htruong/liveboard && bun test web/renderer/default/src/components/markdownPreview.test.ts
```

- [ ] **Step 3: Implement**

Create `web/renderer/default/src/components/markdownPreview.ts`:
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

- [ ] **Step 4: Run, expect 6 pass**

```bash
bun test web/renderer/default/src/components/markdownPreview.test.ts
cd web/renderer/default && bun run typecheck
```
Expected: 6 pass; typecheck clean.

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/components/markdownPreview.ts web/renderer/default/src/components/markdownPreview.test.ts
git commit -m "feat(renderer): add lazy markdown preview helper with regex sanitizer

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Edit / Preview tabs in `<CardDetailModal>`

**Files:**
- Modify: `web/renderer/default/src/components/CardDetailModal.tsx`

Add tab state + Preview branch. Textarea uses `hidden` so it stays mounted.

- [ ] **Step 1: Edit imports + state**

Read `web/renderer/default/src/components/CardDetailModal.tsx`. At the top, add to the existing `react` import (whatever's already there) `useState`. Immediately below the existing refs (around `assigneeRef`), add:
```tsx
  const [tab, setTab] = useState<'edit' | 'preview'>('edit')
  const [previewHtml, setPreviewHtml] = useState<string | null>(null)
  const renderGenRef = useRef(0)
```

Add the tab handlers near the other handlers (`submit`):
```tsx
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

Add the import for the helper near the other imports:
```tsx
import { renderMarkdown } from './markdownPreview.js'
```

- [ ] **Step 2: Replace the body label / textarea block with a tab strip + textarea + preview branch**

Find the existing block (it's the second `<label>` after Title, the one with `aria-label="card body"`). Replace it with:

```tsx
            <div>
              <div className="flex items-center justify-between">
                <span className="block text-xs font-medium text-slate-600 dark:text-slate-300">Body</span>
                <div role="tablist" className="flex gap-1 text-xs">
                  <button
                    type="button"
                    role="tab"
                    aria-selected={tab === 'edit'}
                    onClick={onPickEdit}
                    className={
                      'px-2 py-1 rounded ' +
                      (tab === 'edit'
                        ? 'border-b-2 border-[color:var(--accent-500)] font-semibold text-slate-800 dark:text-slate-100'
                        : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700')
                    }
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={tab === 'preview'}
                    onClick={onPickPreview}
                    className={
                      'px-2 py-1 rounded ' +
                      (tab === 'preview'
                        ? 'border-b-2 border-[color:var(--accent-500)] font-semibold text-slate-800 dark:text-slate-100'
                        : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700')
                    }
                  >
                    Preview
                  </button>
                </div>
              </div>
              <textarea
                ref={bodyRef}
                aria-label="card body"
                rows={6}
                defaultValue={card.body ?? ''}
                hidden={tab === 'preview'}
                className="mt-1 w-full rounded border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm outline-none focus:border-[color:var(--accent-500)]"
              />
              {tab === 'preview' && (
                previewHtml === null ? (
                  <div className="mt-1 min-h-32 rounded border border-slate-200 dark:border-slate-700 px-2 py-1 text-xs italic text-slate-400">
                    Rendering…
                  </div>
                ) : (
                  <div
                    aria-label="card body preview"
                    className="mt-1 min-h-32 rounded border border-slate-200 dark:border-slate-700 px-2 py-1 text-sm prose prose-sm dark:prose-invert max-w-none"
                    dangerouslySetInnerHTML={{ __html: previewHtml }}
                  />
                )
              )}
            </div>
```

(The original outer `<label>` becomes a `<div>` because we now have multiple controls and the textarea isn't the only focusable element.)

If `prose` Tailwind utility isn't available in this project, drop those classes — basic browser styling for `<h1>`, `<p>`, etc. is fine. Verify via the manual smoke.

- [ ] **Step 3: Typecheck + run existing tests**

```bash
cd web/renderer/default && bun test src/components/CardDetailModal.test.tsx && bun run typecheck
```
Expected: existing modal tests still pass (they don't touch the Preview tab).

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/components/CardDetailModal.tsx
git commit -m "feat(renderer): add Edit/Preview tabs to CardDetailModal body

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Preview render test in `CardDetailModal.test.tsx`

**Files:**
- Modify: `web/renderer/default/src/components/CardDetailModal.test.tsx`

- [ ] **Step 1: Add the test inside the existing `describe('CardDetailModal', ...)` block**

Append:
```tsx
  it('Preview tab renders the body as HTML', async () => {
    const { client, qc } = await setup()
    const seedWithBody = { ...seed, body: '# Hi\n\n**bold**' }
    const { findByLabelText, getByRole } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seedWithBody}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={() => {}}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    await findByLabelText('card body') // ensures form mounted
    fireEvent.click(getByRole('tab', { name: /preview/i }))
    const preview = await findByLabelText('card body preview')
    expect(preview.innerHTML).toContain('<h1>Hi</h1>')
    expect(preview.innerHTML).toContain('<strong>bold</strong>')
  })
```

If `setup` and `seed` are already defined at the top of this test file (per existing pattern), reuse them. If `seed` doesn't have a `body` field by default, that's fine — the test spreads its own `body` value.

- [ ] **Step 2: Run + commit**

```bash
cd web/renderer/default && bun test src/components/CardDetailModal.test.tsx && bun run typecheck
git add web/renderer/default/src/components/CardDetailModal.test.tsx
git commit -m "test(renderer): cover CardDetailModal Preview tab rendering

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

If the test fails because `marked` import resolves asynchronously and the `findByLabelText('card body preview')` poll doesn't catch it in time, increase the helper's timeout: `await findByLabelText('card body preview', {}, { timeout: 3000 })`. Don't commit on red.

---

## Task 5: Build, measure, bump bundle budget

**Files:**
- Modify: `scripts/check-bundle-size.sh`

- [ ] **Step 1: Build**

```bash
cd /Users/htruong/code/htruong/liveboard && make renderer
```
Note: `make renderer` runs `bundle-check` after build. It will likely FAIL the gate now because the lazy-marked chunk pushes total over the budget. That's expected.

- [ ] **Step 2: Measure raw size**

```bash
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Record the new total. Example: previously 140 KB → now 152 KB.

- [ ] **Step 3: Pick new `MAX_BYTES`**

Take the new measured value, add ~5 KB headroom, round up to the next 5 KB boundary. Example: measured 152000 → +5120 = 157120 → round to 158720 (155 KB).

- [ ] **Step 4: Update the script**

Edit `scripts/check-bundle-size.sh`. Replace the current `MAX_BYTES="${MAX_BYTES:-...}"` line with the new value, and add a comment with today's date and the measured number:
```sh
# Measured 2026-04-15: <measured> bytes gzipped (added marked lazy chunk in P6.1).
# Budget = measured + ~5 KB headroom, rounded to next 5 KB.
MAX_BYTES="${MAX_BYTES:-<new value>}"
```

- [ ] **Step 5: Re-run gate**

```bash
make renderer
```
Expected: build succeeds, `bundle-check` reports the new total under the new budget, exit 0.

- [ ] **Step 6: Commit**

```bash
git add scripts/check-bundle-size.sh
git commit -m "chore(build): bump bundle budget for marked lazy chunk

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Manual browser smoke

Not a code change.

- [ ] **Step 1: Build + serve**

```bash
make adapter-test
```

- [ ] **Step 2: At <http://localhost:7070/app/> verify**

1. Open any card's detail modal (click the tag strip).
2. Body field shows tabs: Edit (active), Preview.
3. Type some markdown into the textarea: `# Hello\n\n**bold** *italic*\n\n- list item`.
4. Click Preview → "Rendering…" briefly, then the rendered HTML appears (h1, bold, italic, list).
5. Click Edit → textarea returns with content intact.
6. Switch back to Preview → re-renders.
7. Save → modal closes; the body persists. Reopen → body still has the markdown source.
8. Body with `<script>alert(1)</script>foo` → Preview shows `foo`, no alert.
9. Devtools Network: confirm `marked` chunk loads only after first Preview click.
10. `?renderer=stub` still loads.

- [ ] **Step 3: Report.** Capture failures with step + expected vs actual.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `marked` ^12.0.0 dep added | 1 |
| `markdownPreview.ts` + lazy loader + sanitizer | 2 |
| Sanitizer unit tests (script, on*, iframe) | 2 |
| Edit / Preview tabs in modal | 3 |
| Textarea stays mounted via `hidden` | 3 |
| Generation counter prevents render races | 3 |
| Preview renders markdown as HTML test | 4 |
| Bundle gate bump after measurement | 5 |
| Manual smoke covers happy path + sanitization + lazy load | 6 |

## Notes for implementer

1. **`marked.parse(src, { async: false })`** returns `string` synchronously in v12. The `as string` cast is a TS narrowing — if TS rejects, drop the `await` and the `Promise<string>` return type stays correct because `markedPromise` is awaited.
2. **`prose` Tailwind utility**: if the project doesn't have `@tailwindcss/typography`, drop the `prose` classes. Browser default styling for `<h1>`, `<p>`, etc. is sufficient. Don't add the typography plugin solely for this — it's heavyweight.
3. **`hidden` not `display:none` via Tailwind**: the HTML `hidden` attribute is the cleanest way; Tailwind doesn't override it. The textarea retains its uncontrolled value across tab switches.
4. **Lazy chunk emission**: Vite emits `dist/assets/marked-<hash>.js` as a separate chunk on the dynamic import. The `bundle-check` script's `gzip -c web/renderer/default/dist/assets/*.js` glob sums all chunks — both the main and the lazy one count. That's what the budget measures.
5. **Re-render on every Preview click**: the helper recomputes from the current textarea value each time. No memoization — the marked render is fast and the sanitize is cheap.
6. **No commit amending** — forward-only commits.

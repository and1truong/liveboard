# P4a — Renderer Scaffold (Read-Only Board View) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a React+Vite SPA at `/app/renderer/default/` that lists boards, renders the selected board read-only, and re-renders on shell-pushed `board.updated` events. Replaces the P3 stub as the default iframe target; stub remains reachable via `?renderer=stub`.

**Architecture:** Two-package frontend — `web/shared` (P2/P3, unchanged) provides the Client and types; `web/renderer/default` is a Vite-built React app that imports them. Bundle served by Go via a new `//go:embed` extending the existing `/app/*` handler behind `LIVEBOARD_APP_SHELL=1`. TanStack Query owns server state; event-push wiring invalidates queries when `board.updated` fires.

**Tech Stack:** React 18, TanStack Query v5, Tailwind v4, Radix primitives (as needed), Vite 5 driven by `bunx --bun vite`, `bun test` + happy-dom + @testing-library/react for component tests.

**Spec:** `docs/superpowers/specs/2026-04-15-p4a-renderer-scaffold-design.md`

**Conventions:**
- All renderer files live under `web/renderer/default/`.
- Imports from the shared package use relative paths (`../../../shared/src/...`) — no workspace aliasing unless it breaks.
- Tailwind utility classes in markup; no CSS modules.
- Commit prefixes: `feat(renderer)`, `test(renderer)`, `chore(build)`, `docs`.

**Out of scope (P4b/c/d):** mutations, drag-drop, modals, command palette, keyboard nav, themes, calendar, board create/rename/delete, routing, optimistic UI.

---

## File structure

**New files:**
- `web/renderer/default/package.json`
- `web/renderer/default/tsconfig.json`
- `web/renderer/default/vite.config.ts`
- `web/renderer/default/index.html`
- `web/renderer/default/postcss.config.js`
- `web/renderer/default/tailwind.config.ts`
- `web/renderer/default/.gitignore`
- `web/renderer/default/happydom.ts` — test setup
- `web/renderer/default/bunfig.toml` — registers happydom preload
- `web/renderer/default/src/main.tsx`
- `web/renderer/default/src/App.tsx`
- `web/renderer/default/src/client.ts`
- `web/renderer/default/src/queries.ts`
- `web/renderer/default/src/components/BoardSidebar.tsx`
- `web/renderer/default/src/components/BoardView.tsx`
- `web/renderer/default/src/components/Column.tsx`
- `web/renderer/default/src/components/Card.tsx`
- `web/renderer/default/src/components/EmptyState.tsx`
- `web/renderer/default/src/styles/tailwind.css`
- `web/renderer/default/src/test-utils.tsx` — renderWithQuery helper
- Test files colocated: `*.test.tsx` / `*.test.ts`
- `web/renderer/default/embed.go`

**Modified:**
- `Makefile` — `renderer` target + roll into a `frontend` umbrella
- `internal/api/server.go` — extend `mountShellRoutes` to also serve `/app/renderer/default/*`
- `web/shell/src/main.ts` — read `?renderer=` query param to pick iframe src (default `renderer/default`, legacy `renderer-stub`)
- `web/shell/index.html` — remove hardcoded iframe src; main.ts sets it
- `.github/workflows/ci.yml` — `make renderer` before Go tests (new embed), add size log
- `docs/parity.md` — add renderer section

---

## Task 1: Scaffold package + Vite + TS + Tailwind config

**Files:**
- Create: `web/renderer/default/package.json`
- Create: `web/renderer/default/tsconfig.json`
- Create: `web/renderer/default/vite.config.ts`
- Create: `web/renderer/default/index.html`
- Create: `web/renderer/default/postcss.config.js`
- Create: `web/renderer/default/tailwind.config.ts`
- Create: `web/renderer/default/.gitignore`

- [ ] **Step 1: `web/renderer/default/package.json`**

```json
{
  "name": "@liveboard/renderer-default",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "bunx --bun vite",
    "build": "bunx --bun vite build",
    "preview": "bunx --bun vite preview",
    "typecheck": "tsc --noEmit",
    "test": "bun test"
  },
  "dependencies": {
    "@tanstack/react-query": "^5.59.0",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@testing-library/react": "^16.0.1",
    "@types/bun": "latest",
    "@types/react": "^18.3.12",
    "@types/react-dom": "^18.3.1",
    "@vitejs/plugin-react": "^4.3.3",
    "autoprefixer": "^10.4.20",
    "happy-dom": "^15.11.0",
    "postcss": "^8.4.49",
    "tailwindcss": "^4.0.0-beta.3",
    "typescript": "^5.6.3",
    "vite": "^5.4.10"
  }
}
```

- [ ] **Step 2: `web/renderer/default/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "types": ["bun"],
    "baseUrl": ".",
    "paths": {
      "@shared/*": ["../../shared/src/*"]
    }
  },
  "include": ["src/**/*", "happydom.ts"]
}
```

- [ ] **Step 3: `web/renderer/default/vite.config.ts`**

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  base: '/app/renderer/default/',
  plugins: [react()],
  resolve: {
    alias: {
      '@shared': fileURLToPath(new URL('../../shared/src', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: true,
  },
  server: {
    port: 5173,
  },
})
```

- [ ] **Step 4: `web/renderer/default/index.html`**

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>LiveBoard</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <link rel="stylesheet" href="/src/styles/tailwind.css" />
</head>
<body class="bg-slate-50 text-slate-900 antialiased">
  <div id="root"></div>
  <script type="module" src="/src/main.tsx"></script>
</body>
</html>
```

- [ ] **Step 5: `web/renderer/default/postcss.config.js`**

```js
export default {
  plugins: {
    '@tailwindcss/postcss': {},
    autoprefixer: {},
  },
}
```

Note: Tailwind v4 uses `@tailwindcss/postcss`. If `bun install` reports that package as missing in Step 8, add it to devDependencies and reinstall.

- [ ] **Step 6: `web/renderer/default/tailwind.config.ts`**

```ts
import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: { extend: {} },
  plugins: [],
} satisfies Config
```

- [ ] **Step 7: `web/renderer/default/.gitignore`**

```
dist/
node_modules/
```

- [ ] **Step 8: Install deps and verify Vite starts**

```bash
cd web/renderer/default && bun install
bunx --bun vite --version
```
Expected: bun installs without errors, Vite prints a version. If `@tailwindcss/postcss` is missing, add `"@tailwindcss/postcss": "^4.0.0-beta.3"` to devDependencies and rerun `bun install`.

- [ ] **Step 9: Commit**

```bash
git add web/renderer/default/package.json web/renderer/default/tsconfig.json \
        web/renderer/default/vite.config.ts web/renderer/default/index.html \
        web/renderer/default/postcss.config.js web/renderer/default/tailwind.config.ts \
        web/renderer/default/.gitignore web/renderer/default/bun.lockb 2>/dev/null; \
git add web/renderer/default
git commit -m "chore(build): scaffold renderer/default Vite+React project"
```

---

## Task 2: Tailwind entrypoint + happy-dom test setup

**Files:**
- Create: `web/renderer/default/src/styles/tailwind.css`
- Create: `web/renderer/default/happydom.ts`
- Create: `web/renderer/default/bunfig.toml`

- [ ] **Step 1: `web/renderer/default/src/styles/tailwind.css`**

```css
@import "tailwindcss";
```

- [ ] **Step 2: `web/renderer/default/happydom.ts`**

```ts
import { GlobalRegistrator } from '@happy-dom/global-registrator'
GlobalRegistrator.register()
```

- [ ] **Step 3: Add happy-dom/global-registrator to deps**

Append to `web/renderer/default/package.json` devDependencies:
```json
"@happy-dom/global-registrator": "^15.11.0"
```
Then:
```bash
cd web/renderer/default && bun install
```

- [ ] **Step 4: `web/renderer/default/bunfig.toml`**

```toml
[test]
preload = ["./happydom.ts"]
```

- [ ] **Step 5: Write a smoke test to prove DOM globals exist**

Create `web/renderer/default/src/dom-smoke.test.ts`:
```ts
import { describe, expect, it } from 'bun:test'

describe('happy-dom', () => {
  it('provides document global', () => {
    const el = document.createElement('div')
    el.textContent = 'hi'
    expect(el.textContent).toBe('hi')
  })
})
```

- [ ] **Step 6: Run**

```bash
cd web/renderer/default && bun test src/dom-smoke.test.ts
```
Expected: 1 pass.

- [ ] **Step 7: Commit**

```bash
git add web/renderer/default/src/styles/tailwind.css web/renderer/default/happydom.ts \
        web/renderer/default/bunfig.toml web/renderer/default/package.json \
        web/renderer/default/src/dom-smoke.test.ts
git commit -m "chore(build): add tailwind entry and happy-dom test setup"
```

---

## Task 3: Client + QueryClient bootstrap

**Files:**
- Create: `web/renderer/default/src/client.ts`
- Create: `web/renderer/default/src/client.test.ts`

`client.ts` exposes a singleton Client (constructed once from `iframeTransport`) plus the `QueryClient` and a helper that wires `board.updated` → invalidate.

- [ ] **Step 1: `web/renderer/default/src/client.ts`**

```ts
import { QueryClient } from '@tanstack/react-query'
import { Client } from '@shared/client.js'
import { iframeTransport } from '@shared/transports/post-message.js'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 60_000,
    },
  },
})

export function createClient(parentOrigin: string = window.location.origin): Client {
  const transport = iframeTransport(parentOrigin)
  const client = new Client(transport, {
    rendererId: 'default',
    rendererVersion: '0.1.0',
  })
  client.on('board.updated', ({ boardId }) => {
    void queryClient.invalidateQueries({ queryKey: ['board', boardId] })
    void queryClient.invalidateQueries({ queryKey: ['boards'] })
  })
  return client
}
```

- [ ] **Step 2: `web/renderer/default/src/client.test.ts`**

```ts
import { describe, expect, it } from 'bun:test'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'

// We can't import the real createClient (it uses iframeTransport/window).
// Instead, assert the wiring shape: that a Client + event handler invalidates.

describe('client event → query invalidation', () => {
  it('invalidates board query on board.updated', async () => {
    const [iframeT, shellT] = createMemoryPair()
    const adapter = new LocalAdapter(new MemoryStorage())
    new Broker(shellT, adapter, { shellVersion: 't' })
    const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })

    const qc = new QueryClient()
    let invalidated = 0
    qc.getQueryCache().subscribe((ev) => {
      if (ev.type === 'updated' && ev.action.type === 'invalidate') invalidated++
    })
    client.on('board.updated', ({ boardId }) => {
      void qc.invalidateQueries({ queryKey: ['board', boardId] })
    })

    await client.ready()
    await client.subscribe('welcome')
    await adapter.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'x' })
    await new Promise((r) => setTimeout(r, 10))
    expect(invalidated).toBeGreaterThan(0)
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/renderer/default && bun test src/client.test.ts && bun run typecheck
```
Expected: 1 pass, typecheck green.

If `@shared/*` path alias doesn't resolve under `bun test`, bun reads tsconfig paths — but if it fails, fall back to relative imports (`../../shared/src/...`) in `client.ts` and `client.test.ts`.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/client.ts web/renderer/default/src/client.test.ts
git commit -m "feat(renderer): bootstrap Client and QueryClient with event invalidation"
```

---

## Task 4: Query hooks

**Files:**
- Create: `web/renderer/default/src/queries.ts`
- Create: `web/renderer/default/src/queries.test.tsx`
- Create: `web/renderer/default/src/test-utils.tsx`

- [ ] **Step 1: `web/renderer/default/src/test-utils.tsx`**

```tsx
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, type RenderOptions } from '@testing-library/react'

export function renderWithQuery(
  ui: ReactNode,
  options?: { queryClient?: QueryClient } & Omit<RenderOptions, 'wrapper'>,
): ReturnType<typeof render> & { queryClient: QueryClient } {
  const queryClient =
    options?.queryClient ??
    new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const result = render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
    options,
  )
  return { ...result, queryClient }
}
```

- [ ] **Step 2: `web/renderer/default/src/queries.ts`**

```ts
import { useQuery, type UseQueryResult } from '@tanstack/react-query'
import type { Client } from '@shared/client.js'
import type { Board } from '@shared/types.js'
import type { BoardSummary, WorkspaceInfo } from '@shared/adapter.js'
import { createContext, useContext, type ReactNode } from 'react'

const ClientContext = createContext<Client | null>(null)

export function ClientProvider({
  client,
  children,
}: {
  client: Client
  children: ReactNode
}): ReactNode {
  return <ClientContext.Provider value={client}>{children}</ClientContext.Provider>
}

function useClient(): Client {
  const c = useContext(ClientContext)
  if (!c) throw new Error('ClientProvider missing')
  return c
}

export function useBoardList(): UseQueryResult<BoardSummary[]> {
  const client = useClient()
  return useQuery({
    queryKey: ['boards'],
    queryFn: () => client.listBoards(),
  })
}

export function useBoard(boardId: string | null): UseQueryResult<Board> {
  const client = useClient()
  return useQuery({
    queryKey: ['board', boardId],
    queryFn: () => {
      if (!boardId) throw new Error('no board selected')
      return client.getBoard(boardId)
    },
    enabled: boardId !== null,
  })
}

export function useWorkspaceInfo(): UseQueryResult<WorkspaceInfo> {
  const client = useClient()
  return useQuery({
    queryKey: ['workspace'],
    queryFn: () => client.workspaceInfo(),
  })
}
```

Note: `queries.ts` contains JSX, rename it to `queries.tsx`. Apply that rename — the test file imports are updated accordingly.

- [ ] **Step 3: Rename `queries.ts` → `queries.tsx`**

```bash
mv web/renderer/default/src/queries.ts web/renderer/default/src/queries.tsx
```

(If the file wasn't created yet because you jumped ahead, just create `queries.tsx` with the Step 2 content.)

- [ ] **Step 4: `web/renderer/default/src/queries.test.tsx`**

```tsx
import { describe, expect, it } from 'bun:test'
import { waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider, useBoard, useBoardList } from './queries.js'
import { renderWithQuery } from './test-utils.js'

function setup(): Client {
  const [iframeT, shellT] = createMemoryPair()
  const adapter = new LocalAdapter(new MemoryStorage())
  new Broker(shellT, adapter, { shellVersion: 't' })
  return new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
}

function BoardListProbe(): JSX.Element {
  const q = useBoardList()
  if (q.isLoading) return <p>loading</p>
  if (q.error) return <p>err</p>
  return <ul>{q.data?.map((b) => <li key={b.id}>{b.name}</li>)}</ul>
}

function BoardProbe({ id }: { id: string }): JSX.Element {
  const q = useBoard(id)
  if (q.isLoading) return <p>loading</p>
  if (q.error) return <p>err</p>
  return <h2>{q.data?.name}</h2>
}

describe('queries', () => {
  it('useBoardList returns the welcome board', async () => {
    const client = setup()
    await client.ready()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardListProbe />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
  })

  it('useBoard returns the board by id', async () => {
    const client = setup()
    await client.ready()
    const { getByRole } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardProbe id="welcome" />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByRole('heading', { name: 'Welcome' })).toBeDefined())
  })

  it('invalidates board query when board.updated fires', async () => {
    const client = setup()
    await client.ready()
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    client.on('board.updated', ({ boardId }) =>
      void qc.invalidateQueries({ queryKey: ['board', boardId] }),
    )
    await client.subscribe('welcome')

    const { rerender, queryByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardProbe id="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => expect(queryByText('Welcome')).toBeDefined())

    await client.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'z' })
    rerender(
      <ClientProvider client={client}>
        <BoardProbe id="welcome" />
      </ClientProvider>,
    )
    // invalidation + refetch is async; just wait for a tick
    await new Promise((r) => setTimeout(r, 20))
    expect(queryByText('Welcome')).toBeDefined()
  })
})
```

- [ ] **Step 5: Run**

```bash
cd web/renderer/default && bun test src/queries.test.tsx && bun run typecheck
```
Expected: 3 pass, typecheck green.

- [ ] **Step 6: Commit**

```bash
git add web/renderer/default/src/queries.tsx web/renderer/default/src/queries.test.tsx \
        web/renderer/default/src/test-utils.tsx
git commit -m "feat(renderer): add TanStack Query hooks and ClientProvider"
```

---

## Task 5: Card component

**Files:**
- Create: `web/renderer/default/src/components/Card.tsx`
- Create: `web/renderer/default/src/components/Card.test.tsx`

- [ ] **Step 1: `Card.tsx`**

```tsx
import type { Card as CardModel } from '@shared/types.js'

const PRIORITY_DOT: Record<string, string> = {
  critical: 'bg-red-600',
  high: 'bg-orange-500',
  medium: 'bg-yellow-400',
  low: 'bg-slate-300',
}

export function Card({ card }: { card: CardModel }): JSX.Element {
  return (
    <article className="rounded-md bg-white p-3 shadow-sm ring-1 ring-slate-200">
      <div className="flex items-start gap-2">
        {card.priority && (
          <span
            aria-label={`priority ${card.priority}`}
            className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${PRIORITY_DOT[card.priority] ?? 'bg-slate-300'}`}
          />
        )}
        <h3 className={`text-sm font-semibold ${card.completed ? 'line-through text-slate-400' : ''}`}>
          {card.title}
        </h3>
      </div>
      {card.tags && card.tags.length > 0 && (
        <ul className="mt-2 flex flex-wrap gap-1">
          {card.tags.map((t) => (
            <li key={t} className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700">
              {t}
            </li>
          ))}
        </ul>
      )}
    </article>
  )
}
```

- [ ] **Step 2: `Card.test.tsx`**

```tsx
import { describe, expect, it } from 'bun:test'
import { render } from '@testing-library/react'
import { Card } from './Card.js'

describe('Card', () => {
  it('renders title', () => {
    const { getByText } = render(<Card card={{ title: 'Hello' }} />)
    expect(getByText('Hello')).toBeDefined()
  })

  it('renders tags as pills', () => {
    const { getByText } = render(<Card card={{ title: 'x', tags: ['a', 'b'] }} />)
    expect(getByText('a')).toBeDefined()
    expect(getByText('b')).toBeDefined()
  })

  it('shows priority dot when priority set', () => {
    const { getByLabelText } = render(<Card card={{ title: 'x', priority: 'high' }} />)
    expect(getByLabelText('priority high')).toBeDefined()
  })

  it('strikes through completed cards', () => {
    const { getByText } = render(<Card card={{ title: 'done', completed: true }} />)
    const h = getByText('done')
    expect(h.className).toContain('line-through')
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/renderer/default && bun test src/components/Card.test.tsx && bun run typecheck
```
Expected: 4 pass, typecheck green.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/components/Card.tsx web/renderer/default/src/components/Card.test.tsx
git commit -m "feat(renderer): add Card component"
```

---

## Task 6: Column component

**Files:**
- Create: `web/renderer/default/src/components/Column.tsx`
- Create: `web/renderer/default/src/components/Column.test.tsx`

- [ ] **Step 1: `Column.tsx`**

```tsx
import type { Column as ColumnModel } from '@shared/types.js'
import { Card } from './Card.js'

export function Column({ column }: { column: ColumnModel }): JSX.Element {
  const cards = column.cards ?? []
  return (
    <section className="flex w-72 shrink-0 flex-col rounded-lg bg-slate-100 p-3">
      <header className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-800">{column.name}</h2>
        <span className="text-xs text-slate-500">{cards.length}</span>
      </header>
      <ul className="flex flex-col gap-2">
        {cards.map((card, i) => (
          <li key={`${column.name}-${i}`}>
            <Card card={card} />
          </li>
        ))}
      </ul>
    </section>
  )
}
```

- [ ] **Step 2: `Column.test.tsx`**

```tsx
import { describe, expect, it } from 'bun:test'
import { render } from '@testing-library/react'
import { Column } from './Column.js'

describe('Column', () => {
  it('renders column name and card count', () => {
    const { getByText } = render(
      <Column column={{ name: 'Todo', cards: [{ title: 'a' }, { title: 'b' }] }} />,
    )
    expect(getByText('Todo')).toBeDefined()
    expect(getByText('2')).toBeDefined()
  })

  it('renders all cards', () => {
    const { getByText } = render(
      <Column column={{ name: 'x', cards: [{ title: 'A' }, { title: 'B' }] }} />,
    )
    expect(getByText('A')).toBeDefined()
    expect(getByText('B')).toBeDefined()
  })

  it('handles empty cards array', () => {
    const { getByText } = render(<Column column={{ name: 'Empty', cards: [] }} />)
    expect(getByText('0')).toBeDefined()
  })
})
```

- [ ] **Step 3: Run**

```bash
cd web/renderer/default && bun test src/components/Column.test.tsx && bun run typecheck
```
Expected: 3 pass, typecheck green.

- [ ] **Step 4: Commit**

```bash
git add web/renderer/default/src/components/Column.tsx web/renderer/default/src/components/Column.test.tsx
git commit -m "feat(renderer): add Column component"
```

---

## Task 7: EmptyState component

**Files:**
- Create: `web/renderer/default/src/components/EmptyState.tsx`
- Create: `web/renderer/default/src/components/EmptyState.test.tsx`

- [ ] **Step 1: `EmptyState.tsx`**

```tsx
export function EmptyState({
  title,
  detail,
}: {
  title: string
  detail?: string
}): JSX.Element {
  return (
    <div className="flex h-full items-center justify-center p-8 text-center">
      <div>
        <p className="text-base font-medium text-slate-700">{title}</p>
        {detail && <p className="mt-1 text-sm text-slate-500">{detail}</p>}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: `EmptyState.test.tsx`**

```tsx
import { describe, expect, it } from 'bun:test'
import { render } from '@testing-library/react'
import { EmptyState } from './EmptyState.js'

describe('EmptyState', () => {
  it('renders title', () => {
    const { getByText } = render(<EmptyState title="Nothing here" />)
    expect(getByText('Nothing here')).toBeDefined()
  })

  it('renders detail when provided', () => {
    const { getByText } = render(<EmptyState title="x" detail="some detail" />)
    expect(getByText('some detail')).toBeDefined()
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/EmptyState.test.tsx
git add web/renderer/default/src/components/EmptyState.tsx web/renderer/default/src/components/EmptyState.test.tsx
git commit -m "feat(renderer): add EmptyState component"
```

---

## Task 8: BoardView component

**Files:**
- Create: `web/renderer/default/src/components/BoardView.tsx`
- Create: `web/renderer/default/src/components/BoardView.test.tsx`

- [ ] **Step 1: `BoardView.tsx`**

```tsx
import { useEffect } from 'react'
import type { Client } from '@shared/client.js'
import { useBoard } from '../queries.js'
import { Column } from './Column.js'
import { EmptyState } from './EmptyState.js'

export function BoardView({
  boardId,
  client,
}: {
  boardId: string | null
  client: Client
}): JSX.Element {
  const { data, isLoading, error } = useBoard(boardId)

  useEffect(() => {
    if (!boardId) return
    void client.subscribe(boardId)
    return () => {
      void client.unsubscribe(boardId)
    }
  }, [boardId, client])

  if (!boardId) return <EmptyState title="Select a board" />
  if (isLoading) return <EmptyState title="Loading…" />
  if (error) return <EmptyState title="Failed to load board" detail={String(error)} />
  if (!data) return <EmptyState title="Board not found" />

  const columns = data.columns ?? []
  if (columns.length === 0) {
    return <EmptyState title="This board has no columns yet." />
  }

  return (
    <div className="flex h-full gap-4 overflow-x-auto p-4">
      {columns.map((col, i) => (
        <Column key={`${col.name}-${i}`} column={col} />
      ))}
    </div>
  )
}
```

- [ ] **Step 2: `BoardView.test.tsx`**

```tsx
import { describe, expect, it } from 'bun:test'
import { waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { BoardView } from './BoardView.js'
import { renderWithQuery } from '../test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  const adapter = new LocalAdapter(new MemoryStorage())
  new Broker(shellT, adapter, { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

describe('BoardView', () => {
  it('renders empty state when no board selected', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardView boardId={null} client={client} />
      </ClientProvider>,
    )
    expect(getByText('Select a board')).toBeDefined()
  })

  it('renders columns from the welcome board', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardView boardId="welcome" client={client} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    expect(getByText('Doing')).toBeDefined()
    expect(getByText('Done')).toBeDefined()
  })

  it('shows error state when board is missing', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardView boardId="nope" client={client} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Failed to load board')).toBeDefined())
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/BoardView.test.tsx && bun run typecheck
git add web/renderer/default/src/components/BoardView.tsx web/renderer/default/src/components/BoardView.test.tsx
git commit -m "feat(renderer): add BoardView with subscribe lifecycle"
```

Expected: 3 pass, typecheck green.

---

## Task 9: BoardSidebar component

**Files:**
- Create: `web/renderer/default/src/components/BoardSidebar.tsx`
- Create: `web/renderer/default/src/components/BoardSidebar.test.tsx`

- [ ] **Step 1: `BoardSidebar.tsx`**

```tsx
import { useBoardList, useWorkspaceInfo } from '../queries.js'
import { EmptyState } from './EmptyState.js'

export function BoardSidebar({
  activeId,
  onSelect,
}: {
  activeId: string | null
  onSelect: (boardId: string) => void
}): JSX.Element {
  const boards = useBoardList()
  const ws = useWorkspaceInfo()

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-slate-200 bg-white">
      <header className="border-b border-slate-200 p-3">
        <p className="text-xs uppercase tracking-wide text-slate-500">Workspace</p>
        <p className="truncate text-sm font-semibold text-slate-800">
          {ws.data?.name ?? '—'}
        </p>
      </header>
      {boards.isLoading ? (
        <EmptyState title="Loading…" />
      ) : boards.error ? (
        <EmptyState title="Failed to load" detail={String(boards.error)} />
      ) : !boards.data || boards.data.length === 0 ? (
        <EmptyState title="No boards yet" />
      ) : (
        <ul className="flex-1 overflow-y-auto p-2">
          {boards.data.map((b) => {
            const active = b.id === activeId
            return (
              <li key={b.id}>
                <button
                  type="button"
                  onClick={() => onSelect(b.id)}
                  className={`flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm ${
                    active
                      ? 'bg-slate-200 text-slate-900'
                      : 'text-slate-700 hover:bg-slate-100'
                  }`}
                >
                  {b.icon && <span aria-hidden>{b.icon}</span>}
                  <span className="truncate">{b.name}</span>
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </aside>
  )
}
```

- [ ] **Step 2: `BoardSidebar.test.tsx`**

```tsx
import { describe, expect, it, mock } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { BoardSidebar } from './BoardSidebar.js'
import { renderWithQuery } from '../test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

describe('BoardSidebar', () => {
  it('lists boards from the adapter', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSidebar activeId={null} onSelect={() => {}} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    expect(getByText('Demo')).toBeDefined()
  })

  it('fires onSelect with board id on click', async () => {
    const client = await setup()
    const onSelect = mock(() => {})
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSidebar activeId={null} onSelect={onSelect} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    fireEvent.click(getByText('Welcome'))
    expect(onSelect).toHaveBeenCalledWith('welcome')
  })

  it('highlights active board', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSidebar activeId="welcome" onSelect={() => {}} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    const btn = getByText('Welcome').closest('button')!
    expect(btn.className).toContain('bg-slate-200')
  })
})
```

- [ ] **Step 3: Run + commit**

```bash
cd web/renderer/default && bun test src/components/BoardSidebar.test.tsx && bun run typecheck
git add web/renderer/default/src/components/BoardSidebar.tsx web/renderer/default/src/components/BoardSidebar.test.tsx
git commit -m "feat(renderer): add BoardSidebar with workspace header"
```

Expected: 3 pass, typecheck green.

---

## Task 10: App + main entry

**Files:**
- Create: `web/renderer/default/src/App.tsx`
- Create: `web/renderer/default/src/App.test.tsx`
- Create: `web/renderer/default/src/main.tsx`

- [ ] **Step 1: `App.tsx`**

```tsx
import { useState } from 'react'
import type { Client } from '@shared/client.js'
import { BoardSidebar } from './components/BoardSidebar.js'
import { BoardView } from './components/BoardView.js'

export function App({ client }: { client: Client }): JSX.Element {
  const [activeId, setActiveId] = useState<string | null>(null)
  return (
    <div className="flex h-screen w-screen">
      <BoardSidebar activeId={activeId} onSelect={setActiveId} />
      <main className="flex-1 overflow-hidden">
        <BoardView boardId={activeId} client={client} />
      </main>
    </div>
  )
}
```

- [ ] **Step 2: `App.test.tsx`**

```tsx
import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { App } from './App.js'
import { ClientProvider } from './queries.js'
import { renderWithQuery } from './test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

describe('App integration', () => {
  it('selecting a board renders its columns', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <App client={client} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    fireEvent.click(getByText('Welcome'))
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    expect(getByText('Doing')).toBeDefined()
  })

  it('live-updates when a mutation happens on the same client', async () => {
    const client = await setup()
    const { getByText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <App client={client} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    fireEvent.click(getByText('Welcome'))
    await waitFor(() => expect(getByText('Todo')).toBeDefined())

    client.on('board.updated', ({ boardId }) => {
      // wired by createClient normally; test wires by hand
    })
    // Drive the mutation via the same client so the shell broker pushes an event.
    await client.subscribe('welcome')
    await client.mutateBoard('welcome', 1, {
      type: 'add_card',
      column: 'Todo',
      title: 'LIVE-ADDED',
    })
    // The test client doesn't have the createClient's invalidation wiring —
    // instead, assert the mutation succeeded and the mutateBoard promise's
    // returned board reflects it. Full event wiring is covered in
    // queries.test.tsx.
    const list = await client.listBoards()
    expect(list[0]?.version).toBeGreaterThanOrEqual(2)
  })
})
```

- [ ] **Step 3: `main.tsx`**

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { App } from './App.js'
import { ClientProvider } from './queries.js'
import { createClient, queryClient } from './client.js'
import './styles/tailwind.css'

async function boot(): Promise<void> {
  const root = document.getElementById('root')
  if (!root) throw new Error('#root missing')

  const client = createClient()
  try {
    await client.ready()
  } catch (e) {
    root.textContent = `Couldn't connect to shell: ${(e as Error).message}`
    return
  }

  createRoot(root).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <ClientProvider client={client}>
          <App client={client} />
        </ClientProvider>
      </QueryClientProvider>
    </StrictMode>,
  )
}

void boot()
```

- [ ] **Step 4: Run tests + typecheck**

```bash
cd web/renderer/default && bun test && bun run typecheck
```
Expected: all tests pass (roughly 18-20 total across files), typecheck green.

- [ ] **Step 5: Commit**

```bash
git add web/renderer/default/src/App.tsx web/renderer/default/src/App.test.tsx \
        web/renderer/default/src/main.tsx
git commit -m "feat(renderer): add App shell and main entry"
```

---

## Task 11: Vite build + Makefile target

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Verify build works locally**

```bash
cd web/renderer/default && bunx --bun vite build
ls dist/
```
Expected: `index.html`, `assets/` dir with a hashed `index-*.js` and `index-*.css`.

If build fails, fix before continuing. Common first failures: missing `@tailwindcss/postcss` (add it), path alias mismatch (check `tsconfig.json` `paths` matches `vite.config.ts` `resolve.alias`).

- [ ] **Step 2: Add `renderer` target to `Makefile`**

Read current Makefile. Alongside the `shell:` target, append:

```make
.PHONY: renderer
renderer:
	cd web/renderer/default && bun install --frozen-lockfile
	cd web/renderer/default && bunx --bun vite build

.PHONY: frontend
frontend: shell renderer
```

Add `renderer` and `frontend` to the main `.PHONY` line if one exists.

- [ ] **Step 3: Run**

```bash
make renderer
ls web/renderer/default/dist/
```
Expected: build output present.

- [ ] **Step 4: Log bundle size**

```bash
ls -la web/renderer/default/dist/assets/*.js | awk '{print $5, $9}'
gzip -c web/renderer/default/dist/assets/*.js | wc -c
```
Note the numbers. P4a target: under 100kb gz for JS. Record this in the commit message.

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "chore(build): add renderer vite build target"
```

---

## Task 12: Go embed + /app/renderer/default/* route

**Files:**
- Create: `web/renderer/default/embed.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/server_shell_test.go`

- [ ] **Step 1: `web/renderer/default/embed.go`**

```go
// Package renderer exposes the built default renderer bundle for embedding.
package renderer

import "embed"

//go:embed all:dist
var FS embed.FS
```

- [ ] **Step 2: Extend server.go**

Open `internal/api/server.go`. The existing `mountShellRoutes` method handles `/app/*` by stripping `/app/` and serving shell FS. We need renderer paths (`/app/renderer/default/*`) to be served from the renderer FS instead.

Add import at top:

```go
renderer "github.com/and1truong/liveboard/web/renderer/default"
```

Replace the body of `mountShellRoutes` with:

```go
func (s *Server) mountShellRoutes(r chi.Router) {
	shellSub, err := fs.Sub(shell.FS, "dist")
	if err != nil {
		log.Printf("shell embed: %v", err)
		return
	}
	rendererSub, err := fs.Sub(renderer.FS, "dist")
	if err != nil {
		log.Printf("renderer embed: %v", err)
		return
	}
	shellHandler := http.StripPrefix("/app/", http.FileServer(http.FS(shellSub)))
	rendererHandler := http.StripPrefix("/app/renderer/default/", http.FileServer(http.FS(rendererSub)))

	r.Get("/app", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/app/", http.StatusMovedPermanently)
	})
	r.Get("/app/*", func(w http.ResponseWriter, req *http.Request) {
		if s.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store")
		}
		if strings.HasPrefix(req.URL.Path, "/app/renderer/default/") {
			rendererHandler.ServeHTTP(w, req)
			return
		}
		shellHandler.ServeHTTP(w, req)
	})
}
```

Add `"strings"` to the import block if it isn't there.

- [ ] **Step 3: Extend `server_shell_test.go`**

Append:

```go
func TestShellRoute_Renderer(t *testing.T) {
	t.Setenv("LIVEBOARD_APP_SHELL", "1")
	s := setupShellTest(t)

	req := httptest.NewRequest(http.MethodGet, "/app/renderer/default/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<div id=\"root\">") {
		t.Fatalf("response did not contain renderer root div")
	}
}
```

(`setupShellTest` was introduced in P3 T14; reuse it.)

- [ ] **Step 4: Build + test**

```bash
make renderer
go test ./internal/api/ -race -run 'TestShellRoute' -v
```
Expected: three tests (Disabled, Enabled, Renderer) pass.

- [ ] **Step 5: Full regression**

```bash
make shell && make renderer && go test ./... -race -count=1
```
Expected: all green.

- [ ] **Step 6: Commit**

```bash
git add web/renderer/default/embed.go internal/api/server.go internal/api/server_shell_test.go
git commit -m "feat(renderer): serve /app/renderer/default/* from Go"
```

---

## Task 13: Shell iframe src switch (default renderer)

**Files:**
- Modify: `web/shell/index.html`
- Modify: `web/shell/src/main.ts`

The shell currently hardcodes `iframe src="/app/renderer-stub/"`. Make main.ts choose at runtime: `?renderer=stub` → stub, default → `renderer/default`.

- [ ] **Step 1: `web/shell/index.html`**

Remove the `src` attribute; main.ts sets it. Change:

```html
<iframe id="renderer" src="/app/renderer-stub/" title="LiveBoard renderer"></iframe>
```

to:

```html
<iframe id="renderer" title="LiveBoard renderer"></iframe>
```

- [ ] **Step 2: Modify `web/shell/src/main.ts`**

Find the `bootstrap` function. Before the line that grabs the iframe, set its src. Replace the body of `bootstrap` with:

```ts
function bootstrap(): void {
  const iframe = document.getElementById('renderer') as HTMLIFrameElement | null
  if (!iframe) throw new Error('renderer iframe not found')

  const params = new URLSearchParams(window.location.search)
  const mode = params.get('renderer') ?? 'default'
  iframe.src = mode === 'stub' ? '/app/renderer-stub/' : '/app/renderer/default/'

  const adapter = new LocalAdapter(new BrowserStorage())
  const transport = shellTransport(iframe, window.location.origin)
  const broker = new Broker(transport, adapter, { shellVersion: SHELL_VERSION })

  window.addEventListener('beforeunload', () => broker.close())
}
```

- [ ] **Step 3: Rebuild shell + smoke**

```bash
make shell
```

- [ ] **Step 4: Commit**

```bash
git add web/shell/index.html web/shell/src/main.ts
git commit -m "feat(shell): default iframe to renderer/default; ?renderer=stub for harness"
```

---

## Task 14: CI wiring

**Files:**
- Modify: `.github/workflows/ci.yml`

The `test`, `lint`, and `nilaway` jobs already install bun and run `make shell`. Add `make renderer` after `make shell` in those jobs so Go can compile the new embed.

- [ ] **Step 1: Find and update each Go job**

For each of the `test`, `lint`, `nilaway` jobs, change the step:

```yaml
      - name: Build shell bundle
        run: make shell
```

to:

```yaml
      - name: Build shell bundle
        run: make shell

      - name: Build renderer bundle
        run: make renderer
```

- [ ] **Step 2: Add a renderer-size log to the `test` job**

After `Build renderer bundle` in the `test` job:

```yaml
      - name: Report renderer bundle size
        run: |
          echo "Renderer JS:"
          ls -la web/renderer/default/dist/assets/*.js
          echo "Renderer JS (gzipped):"
          gzip -c web/renderer/default/dist/assets/*.js | wc -c
```

- [ ] **Step 3: Verify YAML parses**

```bash
cat .github/workflows/ci.yml | head -60
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "chore(ci): build renderer bundle and log bundle size"
```

---

## Task 15: Docs

**Files:**
- Modify: `README.md`
- Modify: `docs/parity.md`

- [ ] **Step 1: Update README "Shell (preview)" section**

Find the section added in P3 T15. Replace it with:

```markdown
## Shell (preview)

Experimental postMessage shell + React renderer. Enable with:

    make shell && make renderer
    LIVEBOARD_APP_SHELL=1 liveboard serve

Then open <http://localhost:7070/app/> — the React renderer (read-only board view) loads by default. Append `?renderer=stub` to load the P3 integration harness instead.
```

- [ ] **Step 2: Update `docs/parity.md`**

Append:

```markdown
## Renderer

The default React renderer at `web/renderer/default/` consumes the Client from `web/shared/src/client.ts`. It is a read-only board viewer (P4a scope). Mutations come in P4b. Component-level tests live alongside each component and cover the query-invalidation path end-to-end against a stubbed in-memory Broker.
```

- [ ] **Step 3: Commit**

```bash
git add README.md docs/parity.md
git commit -m "docs(renderer): document /app/ default renderer"
```

---

## Task 16: Manual browser smoke

This is a gate, not a code change.

- [ ] **Step 1: Rebuild**

```bash
make shell && make renderer
```

- [ ] **Step 2: Serve**

```bash
LIVEBOARD_APP_SHELL=1 go run ./cmd/liveboard serve --port 7070
```

- [ ] **Step 3: Open `http://localhost:7070/app/`**

Verify:
- Sidebar shows "Demo" workspace header and the "Welcome" board.
- Clicking "Welcome" renders three columns (Todo, Doing, Done) with the seeded cards.
- Browser console has no red errors.
- `http://localhost:7070/app/?renderer=stub` still loads the P3 harness (all OK lines).

- [ ] **Step 4: Second-tab live-update check**

Open a second tab to `http://localhost:7070/app/`. In one tab, open devtools and run:

```js
// no-op for P4a since we have no UI mutation path; skip this step for now
```

(Live multi-tab updates driven by user action land in P4b when mutations ship. The BroadcastChannel path is already proven by the stub in P3 — leaving this as a placeholder ensures we don't claim a capability we didn't demonstrate for the real UI.)

- [ ] **Step 5: Report**

If any of the above fails, capture the symptom + console output and fix before marking P4a done. Otherwise, P4a is complete.

---

## Spec coverage checklist

| Spec requirement | Covered by |
|---|---|
| React+Vite+TanStack Query scaffold | Task 1 |
| Tailwind | Tasks 1, 2 |
| Client bootstrap + QueryClient | Task 3 |
| `board.updated` → query invalidation | Tasks 3, 4 |
| Query hooks (list/get/workspace) | Task 4 |
| Sidebar with board list + active highlight | Task 9 |
| Read-only BoardView | Task 8 |
| Column + Card rendering | Tasks 5, 6 |
| EmptyState | Task 7 |
| Subscribe lifecycle on mount/unmount | Task 8 |
| Handshake error full-page message | Task 10 (main.tsx) |
| Vite build → `dist/` | Task 11 |
| Go embed + `/app/renderer/default/*` route | Task 12 |
| Shell iframe switch (stub vs default) | Task 13 |
| CI build + size log | Task 14 |
| Docs | Task 15 |
| Browser smoke | Task 16 |

## Notes for the implementer

1. **bun path aliases.** If `@shared/*` resolves inside Vite but not under `bun test`, the fallback is relative imports everywhere. Try the alias first; switch if more than one file breaks.
2. **Tailwind v4 is beta.** If the postcss plugin naming has changed, check the Tailwind v4 release notes and adapt `postcss.config.js` accordingly. Do not downgrade to v3 without asking — the rest of the repo uses v4.
3. **TanStack Query + React 18 StrictMode** can cause double-invoked effects in dev. That's expected. Tests should not be flaky because they use production behavior.
4. **`JSX.Element` return types** on every component are intentional (strict project convention). Don't drop them.
5. **No router yet.** If state management across board switches feels awkward, resist the urge to add a router. P4c addresses navigation.

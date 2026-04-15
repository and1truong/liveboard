# P4d.2 — Themes (Light / Dark + Color Themes) — Design

## Goal

Add workspace-level theming to the `/app/` default renderer:
- **Mode**: light / dark / system (auto via `prefers-color-scheme`).
- **Color theme**: one of six palettes (Indigo, GitHub, GitLab, Emerald, Rose, Sunset), matching the HTMX UI.
- Persisted in `localStorage`. Flash-of-wrong-theme eliminated via an inline script in `index.html`.
- Surfaced as a small picker in the sidebar footer.

**Shippable value:** parity with the HTMX UI's theming. After P4d.2, only P4d.3 (bundle gate + cleanup) remains in P4.

## Scope

**In:**
- `<ThemeProvider>` + `useTheme()` — owns mode/theme state, persists, applies classes to `<html>`, subscribes to `prefers-color-scheme`.
- `<ThemePicker>` — Radix DropdownMenu in the sidebar footer: Mode radios + 6 swatches.
- Tailwind `darkMode: 'class'`.
- `themes.css` with `:root.theme-*` custom properties (`--accent-500`, `--accent-600`).
- Inline pre-React script in `index.html` to pre-set classes.
- Initial accent-color wiring on a small set of components (Save buttons, focus rings).
- Dark-mode surface styling on top-level containers (sidebar, board area, cards, modals).

**Out:**
- Per-board theme override.
- Auto-contrast text colors per theme (themes share one accent pair each; dark-mode text is handled by Tailwind dark: variants, not by theme).
- Custom theme creator.
- Exposed via command palette (could add in a future pass; not strictly needed).
- Full component-by-component dark-mode polish (P4d.3 catches stragglers).

## Architecture

```
<html class="dark theme-emerald">     # set by inline script + ThemeProvider
 └─ <ThemeProvider>
     ├─ mode:  'light' | 'dark' | 'system'
     ├─ theme: 'indigo' | 'github' | 'gitlab' | 'emerald' | 'rose' | 'sunset'
     ├─ resolvedDark: boolean
     ├─ setMode / setTheme (persist + re-apply classes)
     └─ subscribes to matchMedia when mode === 'system'

BoardSidebar
 └─ <ThemePicker />
     └─ Radix DropdownMenu
         ├─ Mode: Light / Dark / System  (radios)
         └─ Theme: 6 colored swatches
```

## Storage contract

- `liveboard:mode` → `'light' | 'dark' | 'system'` (default `'system'`).
- `liveboard:theme` → one of the six theme names (default `'indigo'`).

Both keys sit under the existing `liveboard:` prefix in localStorage alongside adapter keys, but never collide (no `board:` or `workspace` suffix).

## `<html>` class model

- `dark` — present iff `resolvedDark` is true.
- `theme-<name>` — exactly one of the six.

Any `theme-*` class already present is removed before the new one is added. The provider never touches other classes on `<html>`.

## `themes.css`

```css
:root { --accent-500: #6366f1; --accent-600: #4f46e5; }        /* fallback = indigo */
:root.theme-indigo  { --accent-500: #6366f1; --accent-600: #4f46e5; }
:root.theme-github  { --accent-500: #2da44e; --accent-600: #1a7f37; }
:root.theme-gitlab  { --accent-500: #fc6d26; --accent-600: #e24329; }
:root.theme-emerald { --accent-500: #10b981; --accent-600: #059669; }
:root.theme-rose    { --accent-500: #f43f5e; --accent-600: #e11d48; }
:root.theme-sunset  { --accent-500: #f97316; --accent-600: #ea580c; }
```

Imported once from `src/styles/tailwind.css`:
```css
@import 'tailwindcss';
@import './themes.css';
```

(Adjust to the renderer's existing CSS entry; path may differ.)

## Flash-of-wrong-theme script

Added to `web/renderer/default/index.html` BEFORE the `<script type="module" src="/src/main.tsx">` tag, inside `<head>`:

```html
<script>
  try {
    const m = localStorage.getItem('liveboard:mode') || 'system';
    const t = localStorage.getItem('liveboard:theme') || 'indigo';
    const dark = m === 'dark' || (m === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
    document.documentElement.classList.toggle('dark', dark);
    document.documentElement.classList.add('theme-' + t);
  } catch {}
</script>
```

The `ThemeProvider` still owns runtime state; this script only handles the pre-hydration render.

## Component contracts

### `<ThemeProvider>` + `useTheme()`

```ts
type Mode = 'light' | 'dark' | 'system'
type ThemeName = 'indigo' | 'github' | 'gitlab' | 'emerald' | 'rose' | 'sunset'

interface ThemeCtx {
  mode: Mode
  theme: ThemeName
  resolvedDark: boolean
  setMode: (m: Mode) => void
  setTheme: (t: ThemeName) => void
}
```

Initialization:
1. Read `liveboard:mode` + `liveboard:theme` from localStorage; fall back to defaults.
2. `useState` seeded from step 1.
3. `useEffect` subscribes to `matchMedia('(prefers-color-scheme: dark)')` while `mode === 'system'`; updates `resolvedDark` on change; cleans up on mode/unmount.
4. `useEffect` keyed on `(mode, resolvedDark)` applies the `dark` class, persists `mode`.
5. `useEffect` keyed on `theme` removes all `theme-*` classes from `<html>` and adds the new one; persists `theme`.

`useTheme()` reads context; throws if used outside provider.

### `<ThemePicker />`

No props. Uses `useTheme()`.

```tsx
<DropdownMenu.Root>
  <DropdownMenu.Trigger aria-label="Theme picker">
    🎨  {/* or a Tabler-style sun/moon icon */}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content sideOffset={4} className="...">
    <DropdownMenu.Label>Mode</DropdownMenu.Label>
    <DropdownMenu.RadioGroup value={mode} onValueChange={setMode}>
      <DropdownMenu.RadioItem value="light">Light</DropdownMenu.RadioItem>
      <DropdownMenu.RadioItem value="dark">Dark</DropdownMenu.RadioItem>
      <DropdownMenu.RadioItem value="system">System</DropdownMenu.RadioItem>
    </DropdownMenu.RadioGroup>
    <DropdownMenu.Separator />
    <DropdownMenu.Label>Theme</DropdownMenu.Label>
    <div className="flex gap-1 p-1">
      {THEME_SWATCHES.map((t) => (
        <button
          key={t.name}
          aria-label={`theme ${t.name}`}
          aria-pressed={theme === t.name}
          onClick={() => setTheme(t.name)}
          style={{ backgroundColor: t.swatchColor }}
          className={`h-6 w-6 rounded-full ${theme === t.name ? 'ring-2 ring-offset-2 ring-slate-400' : ''}`}
        />
      ))}
    </div>
  </DropdownMenu.Content>
</DropdownMenu.Root>
```

`THEME_SWATCHES` is a local constant array of `{ name, swatchColor }` pairs using the `--accent-500` literal hex values.

### Sidebar footer placement

In `BoardSidebar`, insert `<ThemePicker />` between the boards list container and `<AddBoardButton />`, e.g. inside a small `<div className="flex items-center justify-end px-2 py-1 border-t">`.

## Accent-color wiring (conservative scope)

Replace hard-coded `bg-blue-600`, `bg-blue-400`, `ring-blue-400`, `focus:ring-blue-400` on these surfaces:
- `<BoardSettingsModal>` Save button.
- `<CardDetailModal>` Save button.
- `<AddCardButton>`, `<AddColumnButton>`, `<AddBoardButton>` inline inputs (focus ring).
- `<CardEditable>` inline rename input (focus ring).
- `<ColumnHeader>` inline rename input (focus ring).
- `<SortableCard>` focus ring.
- `<BoardRow>` inline rename input (focus ring).

Each becomes `bg-[color:var(--accent-600)] hover:bg-[color:var(--accent-500)]` or `focus:ring-[color:var(--accent-500)]` or `ring-[color:var(--accent-500)]`.

## Dark-mode surface styling

Minimum viable dark styling applied at the top-level containers:

| Component / element | Light | Dark |
|---|---|---|
| `<aside>` sidebar | `bg-white border-slate-200` | `dark:bg-slate-900 dark:border-slate-800` |
| Workspace header text | `text-slate-800` | `dark:text-slate-100` |
| `<main>` board area background | body default | `dark:bg-slate-950` |
| Card surface | `bg-white ring-slate-200` | `dark:bg-slate-800 dark:ring-slate-700` |
| Card title text | `text-slate-900` | `dark:text-slate-100` |
| Column surface | `bg-slate-100` | `dark:bg-slate-900` |
| Column name | `text-slate-800` | `dark:text-slate-100` |
| Modal overlay | `bg-black/40` | unchanged |
| Modal content | `bg-white` | `dark:bg-slate-800` |
| Modal title | `text-slate-800` | `dark:text-slate-100` |
| Toast host | sonner `richColors` theming | inherits |

Buttons and smaller surfaces inherit enough contrast from parents for P4d.2. P4d.3 sweeps stragglers.

## Testing

### `ThemeContext.test.tsx`

- Default: `mode === 'system'`, `theme === 'indigo'`, no read from localStorage returns → provider uses defaults.
- `setMode('dark')` persists `liveboard:mode=dark` and adds `dark` class.
- `setMode('light')` removes the `dark` class.
- `setTheme('emerald')` persists `liveboard:theme=emerald`, removes `theme-indigo`, adds `theme-emerald`.
- Provider read initializer: pre-seed localStorage then mount → mode/theme match seeded values.

happy-dom provides `window.matchMedia` as a stub. If the stub doesn't emit `change` events, the `setMode('system')` + OS-change path is covered only by manual smoke.

### `ThemePicker.test.tsx`

- Trigger renders; menu opens on click.
- Mode radios reflect the current mode; clicking one calls `setMode`.
- 6 swatches render with correct `aria-pressed` for the active one.
- Clicking a swatch calls `setTheme(name)`.

Radix DropdownMenu + happy-dom — same `waitFor` patterns as P4b.1a / P4c.1 `ColumnHeader` / `BoardRow`. Keep coverage minimal; manual smoke covers the visual.

## Visual spec

- ThemePicker trigger: 24×24 button with a palette icon or a 2×2 mini-grid of the current theme color + a moon. Text-only "🎨" is acceptable for P4d.2.
- Swatches: 24×24 rounded-full buttons with the hex accent color; active swatch has `ring-2 ring-offset-2`.
- Mode radios: standard Radix styling with a check next to the active value.

Dark mode visual: slate-950 board background, slate-900 columns, slate-800 cards. Text contrast at WCAG AA.

## Risks

- **Tailwind v4 arbitrary color value syntax**: `bg-[color:var(--accent-600)]` must be valid in the project's Tailwind version. Verify with a one-off build after wiring the first button; fall back to explicit `style={{backgroundColor: 'var(--accent-600)'}}` if the Tailwind build rejects.
- **`darkMode: 'class'` first-time breakage**: every component without explicit dark styles falls back to the light-mode palette even under `.dark`. Spec scopes dark styling to top-level surfaces to avoid chasing every leaf. Acceptable for P4d.2; P4d.3 finishes.
- **Flash of wrong theme in dev HMR**: Vite's HMR doesn't re-run the inline script on reload, but the ThemeProvider re-applies classes on mount, so the first paint may briefly flash. Acceptable in dev.
- **matchMedia change events under happy-dom**: system-mode re-resolve on OS change is verified only by manual smoke (toggle OS dark mode with "system" selected in picker).
- **Theme choice persistence across workspaces**: single localStorage namespace → same theme applies to every `/app/` workspace in the same origin. Intentional; matches HTMX UI.

## Open questions

None blocking. Pre-decided:
- 6 themes, workspace-level.
- Light/Dark/System, default System.
- Sidebar-footer picker via Radix menu.
- Inline pre-React script prevents flash.
- Tailwind `darkMode: 'class'`.
- Dark styling scoped to top-level surfaces in P4d.2; stragglers in P4d.3.

## Dependencies on prior work

- P4b.1a: Radix DropdownMenu pattern.
- P4a: Vite config and existing CSS entry.
- P4c.1: `<BoardSidebar>` layout (footer space already reserved via `<AddBoardButton>` border-top convention).
- P4d.1: `<BoardSettingsModal>` Save button to re-color.

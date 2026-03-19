---
name: alpinejs
description: "Write Alpine.js client-side interactivity in HTML templates. Use this skill whenever the user wants to add reactive behavior, event handling, two-way binding, transitions, conditional rendering, list iteration, or any x- directive to HTML markup. Also trigger when the user mentions Alpine, x-data, x-on, x-model, x-show, x-if, x-for, x-bind, $store, $dispatch, $refs, or any Alpine plugin (Mask, Intersect, Sort, Morph, Focus, Collapse, Anchor, Persist, Resize)."
---

# Alpine.js Skill

Alpine.js (~17kB) is included via CDN `<script defer>` tag. No build step. HTML **is** the template — all behavior is declared via `x-` attributes directly in markup.

## Directives Quick Reference

### State & Initialization
- `x-data="{ key: value }"` — declares component + reactive scope. All other directives require an ancestor `x-data`. Supports object literals, `Alpine.data()` refs, or empty. `init()` auto-called on init, `destroy()` on removal.
- `x-init="expression"` — runs once on element init. Supports `await`. If `x-data` has `init()` too, `init()` runs first.

### Rendering
- `x-text="expr"` — sets `innerText` (safe, no HTML)
- `x-html="expr"` — sets `innerHTML` (**XSS risk** — trusted content only)
- `x-show="expr"` — toggles `display: none`. Element stays in DOM. Works with `x-transition`. Modifier: `.important`
- `x-if="expr"` — **must be on `<template>`** with single root child. Adds/removes from DOM entirely. **No `x-transition` support.**
- `x-for="item in items"` — **must be on `<template>`** with single root child. Supports `(item, index) in items`, `(value, key) in object`, `i in 10`. Use `:key` for efficient reorder.
- `x-cloak` — hidden until Alpine inits. Requires CSS: `[x-cloak] { display: none !important; }`

### Binding & Events
- `x-bind:attr="expr"` or `:attr="expr"` — dynamic attributes. Special: `:class` (object syntax `{ 'hidden': !show }`, preserves existing), `:style` (object syntax, merges existing). Can bind entire object of directives.
- `x-on:event="expr"` or `@event="expr"` — event listener. `$event` available.
  - Modifiers: `.prevent`, `.stop`, `.outside`, `.window`, `.document`, `.once`, `.debounce[.Nms]`, `.throttle[.Nms]`, `.self`, `.camel`, `.dot`, `.passive`, `.capture`
  - Keyboard: `.enter`, `.space`, `.escape`, `.tab`, `.shift`, `.ctrl`, `.cmd`, `.meta`, `.alt`, `.up`, `.down`, `.left`, `.right`, plus any `KeyboardEvent.key` in kebab-case
- `x-model="prop"` — two-way binding for inputs/textareas/selects/checkboxes/radios.
  - Modifiers: `.lazy`, `.number`, `.boolean`, `.debounce`, `.throttle`, `.fill`, `.blur`, `.enter`
- `x-modelable="prop"` — expose internal state for external `x-model` binding

### DOM & Lifecycle
- `x-ref="name"` — element ref accessible via `$refs.name`
- `x-effect="expr"` — reactive side effect, auto-detects dependencies, re-runs on change
- `x-transition` — animate `x-show`. Modifier-based: `.duration.Nms`, `.delay.Nms`, `.opacity`, `.scale[.N]`, `.origin.top[.right]`. CSS class-based: `x-transition:enter`, `:enter-start`, `:enter-end`, `:leave`, `:leave-start`, `:leave-end`
- `x-teleport="selector"` — **must be on `<template>`**. Moves content elsewhere in DOM, retains Alpine scope.
- `x-id="['name']"` — scopes `$id()` generation for consistent IDs
- `x-ignore` — skip Alpine processing for element and descendants

## Magic Properties

- `$el` — current DOM element
- `$refs` — object of `x-ref` elements in component scope
- `$store` — access `Alpine.store()` global stores (reactive)
- `$watch('prop', (newVal, oldVal) => {})` — watch reactive property. **Warning:** modifying watched prop in callback = infinite loop
- `$dispatch('event', detail?)` — dispatch CustomEvent (bubbles). Dispatch `'input'` to trigger parent `x-model`
- `$nextTick(cb?)` — runs after DOM update. Returns Promise if no callback
- `$root` — component root element (nearest `x-data` ancestor)
- `$data` — full reactive scope object
- `$id('name', suffix?)` — unique ID within `x-id` scope
- `$event` — native event object inside `x-on` handlers

## Global API

- `Alpine.data('name', () => ({...}))` — register reusable component. Factory returns data object with props, methods, `init()`, `destroy()`
- `Alpine.store('name', data)` — create global reactive store. `Alpine.store('name')` retrieves it. Supports `init()` method
- `Alpine.bind('name', () => ({...}))` — reusable attribute/directive bundles, applied via `x-bind="name"`
- `Alpine.reactive(obj)` — wrap object in reactive Proxy
- `Alpine.effect(cb)` — persistent reactive side effect (not element-scoped)
- `Alpine.directive('name', (el, {value, modifiers, expression}, {effect, cleanup, evaluate, evaluateLater}) => {})` — custom `x-name` directive
- `Alpine.magic('name', (el) => value)` — custom `$name` magic
- `Alpine.plugin(cb)` — register plugin (calls cb with Alpine)
- `Alpine.start()` — init Alpine (once per page, after all registration)

## Lifecycle Events

- `document` fires `alpine:init` (register stores/directives/plugins here) then `alpine:initialized` (page fully processed)

## Official Plugins

- **Morph** — `Alpine.morph(el, newHtml, opts?)`, `Alpine.morphBetween(start, end, newHtml, opts?)`. Preserves Alpine state during DOM patch. Hooks: `updating`, `updated`, `removing`, `removed`, `adding`, `added`, `key`, `lookahead`
- **Mask** — `x-mask="999-999"` (wildcards: `9`=digit, `a`=letter, `*`=any). `x-mask:dynamic` with `$input`. `$money($input, dec?, thou?, precision?)`
- **Intersect** — `x-intersect="expr"`, `x-intersect:leave`. Modifiers: `.once`, `.half`, `.full`, `.threshold.N`, `.margin.Npx`
- **Resize** — `x-resize="expr"` with `$width`, `$height`. Modifier: `.document`
- **Persist** — `$persist(value)` in `x-data`. `.as('key')`, `.using(sessionStorage)`
- **Focus** — `x-trap="expr"` traps focus. Modifiers: `.inert`, `.noscroll`, `.noreturn`, `.noautofocus`. `$focus` magic API
- **Collapse** — `x-collapse` with `x-show`. Modifiers: `.duration.Nms`, `.min.Npx`
- **Anchor** — `x-anchor="$refs.el"`. Position: `.bottom`, `.top[-start/-end]`, `.left`, `.right`. `.offset.Npx`, `.no-style` exposes `$anchor.x/y`
- **Sort** — `x-sort` container + `x-sort:item` children. `x-sort:handle`, `x-sort:group="name"`, `x-sort:ignore`, `x-sort:config`. `$item`, `$position` in handler. `.ghost` modifier

## Preferred Patterns & Examples

These are the idiomatic Alpine.js patterns to follow. Each demonstrates how to compose directives together for common UI needs.

### Toggle / Dropdown
The most common Alpine pattern — a boolean controlling visibility with click-outside dismissal:
```html
<div x-data="{ open: false }">
    <button @click="open = !open">Toggle</button>
    <div x-show="open" @click.outside="open = false" x-transition>
        Dropdown contents...
    </div>
</div>
```

### Search Filter with Computed Getter
Reactive filtering using a JS getter — the filtered list updates automatically as the user types:
```html
<div x-data="{
    search: '',
    items: ['foo', 'bar', 'baz'],
    get filteredItems() {
        return this.items.filter(i => i.startsWith(this.search))
    }
}">
    <input x-model="search" placeholder="Search...">
    <ul>
        <template x-for="item in filteredItems" :key="item">
            <li x-text="item"></li>
        </template>
    </ul>
</div>
```

### Reusable Component via Alpine.data()
Extract repeated logic into a registered component so multiple elements share the same behavior without duplicating code:
```html
<script>
document.addEventListener('alpine:init', () => {
    Alpine.data('dropdown', () => ({
        open: false,
        toggle() { this.open = !this.open },
    }))
})
</script>

<div x-data="dropdown">
    <button @click="toggle">Expand</button>
    <span x-show="open">Content...</span>
</div>
```

### Reusable Directive Bundles via x-bind Object
Encapsulate entire directive sets (event handlers, show/hide, refs) into named objects — keeps templates clean and makes complex components composable:
```html
<div x-data="dropdown">
    <button x-bind="trigger">Open</button>
    <span x-bind="dialogue">Contents</span>
</div>

<script>
document.addEventListener('alpine:init', () => {
    Alpine.data('dropdown', () => ({
        open: false,
        trigger: {
            ['@click']() { this.open = true },
        },
        dialogue: {
            ['x-show']() { return this.open },
            ['@click.outside']() { this.open = false },
        },
    }))
})
</script>
```

### Global State with $store
Share state across unrelated components anywhere on the page:
```html
<script>
document.addEventListener('alpine:init', () => {
    Alpine.store('tabs', {
        current: 'first',
        items: ['first', 'second', 'third'],
    })
})
</script>

<div x-data>
    <template x-for="tab in $store.tabs.items">
        <button @click="$store.tabs.current = tab"
                :class="{ 'active': $store.tabs.current === tab }"
                x-text="tab"></button>
    </template>
</div>
```

### Cross-Component Communication with $dispatch
Use custom events to communicate between sibling components that don't share scope:
```html
<div x-data @notify.window="alert($event.detail.message)">
    Listener...
</div>

<div x-data>
    <button @click="$dispatch('notify', { message: 'Hello!' })">
        Send
    </button>
</div>
```

### Class Binding Patterns
Object syntax preserves existing classes and is the preferred approach:
```html
<!-- Object syntax (preferred) -->
<div :class="{ 'hidden': !show, 'bg-red': isError }">...</div>

<!-- Ternary for swapping classes -->
<div :class="active ? 'bg-blue text-white' : 'bg-gray'">...</div>

<!-- Short-circuit for single class -->
<div :class="loading && 'opacity-50'">...</div>
```

### Style Binding
Object syntax merges with existing inline styles:
```html
<div style="padding: 1rem;" :style="{ color: 'red', display: 'flex' }">...</div>
```

### Form Inputs with x-model
Two-way binding works with all input types — use modifiers to control sync timing and type coercion:
```html
<!-- Text with debounce -->
<input type="text" x-model.debounce.500ms="search">

<!-- Lazy sync (on blur/change, not every keystroke) -->
<input type="text" x-model.lazy="username">
<span x-show="username.length > 20">Too long</span>

<!-- Number coercion -->
<input type="text" x-model.number="age">

<!-- Multiple checkboxes → array -->
<input type="checkbox" value="red" x-model="colors">
<input type="checkbox" value="blue" x-model="colors">
<span x-text="colors"></span>  <!-- ["red", "blue"] -->
```

### Transitions with CSS Classes (Tailwind)
Full control over enter/leave animations — each phase gets its own classes:
```html
<div x-show="open"
    x-transition:enter="transition ease-out duration-300"
    x-transition:enter-start="opacity-0 transform scale-90"
    x-transition:enter-end="opacity-100 transform scale-100"
    x-transition:leave="transition ease-in duration-200"
    x-transition:leave-start="opacity-100 transform scale-100"
    x-transition:leave-end="opacity-0 transform scale-90">
    ...
</div>
```

### Init with Async Fetch
Load data on component initialization — `x-init` supports `await` natively:
```html
<div x-data="{ posts: [] }"
     x-init="posts = await (await fetch('/api/posts')).json()">
    <template x-for="post in posts" :key="post.id">
        <h2 x-text="post.title"></h2>
    </template>
</div>
```

### Watching State Changes
React to specific property changes — useful for side effects like saving or analytics:
```html
<div x-data="{ open: false }"
     x-init="$watch('open', val => console.log('open changed to', val))">
    ...
</div>
```

### Reactive Side Effects with x-effect
Unlike `$watch`, you don't specify what to watch — Alpine auto-detects all dependencies:
```html
<div x-data="{ count: 0 }"
     x-effect="document.title = 'Count: ' + count">
    <button @click="count++">+</button>
</div>
```

## Key Rules

1. `x-if` and `x-for` **must** be on `<template>` with exactly one root child
2. `x-transition` only works with `x-show`, **not** `x-if`
3. `x-html` is an XSS vector — never use with user input
4. Register stores/directives/plugins inside `alpine:init` listener or before `Alpine.start()`
5. CSP build (`@alpinejs/csp`) exists for strict CSP — no arrow functions, template literals, or global function calls in expressions

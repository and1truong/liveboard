# Alpine.js: the complete feature reference

**Alpine.js is a lightweight JavaScript framework — 15 directives, 9 magic properties, and a handful of global methods — that lets you compose reactive behavior directly in HTML markup.** Weighing in at roughly **~17kB**, it positions itself as "jQuery for the modern web," ideal for adding interactivity to server-rendered pages without a full SPA framework. Built on Vue.js's reactivity engine (`@vue/reactivity`), Alpine delivers a declarative, attribute-driven programming model with no virtual DOM, no build step requirement, and zero boilerplate. Below is an exhaustive accounting of every feature, directive, magic property, method, plugin, and capability Alpine.js offers.

---

## Core philosophy and architecture

Alpine exists to fill the gap between plain HTML and heavy JavaScript frameworks like Vue or React. You include a single `<script>` tag (or install via NPM), add attributes to your HTML, and gain full reactivity. There is no compilation step, no JSX, no template language — your HTML **is** your template.

**Installation** happens two ways. Via CDN: include `<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>` in your `<head>` (the `defer` attribute is mandatory). Via NPM: `npm install alpinejs`, then `import Alpine from 'alpinejs'; Alpine.start()`. The `Alpine.start()` call must happen exactly **once per page**. Extension code (custom directives, plugins, stores) must be registered after Alpine is loaded but before `Alpine.start()` — either inside an `alpine:init` event listener or between the import and the start call.

Alpine's reactivity model uses **JavaScript Proxies** (via `Alpine.reactive()`) and an **effect system** (via `Alpine.effect()`) — the same engine that powers Vue 3. When reactive data changes, every expression that depends on it re-evaluates automatically. There is no diffing algorithm or virtual DOM; Alpine patches the real DOM directly.

Two lifecycle events fire on the `document`: **`alpine:init`** (after Alpine loads, before page initialization — use this to register data, stores, directives, and plugins) and **`alpine:initialized`** (after Alpine finishes processing the entire page).

---

## All 18 directives

Alpine's directives are HTML attributes prefixed with `x-` that declare reactive behavior. Every directive below is documented with its full syntax, all modifiers, and key behaviors.

### x-data — the foundation of every component

Declares an Alpine component and provides its reactive data scope. All other directives require an ancestor `x-data`. Accepts a JavaScript object literal (`x-data="{ open: false }"`), a reference to an `Alpine.data()` provider (`x-data="dropdown"`), or can be empty (`x-data` or `x-data="{}"`). Properties, methods, and JavaScript getters are all supported. Child elements inherit parent scope; nested `x-data` components can override parent properties. When calling methods from event handlers, trailing parentheses are optional. An `init()` method inside the data object is auto-called during initialization; a `destroy()` method is auto-called when the component is removed from the DOM (e.g., by `x-if`).

### x-init — run code on initialization

Hooks into the initialization phase of any element. The expression runs once when the element initializes. Works inside or outside `x-data` blocks. Supports `await` for async operations like `fetch`. If both an `x-data` `init()` method and `x-init` exist on the same element, `init()` runs **first**. Combine with `$nextTick` to run code after Alpine finishes rendering.

### x-show — toggle visibility via CSS

Shows or hides an element by toggling `display: none` based on a JavaScript expression. The element **stays in the DOM**. Works with `x-transition` for animations. Has one modifier: **`.important`** — applies `display: none !important` to override CSS specificity conflicts.

### x-bind — dynamic HTML attributes

Sets any HTML attribute to the result of a JavaScript expression. **Shorthand: `:` (colon)**. Has special behavior for two attributes: **class** (preserves existing classes, supports object syntax `{ 'hidden': !show }` and ternary expressions) and **style** (merges with existing styles, supports object syntax `{ color: 'red' }`). Can also bind an **entire object of directives** to an element at once — keys are attribute/directive names, values are strings or callback functions — enabling reusable component templates when combined with `Alpine.data()`.

### x-on — event handling

Listens for any DOM event and executes a JavaScript expression. **Shorthand: `@`**. Access the native event object via the `$event` magic. When referencing methods without parentheses, the event object is passed automatically as the first argument. Event names are case-insensitive; use `.camel` for camelCase custom events. All modifiers:

- **`.prevent`** — calls `event.preventDefault()`
- **`.stop`** — calls `event.stopPropagation()`
- **`.outside`** — fires only when clicks originate outside the element (expression only evaluates when element is visible)
- **`.window`** — registers listener on `window` instead of the element
- **`.document`** — registers listener on `document`
- **`.once`** — handler fires only once
- **`.debounce`** — debounces the handler (default **250ms**, customizable: `.debounce.500ms`)
- **`.throttle`** — throttles the handler (default **250ms**, customizable: `.throttle.750ms`)
- **`.self`** — fires only if the event originated on the element itself, not a child
- **`.camel`** — converts kebab-case event name to camelCase
- **`.dot`** — converts dashes to dots in event name
- **`.passive`** — adds a passive event listener (important for touch/scroll performance)
- **`.capture`** — executes listener in the capturing phase
- **Keyboard modifiers:** `.enter`, `.space`, `.escape`, `.tab`, `.shift`, `.ctrl`, `.cmd`, `.meta`, `.alt`, `.up`, `.down`, `.left`, `.right`, `.caps-lock`, `.equal`, `.period`, `.comma`, `.slash`, `.page-down`, plus any valid `KeyboardEvent.key` name in kebab-case
- **Mouse modifiers** (work on click, auxclick, contextmenu, dblclick, mouseover, mousemove, mouseenter, mouseleave, mouseout, mouseup, mousedown): `.shift`, `.ctrl`, `.cmd`, `.meta`, `.alt`

### x-text — set text content

Sets an element's `innerText` to the result of a JavaScript expression. Safe for user content (no HTML rendering). Reactive — updates automatically when data changes.

### x-html — set inner HTML

Sets an element's `innerHTML` to the result of an expression. **Security warning:** never use on user-provided content due to **XSS vulnerability** risk. Only use on trusted content.

### x-model — two-way data binding

Binds an input element's value to Alpine data bidirectionally. Supports: text inputs, textareas, checkboxes (single boolean or multiple array), radio buttons, selects (single, multiple, with placeholder, dynamically populated), and range inputs. Exposes programmatic access via `el._x_model.get()` and `el._x_model.set()`. All modifiers:

- **`.lazy`** — syncs on `change` event (when input loses focus) instead of every keystroke
- **`.change`** — functionally equivalent to `.lazy`
- **`.blur`** — syncs when input loses focus regardless of value change
- **`.enter`** — syncs when Enter key is pressed
- **`.number`** — casts the value to a JavaScript number
- **`.boolean`** — casts the value to a JavaScript boolean
- **`.debounce`** — debounces updates (default 250ms, customizable)
- **`.throttle`** — throttles updates (default 250ms, customizable)
- **`.fill`** — if the bound property is empty, populates it from the input's `value` attribute

The `.change`, `.blur`, and `.enter` modifiers can be combined.

### x-modelable — expose internal state for x-model binding

Exposes any Alpine property as the target of an external `x-model` directive. Creates a two-way binding bridge between nested Alpine scopes. Used with backend templating frameworks to make reusable components whose internal state can be bound from outside, as if it were a native input.

### x-for — list rendering

Iterates over arrays, objects, or numeric ranges to create repeated DOM elements. **Must be declared on a `<template>` element**, and that template **must contain exactly one root element**. Supports index access via `(item, index) in items` syntax, object iteration via `(value, key) in object`, and numeric ranges via `i in 10`. Use `:key` on the template element for efficient re-ordering. Nested loops are supported and have access to parent loop variables.

### x-transition — animate show/hide

Provides smooth CSS transitions when elements are shown or hidden with `x-show`. **Does not work with `x-if`.** Two approaches:

**Transition Helper (modifier-based):** Add `x-transition` for default fade + scale animation. Defaults: **150ms entering, 75ms leaving**. Modifiers: `.duration.500ms` (custom duration), `.delay.50ms`, `.opacity` (opacity only), `.scale` (scale only, customizable: `.scale.80`), `.origin.top` (transform origin, combinable: `.origin.top.right`). Enter and leave can be configured separately: `x-transition:enter.duration.500ms x-transition:leave.duration.400ms`.

**CSS Classes (full control):** Six directives for granular animation stages: `x-transition:enter`, `x-transition:enter-start`, `x-transition:enter-end`, `x-transition:leave`, `x-transition:leave-start`, `x-transition:leave-end`. Each accepts CSS class strings (e.g., Tailwind classes). `enter` and `leave` apply during the entire phase; `start` classes are added before/immediately and removed after one frame; `end` classes are added after one frame and removed when animation completes.

### x-effect — reactive side effects

Re-evaluates an expression whenever any of its reactive dependencies change. Unlike `$watch`, you don't specify what to watch — Alpine **auto-detects all reactive properties** used in the expression. Runs immediately on initialization and on every subsequent dependency change. Automatically cleaned up when the element is removed.

### x-ref — name a DOM element

Assigns a reference name to an element, accessible via `$refs.name`. A scoped, succinct alternative to `document.querySelector`. Dynamic refs via `:x-ref="expression"` are supported.

### x-cloak — prevent flash of unstyled content

Hides an element until Alpine finishes initializing. **Requires CSS:** `[x-cloak] { display: none !important; }`. Alpine removes the `x-cloak` attribute on initialization, making the element visible.

### x-ignore — skip Alpine processing

Prevents Alpine from initializing the element and all its descendants. Useful for sections managed by third-party libraries.

### x-if — conditional DOM rendering

Conditionally adds or **completely removes** elements from the DOM. **Must be on a `<template>` tag** with a single root child element. Unlike `x-show`, the element doesn't exist in the DOM when the condition is false. **Does not support `x-transition`.**

### x-teleport — move elements in the DOM

Transports Alpine template content to another location in the DOM, specified by a CSS selector. **Must be on a `<template>` element.** Teleported content retains access to the original Alpine scope. Useful for modals that need to escape z-index stacking contexts. Supports nesting. Events from teleported content bubble from their actual position; use event listeners on the `<template>` tag to bridge this gap.

### x-id — scope unique IDs

Declares an ID scope for `$id()` generation. Accepts an array of ID name strings. Within a scope, all calls to `$id('name')` return the **same** generated ID. Supports nesting (inner scopes get unique IDs). Designed for reusable components that need consistent label-for/input-id pairings and ARIA attributes.

---

## All 9 magic properties (plus $event)

Magic properties are prefixed with `$` and available in any Alpine expression.

**`$el`** returns the **current DOM element** where the expression is declared. Useful for passing the element to third-party libraries or manipulating DOM properties directly.

**`$refs`** is an object containing all elements marked with `x-ref` in the same component scope. Keys are ref names; values are native DOM elements. Scoped to the nearest `x-data` component.

**`$store`** provides access to global stores registered via `Alpine.store()`. Stores are reactive — expressions using `$store` automatically re-evaluate when store data changes. Supports both object stores with methods and single-value primitive stores.

**`$watch(property, callback)`** watches a reactive property (specified as a dot-notation string) and fires a callback on change. The callback receives `(newValue, oldValue)`. Supports deeply nested properties. Auto-watches all levels of an object. **Warning:** modifying a watched property inside the callback causes an infinite loop.

**`$dispatch(eventName, detail?)`** dispatches a `CustomEvent` from the current element. The optional second argument becomes `event.detail`. Events bubble up through the DOM — use `.window` on the listener for sibling/cross-component communication. Dispatching an `'input'` event can trigger `x-model` updates on parent elements, enabling custom input components.

**`$nextTick(callback?)`** executes code after Alpine completes its reactive DOM updates. Accepts a callback or returns a **Promise** for async/await usage. Essential when you need to read DOM state that reflects recent data changes.

**`$root`** returns the root DOM element of the current Alpine component (the nearest ancestor with `x-data`).

**`$data`** returns the full reactive data scope object, including merged parent scope data. Useful for passing Alpine's entire state to external JavaScript functions.

**`$id(name, suffix?)`** generates a unique ID string (e.g., `"text-input-1"`). Within an `x-id` scope, same-name calls return the same ID. Optional second parameter appends a suffix for loop-based ID generation (useful for `aria-activedescendant` and similar attributes).

**`$event`** (not a separate page but documented) — available inside `x-on` handlers, provides the native browser event object.

---

## Global methods and the extensibility API

Alpine exposes several global methods on the `Alpine` object for configuration, state management, and extensibility.

**`Alpine.data(name, callback)`** registers a reusable component. The callback is a factory function returning a data object with properties, methods, getters, `init()`, and `destroy()`. Supports initial parameters: `x-data="dropdown(true)"`. Magic properties are accessible via `this` inside the object. Can encapsulate entire directive bundles using x-bind objects (trigger/dialogue pattern).

**`Alpine.store(name, data)`** creates a global reactive store accessible via `$store`. Supports object stores (with methods and an auto-called `init()`) and single-value primitive stores. Calling `Alpine.store(name)` without a second argument **retrieves** the store externally.

**`Alpine.bind(name, callback)`** registers a reusable set of attributes and directives that can be applied to elements via `x-bind="name"`.

**`Alpine.reactive(object)`** wraps a plain JavaScript object in a reactive Proxy. Changes to the proxy are tracked by Alpine's effect system.

**`Alpine.effect(callback)`** registers a reactive side effect. The callback runs immediately, and Alpine tracks all reactive data accessed within it. When any tracked dependency changes, the callback re-runs. Unlike the `effect` provided inside custom directives, `Alpine.effect()` persists — it is not tied to an element's lifecycle.

**`Alpine.directive(name, callback)`** registers a custom directive (`x-[name]`). The callback receives `(el, { value, modifiers, expression }, { Alpine, effect, cleanup, evaluate, evaluateLater })`. Key utilities: `evaluate(expression)` runs a JS expression once in the element's scope; `evaluateLater(expression)` compiles the expression into a reusable function for reactive use with `effect()`. The `cleanup()` function registers teardown logic. Chain `.before('directiveName')` to control execution order.

**`Alpine.magic(name, callback)`** registers a custom magic property/method (`$[name]`). The callback receives `(el, { Alpine })` and returns a value or a function. Under the hood, magics are getters — they re-evaluate on every access.

**`Alpine.plugin(callback)`** registers a plugin. A convenience wrapper that immediately invokes the callback with the `Alpine` global as its argument. Plugins are just functions that call `Alpine.directive()`, `Alpine.magic()`, etc.

**`Alpine.start()`** initializes Alpine on the page. Must be called exactly once. In NPM setups, all registration must happen before this call.

**`Alpine.morph(el, newHtml, options?)`** and **`Alpine.morphBetween(startMarker, endMarker, newHtml, options?)`** are provided by the Morph plugin (detailed below).

---

## The 9 official plugins and what each provides

Every plugin follows the same installation pattern: include its CDN script **before** the Alpine core script, or install via NPM and register with `Alpine.plugin()`.

### Mask — input formatting

Automatically formats text inputs as users type. The **`x-mask`** directive accepts a pattern string using wildcards: `9` (digit), `a` (letter), `*` (any character). Non-matching characters (slashes, dashes) are auto-inserted. **`x-mask:dynamic`** allows the mask to change based on current input — the `$input` magic holds the current value. The built-in **`$money($input, decimalSep?, thousandsSep?, precision?)`** helper handles currency formatting with configurable separators and precision.

### Intersect — viewport detection

A wrapper around the Intersection Observer API. **`x-intersect`** (alias: `x-intersect:enter`) fires when an element enters the viewport. **`x-intersect:leave`** fires when it exits. Modifiers: **`.once`** (trigger only once), **`.half`** (50% visible threshold), **`.full`** (99% visible), **`.threshold.{0-100}`** (custom threshold), **`.margin.{value}`** (expand/shrink the detection boundary, supports px and % values like CSS margin shorthand).

### Resize — element size observation

A wrapper around the Resize Observer API. **`x-resize`** fires when an element's dimensions change, providing **`$width`** and **`$height`** magic properties within the expression. The **`.document`** modifier observes the entire document instead.

### Persist — state persistence across page loads

Persists Alpine data in `localStorage`. The **`$persist(value)`** magic wraps a value in `x-data` to survive page reloads. Chainable methods: **`.as('customKey')`** sets a custom storage key (default is `_x_` + property name), **`.using(storageDriver)`** swaps the storage backend (e.g., `sessionStorage` or a custom cookie driver exposing `getItem`/`setItem`). **`Alpine.$persist`** is the global version for use with `Alpine.store()`.

### Focus — focus management and trapping

Manages focus for modals, dialogs, and keyboard navigation. **`x-trap`** traps focus within an element when its expression is true; focus returns to the previously focused element on release. Supports nesting. Modifiers: **`.inert`** (adds `aria-hidden="true"` to all other elements for accessibility), **`.noscroll`** (disables page scrolling while trapped), **`.noreturn`** (prevents returning focus on release), **`.noautofocus`** (prevents auto-focusing the first focusable element). The **`$focus`** magic exposes a rich API: `focus(el)`, `focusable(el)`, `focusables()`, `focused()`, `lastFocused()`, `within(el)`, `first()`, `last()`, `next()`, `previous()`, `noscroll()`, `wrap()`, `getFirst()`, `getLast()`, `getNext()`, `getPrevious()`.

### Collapse — height-based expand/collapse

Smooth expand/collapse animations by animating the `height` property. **`x-collapse`** must coexist with `x-show` on the same element. Modifiers: **`.duration.{time}`** (custom animation duration), **`.min.{height}px`** (collapse to a minimum height instead of zero, creating a "cut off" effect).

### Anchor — positioned elements

Anchors an element's position to another element using Floating UI. **`x-anchor`** accepts a DOM element reference (typically via `$refs`). Auto-flips positioning when insufficient space. **Position modifiers:** `.bottom`, `.bottom-start`, `.bottom-end`, `.top`, `.top-start`, `.top-end`, `.left`, `.left-start`, `.left-end`, `.right`, `.right-start`, `.right-end`. **`.offset.{px}`** adds a gap. **`.no-style`** disables auto-positioning, exposing **`$anchor.x`** and **`$anchor.y`** for manual styling.

### Morph — intelligent DOM patching

Morphs a live DOM element into new HTML while preserving browser and Alpine state (scroll position, input values, reactive data). **`Alpine.morph(el, newHtml, options?)`** patches a single element. **`Alpine.morphBetween(startMarker, endMarker, newHtml, options?)`** patches a range. Lifecycle hooks: `updating`, `updated`, `removing`, `removed`, `adding`, `added`, `key` (custom key function), and `lookahead` (boolean for optimized element reordering). Core infrastructure for frameworks like **Laravel Livewire**.

### Sort — drag-and-drop reordering

Enables drag-and-drop sorting powered by SortableJS. **`x-sort`** on a container makes its `x-sort:item` children draggable. **`x-sort:handle`** designates a drag handle within an item. **`x-sort:group="name"`** enables cross-list dragging between containers with matching group names. **`x-sort:ignore`** prevents specific child elements from initiating drag. **`x-sort:config`** passes custom SortableJS options. Provides **`$item`** (the sorted element's key) and **`$position`** (new index) in handler expressions. The **`.ghost`** modifier shows a translucent placeholder during drag (styled via `.sortable-ghost` CSS class). Alpine adds a `.sorting` class to `<body>` during drag operations.

---

## Reactivity system under the hood

Alpine's reactivity is built on two primitives: **`Alpine.reactive()`** creates a Proxy-wrapped object that intercepts get/set operations, and **`Alpine.effect()`** runs a callback that automatically re-executes whenever any reactive data it accessed changes. Every Alpine directive and magic property uses these same primitives internally — `x-text`, `x-show`, `x-bind`, and all others are thin wrappers around `effect()` calls that read reactive data and update the DOM accordingly. This means you could theoretically reconstruct all of Alpine's behavior using just `Alpine.reactive()` and `Alpine.effect()`.

---

## CSP-compatible build and async support

For environments with strict **Content Security Policy** headers that prohibit `unsafe-eval`, Alpine provides a separate CSP build (`@alpinejs/csp`). This build supports most common inline expressions — property access, arithmetic, comparisons, ternaries, assignments, method calls — but **cannot** handle complex expressions like arrow functions, template literals, destructuring, or global function calls (`console.log`, `Math.max`). The recommended pattern is extracting complex logic into `Alpine.data()` components.

Alpine natively supports **async functions** wherever standard functions work. You can use `await` in `x-init`, `x-text`, event handlers, and elsewhere. Alpine auto-detects async functions even without trailing parentheses.

---

## Conclusion

Alpine.js packs a remarkably complete toolkit into a minimal footprint. Its **18 directives** cover everything from state management and templating to conditional rendering, list iteration, transitions, teleportation, and accessibility-friendly ID generation. The **9 magic properties** provide DOM access, global state, event dispatch, reactive watching, and async-aware DOM timing. The **extensibility API** — `Alpine.directive()`, `Alpine.magic()`, and `Alpine.plugin()` — makes it infinitely composable. And the **9 official plugins** extend it into input masking, intersection observation, resize detection, state persistence, focus trapping, collapse animations, positioned elements, DOM morphing, and drag-and-drop sorting. For developers building server-rendered applications who need interactivity without the weight of a full SPA framework, Alpine.js delivers a uniquely pragmatic balance of power and simplicity.

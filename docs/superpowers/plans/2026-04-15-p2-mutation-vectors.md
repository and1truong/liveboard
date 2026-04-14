# P2 — Shared Mutation Vectors (Go ↔ TS Parity) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent permanent drift between the canonical Go mutation engine (`internal/board/board.go`) and the future TypeScript `LocalAdapter` (P3) by establishing a shared JSON vector suite that both languages must pass.

**Architecture:** Hand-maintain parallel implementations. Go already has `ApplyX` functions (P1). Add a TypeScript `applyOp(board, op): board` module that mirrors the Go semantics. A single `testdata/mutations/*.json` suite of `{board_before, op, board_after | expected_error}` vectors is consumed by a Go runner (pure in-memory, bypassing disk IO) and a TS runner (vitest). Adding a vector requires both runners green in CI.

**Tech Stack:** Go 1.24 (existing), TypeScript 5.x (new), vitest (new), pnpm (new — lightweight). No codegen. No runtime dependency on TS from Go or vice versa.

**Spec:** `docs/superpowers/specs/2026-04-15-iframe-renderer-architecture-design.md` §Go/TS parity
**Plan of plans:** `docs/superpowers/plans/2026-04-15-iframe-renderer-plan-of-plans.md`

**Out of scope for P2:**
- Shell, renderer, LocalAdapter runtime (P3+)
- RestAdapter (P5)
- Code generation from a single schema (spec says hand-maintain)
- Using the TS module anywhere in production (it's test-only until P3 imports it)
- `move_card_to_board` cross-board op (tracked separately; not in P1's `MutationOp` union)

**Conventions:**
- Go vectors live at `testdata/mutations/` at the repo root (reachable by both runners without symlink shenanigans).
- Each vector is one JSON file; filename matches op type + scenario (e.g. `add_card_happy.json`, `move_card_version_conflict.json`).
- TS lives under `web/shared/` — a new top-level TS project. pnpm workspace is not needed yet (single package).
- Commit convention matches P1: `feat(parity): ...`, `test(parity): ...`, `chore(ci): ...`.

---

## File structure

New files:
- `testdata/mutations/add_card_happy.json` (and ~25 more vectors — one per op + error cases)
- `internal/parity/runner_test.go` — Go vector runner
- `web/shared/package.json` — TS project root
- `web/shared/tsconfig.json`
- `web/shared/vitest.config.ts`
- `web/shared/.gitignore` — ignore `node_modules`, `dist`
- `web/shared/src/types.ts` — Board, Column, Card, BoardSettings, MutationOp type declarations
- `web/shared/src/boardOps.ts` — `applyOp(board: Board, op: MutationOp): Board` + 17 per-variant pure functions
- `web/shared/src/boardOps.test.ts` — vitest runner consuming the same JSON vectors
- `docs/parity.md` — how to add a vector, how both sides are tested
- `.github/workflows/ci.yml` — add TS test job

Modified:
- `internal/api/v1/mutations.go` — export `applyOp` as `Apply` so the parity runner can call it
- `internal/api/v1/mutations.go` + `_test.go` — update the one internal call site

---

## Task 1: Export `Apply` in the v1 package

**Files:**
- Modify: `internal/api/v1/mutations.go`

The Go vector runner needs a pure in-memory dispatcher that doesn't touch disk. `applyOp` already exists — rename it to `Apply` and export.

- [ ] **Step 1: Rename the function**

In `internal/api/v1/mutations.go`, find:

```go
func applyOp(b *models.Board, op MutationOp) error {
```

Rename to:

```go
// Apply mutates the board in-place according to op.
// This is the pure in-memory dispatcher — no disk IO, no locking, no version bump.
// The HTTP handler wraps this inside Engine.MutateBoard to add those concerns.
// Shared with the parity vector runner in internal/parity.
func Apply(b *models.Board, op MutationOp) error {
```

Update the one internal caller in the same file:

```go
func Dispatch(eng *board.Engine, boardPath string, clientVersion int, op MutationOp) (*models.Board, error) {
    var out *models.Board
    err := eng.MutateBoard(boardPath, clientVersion, func(b *models.Board) error {
        if e := Apply(b, op); e != nil {
            return e
        }
        out = b
        return nil
    })
    return out, err
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/api/v1/ -race`
Expected: PASS (internal-only rename, no behavior change).

- [ ] **Step 3: Run full suite**

Run: `go test ./... -race`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/api/v1/mutations.go
git commit -m "refactor(api): export Apply for parity runner"
```

---

## Task 2: Vector JSON schema + first happy-path vector

**Files:**
- Create: `testdata/mutations/add_card_happy.json`
- Create: `docs/parity.md`

Define the vector contract once, write one canonical example, document the shape.

- [ ] **Step 1: Create the vector directory**

```bash
mkdir -p testdata/mutations
```

- [ ] **Step 2: Write the first vector**

Create `testdata/mutations/add_card_happy.json`:

```json
{
  "name": "add_card_happy",
  "description": "Appending a card to an existing column increments length and preserves other columns.",
  "board_before": {
    "version": 1,
    "name": "Demo",
    "columns": [
      {
        "name": "Todo",
        "cards": [
          { "title": "existing" }
        ]
      },
      {
        "name": "Done",
        "cards": []
      }
    ]
  },
  "op": {
    "type": "add_card",
    "column": "Todo",
    "title": "added",
    "prepend": false
  },
  "board_after": {
    "version": 1,
    "name": "Demo",
    "columns": [
      {
        "name": "Todo",
        "cards": [
          { "title": "existing" },
          { "title": "added" }
        ]
      },
      {
        "name": "Done",
        "cards": []
      }
    ]
  }
}
```

**Notes:**
- `board_before` and `board_after` use the `pkg/models.Board` JSON shape (the same shape `GET /api/v1/boards/{slug}` returns).
- `version` is deliberately unchanged: `Apply` is the pure dispatcher; `MutateBoard` is what bumps version. Vectors test `Apply` only.
- Omit fields that equal the Go zero value and aren't meaningful to the test. The vector comparison is a structural JSON diff after round-trip; only populate what matters.

- [ ] **Step 3: Write docs/parity.md**

Create `docs/parity.md`:

````markdown
# Go ↔ TS Parity Vectors

Shared test suite that both the Go engine and the TypeScript `boardOps` module must pass. This is the guardrail that prevents drift between `internal/board/board.go` and `web/shared/src/boardOps.ts`.

## Running

Go:
```
go test ./internal/parity/ -race
```

TypeScript:
```
cd web/shared && pnpm install && pnpm test
```

CI runs both. A PR that adds or changes a vector must have both runners green.

## Vector format

One JSON file per scenario in `testdata/mutations/`.

```json
{
  "name": "add_card_happy",
  "description": "human-readable scenario description",
  "board_before": { "<pkg/models.Board JSON>" },
  "op": { "<MutationOp JSON>" },
  "board_after": { "<pkg/models.Board JSON>" }
}
```

For error cases, replace `board_after` with `expected_error`:

```json
{
  "name": "move_card_out_of_range",
  "board_before": { "...": "..." },
  "op": { "type": "move_card", "col_idx": 99, "card_idx": 0, "target_column": "Done" },
  "expected_error": "OUT_OF_RANGE"
}
```

Canonical error strings: `NOT_FOUND`, `OUT_OF_RANGE`, `INVALID`, `ALREADY_EXISTS`. (`VERSION_CONFLICT` is not reachable through `Apply` — that's checked by `MutateBoard`, not the pure dispatcher, and is not in scope for parity vectors.)

## Adding a vector

1. Create `testdata/mutations/<op>_<scenario>.json`.
2. Run `go test ./internal/parity/` — must pass.
3. Run `cd web/shared && pnpm test` — must pass.
4. If one side fails, the two implementations have diverged. Fix until both are green.

## What belongs in a vector

- One `MutationOp` applied to one `Board`.
- Deterministic result (no time, no random, no disk).
- Sensitive to the behavior the mutation is responsible for — if `add_card` is supposed to append, the vector must include a non-empty `board_before[columns][x].cards` so that "append" is distinguishable from "replace" or "prepend".

## What doesn't belong

- Version-conflict scenarios (checked by `MutateBoard`, not `Apply`).
- SSE or disk side effects (handled by the engine wrapper, not the pure dispatcher).
- Multi-board ops (`move_card_to_board` — out of scope; not in `MutationOp`).
````

- [ ] **Step 4: Commit**

```bash
git add testdata/mutations/add_card_happy.json docs/parity.md
git commit -m "test(parity): add vector schema and first happy-path vector"
```

---

## Task 3: Go vector runner

**Files:**
- Create: `internal/parity/runner_test.go`

Pure Go test that loads every JSON file in `testdata/mutations/`, runs it through `v1.Apply`, and asserts the result equals `board_after` (or the error code matches `expected_error`).

- [ ] **Step 1: Write the runner test**

Create `internal/parity/runner_test.go`:

```go
package parity_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/pkg/models"
)

type vector struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	BoardBefore   json.RawMessage `json:"board_before"`
	Op            json.RawMessage `json:"op"`
	BoardAfter    json.RawMessage `json:"board_after,omitempty"`
	ExpectedError string          `json:"expected_error,omitempty"`
}

func TestVectorSuite(t *testing.T) {
	// testdata/mutations lives at the repo root; tests run with CWD = package dir.
	root := filepath.Join("..", "..", "testdata", "mutations")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read testdata dir: %v", err)
	}

	found := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		found++
		path := filepath.Join(root, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			runVector(t, path)
		})
	}
	if found == 0 {
		t.Fatal("no vectors found")
	}
}

func runVector(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var vec vector
	if err := json.Unmarshal(raw, &vec); err != nil {
		t.Fatalf("parse vector: %v", err)
	}

	var b models.Board
	if err := json.Unmarshal(vec.BoardBefore, &b); err != nil {
		t.Fatalf("parse board_before: %v", err)
	}

	var op v1.MutationOp
	if err := json.Unmarshal(vec.Op, &op); err != nil {
		t.Fatalf("parse op: %v", err)
	}

	applyErr := v1.Apply(&b, op)

	if vec.ExpectedError != "" {
		if applyErr == nil {
			t.Fatalf("want error %q, got nil", vec.ExpectedError)
		}
		if got := sentinelCode(applyErr); got != vec.ExpectedError {
			t.Fatalf("want error %q, got %q (%v)", vec.ExpectedError, got, applyErr)
		}
		return
	}

	if applyErr != nil {
		t.Fatalf("unexpected error: %v", applyErr)
	}

	// Compare as JSON round-trip so slice-vs-nil differences normalize.
	gotJSON, err := json.Marshal(&b)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var want, got any
	if err := json.Unmarshal(vec.BoardAfter, &want); err != nil {
		t.Fatalf("parse board_after: %v", err)
	}
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatalf("re-parse result: %v", err)
	}

	if diff := jsonDiff(want, got); diff != "" {
		t.Errorf("board mismatch:\n%s", diff)
	}
}

func sentinelCode(err error) string {
	switch {
	case errors.Is(err, board.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, board.ErrOutOfRange):
		return "OUT_OF_RANGE"
	case errors.Is(err, board.ErrAlreadyExists):
		return "ALREADY_EXISTS"
	case errors.Is(err, board.ErrInvalidInput):
		return "INVALID"
	default:
		return "INTERNAL"
	}
}

// jsonDiff returns an empty string when want and got are structurally equal
// under Go's deep-equal of decoded JSON values, or a human-readable diff otherwise.
func jsonDiff(want, got any) string {
	wb, _ := json.MarshalIndent(want, "", "  ")
	gb, _ := json.MarshalIndent(got, "", "  ")
	if string(wb) == string(gb) {
		return ""
	}
	return "want:\n" + string(wb) + "\n\ngot:\n" + string(gb)
}
```

**Implementer notes:**
- Verify the exact sentinel names by grepping `internal/board/board.go` for `var Err`. If `ErrAlreadyExists` or `ErrInvalidInput` don't exist, drop that case. The plan assumes the P1 set: `ErrNotFound`, `ErrOutOfRange`, `ErrVersionConflict`, `ErrInvalidInput`, plus `workspace.ErrAlreadyExists`. Adapt `sentinelCode` to match what's actually defined.
- The relative path `../../testdata/mutations` works because Go tests run with CWD set to the package directory, and `internal/parity/` is two dirs deep.

- [ ] **Step 2: Run the test**

Run: `go test ./internal/parity/ -race -v`
Expected: PASS — one subtest `add_card_happy.json`.

- [ ] **Step 3: Commit**

```bash
git add internal/parity/runner_test.go
git commit -m "test(parity): add Go vector runner"
```

---

## Task 4: TypeScript project scaffold

**Files:**
- Create: `web/shared/package.json`
- Create: `web/shared/tsconfig.json`
- Create: `web/shared/vitest.config.ts`
- Create: `web/shared/.gitignore`

Minimal TS project. No framework, no bundler — just `tsc` for type-check and `vitest` for tests.

- [ ] **Step 1: Create package.json**

Create `web/shared/package.json`:

```json
{
  "name": "@liveboard/shared",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest",
    "typecheck": "tsc --noEmit"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "vitest": "^1.6.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json**

Create `web/shared/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ES2022",
    "moduleResolution": "bundler",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "esModuleInterop": true,
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "noEmit": true
  },
  "include": ["src/**/*"]
}
```

- [ ] **Step 3: Create vitest.config.ts**

Create `web/shared/vitest.config.ts`:

```ts
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    include: ['src/**/*.test.ts'],
  },
})
```

- [ ] **Step 4: Create .gitignore**

Create `web/shared/.gitignore`:

```
node_modules/
dist/
.vitest-cache/
```

- [ ] **Step 5: Install and verify tooling**

Run:
```bash
cd web/shared
pnpm install
pnpm typecheck
```

Expected: pnpm resolves deps; typecheck exits 0 (no TS files yet, nothing to check).

If pnpm is not installed, run `npm install -g pnpm` first. If the engineer prefers `npm`, the commands `npm install` / `npm test` / `npx tsc --noEmit` also work — but keep the `pnpm-lock.yaml` as the lockfile to match the plan.

- [ ] **Step 6: Commit**

```bash
git add web/shared/package.json web/shared/tsconfig.json web/shared/vitest.config.ts web/shared/.gitignore web/shared/pnpm-lock.yaml
git commit -m "chore(parity): scaffold web/shared TS project"
```

---

## Task 5: TypeScript types

**Files:**
- Create: `web/shared/src/types.ts`

Hand-maintained TS declarations that mirror `pkg/models/models.go` and `internal/api/v1/mutations.go`. Vectors are the guardrail; types are for editor ergonomics.

- [ ] **Step 1: Read the Go types**

Before writing, read:
- `/Users/htruong/code/htruong/liveboard/pkg/models/models.go` — `Board`, `Column`, `Card`, `BoardSettings`
- `/Users/htruong/code/htruong/liveboard/internal/api/v1/mutations.go` — every `*Op` struct and the `MutationOp` discriminator

Copy JSON tag names *exactly*.

- [ ] **Step 2: Write types.ts**

Create `web/shared/src/types.ts`:

```ts
// Hand-maintained mirror of pkg/models/models.go and internal/api/v1/mutations.go.
// Field names MUST match Go JSON tags. Vector tests catch drift.

export interface Board {
  version?: number
  name?: string
  description?: string
  icon?: string
  tags?: string[]
  members?: string[]
  list_collapse?: boolean[]
  settings?: BoardSettings
  columns?: Column[]
  file_path?: string
}

export interface Column {
  name: string
  cards: Card[]
  collapsed?: boolean
}

export interface Card {
  title: string
  body?: string
  tags?: string[]
  inline_tags?: string[]
  priority?: string
  due?: string
  assignee?: string
  completed?: boolean
  no_checkbox?: boolean
  metadata?: Record<string, string>
}

export interface BoardSettings {
  show_checkbox?: boolean | null
  card_position?: string | null
  expand_columns?: boolean | null
  view_mode?: string | null
  card_display_mode?: string | null
  week_start?: string | null
  color_theme?: string | null
}

// Tagged union — discriminator is `type`.
export type MutationOp =
  | { type: 'add_card'; column: string; title: string; prepend?: boolean }
  | { type: 'move_card'; col_idx: number; card_idx: number; target_column: string }
  | {
      type: 'reorder_card'
      col_idx: number
      card_idx: number
      before_idx: number
      target_column: string
    }
  | {
      type: 'edit_card'
      col_idx: number
      card_idx: number
      title: string
      body: string
      tags: string[]
      priority: string
      due: string
      assignee: string
    }
  | { type: 'delete_card'; col_idx: number; card_idx: number }
  | { type: 'complete_card'; col_idx: number; card_idx: number }
  | { type: 'tag_card'; col_idx: number; card_idx: number; tags: string[] }
  | { type: 'add_column'; name: string }
  | { type: 'rename_column'; old_name: string; new_name: string }
  | { type: 'delete_column'; name: string }
  | { type: 'move_column'; name: string; after_col: string }
  | { type: 'sort_column'; col_idx: number; sort_by: string }
  | { type: 'toggle_column_collapse'; col_idx: number }
  | { type: 'update_board_meta'; name: string; description: string; tags: string[] }
  | { type: 'update_board_members'; members: string[] }
  | { type: 'update_board_icon'; icon: string }
  | { type: 'update_board_settings'; settings: BoardSettings }

// Canonical error codes. Thrown by applyOp as Error instances with .code set.
export type ErrorCode = 'NOT_FOUND' | 'OUT_OF_RANGE' | 'INVALID' | 'ALREADY_EXISTS' | 'INTERNAL'

export class OpError extends Error {
  constructor(public code: ErrorCode, message: string) {
    super(message)
    this.name = 'OpError'
  }
}
```

- [ ] **Step 3: Typecheck**

Run:
```bash
cd web/shared && pnpm typecheck
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/types.ts
git commit -m "feat(parity): add TS types mirroring Go models"
```

---

## Task 6: TS `applyOp` dispatcher skeleton (no variants yet) + vector runner

**Files:**
- Create: `web/shared/src/boardOps.ts`
- Create: `web/shared/src/boardOps.test.ts`

Start with a dispatcher that throws for every op, plus a vitest runner that loads the same JSON vectors. Confirm the pipeline works end-to-end before filling in variants.

- [ ] **Step 1: Write boardOps.ts skeleton**

Create `web/shared/src/boardOps.ts`:

```ts
import type { Board, MutationOp } from './types.js'
import { OpError } from './types.js'

// Apply returns a new board with op applied. Input is not mutated.
// Mirrors internal/api/v1.Apply semantics. Shared parity vectors guard drift.
export function applyOp(board: Board, op: MutationOp): Board {
  // Structured clone keeps callers from seeing our in-place edits.
  const b: Board = structuredClone(board)

  switch (op.type) {
    default:
      // Exhaustiveness: TS compiler flags missing cases via `never`.
      throw new OpError('INTERNAL', `unimplemented op: ${(op as MutationOp).type}`)
  }
}
```

- [ ] **Step 2: Write the vitest runner**

Create `web/shared/src/boardOps.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { readFileSync, readdirSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { applyOp } from './boardOps.js'
import type { Board, MutationOp, ErrorCode } from './types.js'
import { OpError } from './types.js'

interface Vector {
  name: string
  description?: string
  board_before: Board
  op: MutationOp
  board_after?: Board
  expected_error?: ErrorCode
}

// __dirname-free path resolution. Vectors live at repo root /testdata/mutations.
// This file is at web/shared/src/boardOps.test.ts — four levels deep.
const vectorDir = resolve(process.cwd(), '..', '..', 'testdata', 'mutations')

const vectorFiles = readdirSync(vectorDir).filter((f) => f.endsWith('.json'))

describe('mutation vectors', () => {
  if (vectorFiles.length === 0) {
    it('finds vectors', () => {
      throw new Error(`no vectors in ${vectorDir}`)
    })
    return
  }

  for (const file of vectorFiles) {
    it(file, () => {
      const raw = readFileSync(join(vectorDir, file), 'utf8')
      const vec: Vector = JSON.parse(raw)

      if (vec.expected_error) {
        expect(() => applyOp(vec.board_before, vec.op)).toThrow(OpError)
        try {
          applyOp(vec.board_before, vec.op)
        } catch (e) {
          expect((e as OpError).code).toBe(vec.expected_error)
        }
        return
      }

      const got = applyOp(vec.board_before, vec.op)
      // Round-trip both sides through JSON so undefined-vs-missing normalize.
      expect(JSON.parse(JSON.stringify(got))).toEqual(
        JSON.parse(JSON.stringify(vec.board_after)),
      )
    })
  }
})
```

- [ ] **Step 3: Run vitest — expect the one vector to FAIL**

Run:
```bash
cd web/shared && pnpm test
```

Expected: one test `add_card_happy.json` fails (`unimplemented op: add_card`). That's correct — proves the runner loads vectors. Failure surface is now a TS implementation gap, which the next task closes.

- [ ] **Step 4: Commit**

```bash
git add web/shared/src/boardOps.ts web/shared/src/boardOps.test.ts
git commit -m "feat(parity): add TS applyOp skeleton and vector runner"
```

---

## Task 7: Implement `add_card` in TS + make the first vector pass

**Files:**
- Modify: `web/shared/src/boardOps.ts`

Fill in the first variant. End of this task: both runners green with the same vector.

- [ ] **Step 1: Implement add_card**

Replace the `switch (op.type)` block in `web/shared/src/boardOps.ts`:

```ts
  switch (op.type) {
    case 'add_card': {
      const col = (b.columns ?? []).find((c) => c.name === op.column)
      if (!col) throw new OpError('NOT_FOUND', `column ${op.column}`)
      const card = { title: op.title }
      if (op.prepend) {
        col.cards = [card, ...(col.cards ?? [])]
      } else {
        col.cards = [...(col.cards ?? []), card]
      }
      return b
    }
    default:
      throw new OpError('INTERNAL', `unimplemented op: ${(op as MutationOp).type}`)
  }
```

- [ ] **Step 2: Run both runners**

Run:
```bash
cd web/shared && pnpm test
```

Expected: `add_card_happy.json` passes.

Run:
```bash
cd /Users/htruong/code/htruong/liveboard && go test ./internal/parity/ -race
```

Expected: passes (already did in Task 3).

- [ ] **Step 3: Commit**

```bash
git add web/shared/src/boardOps.ts
git commit -m "feat(parity): implement add_card in TS"
```

---

## Task 8: Card-index ops — TS impl + vectors

**Files:**
- Modify: `web/shared/src/boardOps.ts`
- Create: `testdata/mutations/move_card_happy.json`
- Create: `testdata/mutations/reorder_card_same_column.json`
- Create: `testdata/mutations/reorder_card_cross_column.json`
- Create: `testdata/mutations/edit_card_happy.json`
- Create: `testdata/mutations/delete_card_happy.json`
- Create: `testdata/mutations/complete_card_happy.json`
- Create: `testdata/mutations/tag_card_happy.json`

These six ops all take `col_idx` / `card_idx` (index-based). Implement them together because they share the same lookup pattern.

- [ ] **Step 1: Read the Go source**

Read `/Users/htruong/code/htruong/liveboard/internal/board/board.go` — specifically `ApplyMoveCard`, `ApplyReorderCard`, `ApplyEditCard`, `ApplyDeleteCard`, `ApplyCompleteCard`, `ApplyTagCard`. Mirror each body in TS. Note the bounds-check error paths — those map to `OUT_OF_RANGE` and `NOT_FOUND`.

- [ ] **Step 2: Write vectors**

Create `testdata/mutations/move_card_happy.json`:

```json
{
  "name": "move_card_happy",
  "description": "Move first card from column 0 to 'Done' — appends there, removes here.",
  "board_before": {
    "version": 1,
    "columns": [
      { "name": "Todo", "cards": [{ "title": "a" }, { "title": "b" }] },
      { "name": "Done", "cards": [] }
    ]
  },
  "op": { "type": "move_card", "col_idx": 0, "card_idx": 0, "target_column": "Done" },
  "board_after": {
    "version": 1,
    "columns": [
      { "name": "Todo", "cards": [{ "title": "b" }] },
      { "name": "Done", "cards": [{ "title": "a" }] }
    ]
  }
}
```

Create `testdata/mutations/reorder_card_same_column.json`:

```json
{
  "name": "reorder_card_same_column",
  "description": "Move card from idx 2 to idx 0 within Todo.",
  "board_before": {
    "columns": [
      {
        "name": "Todo",
        "cards": [{ "title": "a" }, { "title": "b" }, { "title": "c" }]
      }
    ]
  },
  "op": {
    "type": "reorder_card",
    "col_idx": 0,
    "card_idx": 2,
    "before_idx": 0,
    "target_column": "Todo"
  },
  "board_after": {
    "columns": [
      {
        "name": "Todo",
        "cards": [{ "title": "c" }, { "title": "a" }, { "title": "b" }]
      }
    ]
  }
}
```

Create `testdata/mutations/reorder_card_cross_column.json`:

```json
{
  "name": "reorder_card_cross_column",
  "description": "Move card from Todo idx 0 to Done idx 0.",
  "board_before": {
    "columns": [
      { "name": "Todo", "cards": [{ "title": "a" }, { "title": "b" }] },
      { "name": "Done", "cards": [{ "title": "x" }] }
    ]
  },
  "op": {
    "type": "reorder_card",
    "col_idx": 0,
    "card_idx": 0,
    "before_idx": 0,
    "target_column": "Done"
  },
  "board_after": {
    "columns": [
      { "name": "Todo", "cards": [{ "title": "b" }] },
      { "name": "Done", "cards": [{ "title": "a" }, { "title": "x" }] }
    ]
  }
}
```

Create `testdata/mutations/edit_card_happy.json`:

```json
{
  "name": "edit_card_happy",
  "description": "Full field replace on an existing card.",
  "board_before": {
    "columns": [
      {
        "name": "Todo",
        "cards": [
          { "title": "old title", "body": "old body" }
        ]
      }
    ]
  },
  "op": {
    "type": "edit_card",
    "col_idx": 0,
    "card_idx": 0,
    "title": "new title",
    "body": "new body",
    "tags": ["a", "b"],
    "priority": "high",
    "due": "2026-04-30",
    "assignee": "alice"
  },
  "board_after": {
    "columns": [
      {
        "name": "Todo",
        "cards": [
          {
            "title": "new title",
            "body": "new body",
            "tags": ["a", "b"],
            "priority": "high",
            "due": "2026-04-30",
            "assignee": "alice"
          }
        ]
      }
    ]
  }
}
```

**Note:** the `board_after` shape must exactly match what `ApplyEditCard` produces, including field ordering of `tags`/`inline_tags` if that function re-extracts inline tags from the title. Read the Go source and adjust — if `edit_card` preserves the title as-is, use "new title"; if it strips/re-extracts inline tags, include only a title without `#`.

Create `testdata/mutations/delete_card_happy.json`:

```json
{
  "name": "delete_card_happy",
  "description": "Delete middle card; others shift up.",
  "board_before": {
    "columns": [
      {
        "name": "Todo",
        "cards": [{ "title": "a" }, { "title": "b" }, { "title": "c" }]
      }
    ]
  },
  "op": { "type": "delete_card", "col_idx": 0, "card_idx": 1 },
  "board_after": {
    "columns": [
      { "name": "Todo", "cards": [{ "title": "a" }, { "title": "c" }] }
    ]
  }
}
```

Create `testdata/mutations/complete_card_happy.json`:

```json
{
  "name": "complete_card_happy",
  "description": "Toggling an incomplete card marks it complete.",
  "board_before": {
    "columns": [{ "name": "Todo", "cards": [{ "title": "a" }] }]
  },
  "op": { "type": "complete_card", "col_idx": 0, "card_idx": 0 },
  "board_after": {
    "columns": [
      { "name": "Todo", "cards": [{ "title": "a", "completed": true }] }
    ]
  }
}
```

Create `testdata/mutations/tag_card_happy.json`:

```json
{
  "name": "tag_card_happy",
  "description": "Replace tags on a card.",
  "board_before": {
    "columns": [
      { "name": "Todo", "cards": [{ "title": "a", "tags": ["old"] }] }
    ]
  },
  "op": { "type": "tag_card", "col_idx": 0, "card_idx": 0, "tags": ["new", "set"] },
  "board_after": {
    "columns": [
      { "name": "Todo", "cards": [{ "title": "a", "tags": ["new", "set"] }] }
    ]
  }
}
```

- [ ] **Step 3: Run Go runner against new vectors**

Run:
```bash
go test ./internal/parity/ -race -v
```

Expected: all pass. If one fails, the vector's `board_after` doesn't match Go's actual output — fix the vector to match Go (Go is canonical).

- [ ] **Step 4: Implement card-index ops in TS**

Extend `web/shared/src/boardOps.ts` switch with the six new cases. Mirror Go semantics:

```ts
    case 'move_card': {
      const cols = b.columns ?? []
      const src = cols[op.col_idx]
      if (!src) throw new OpError('OUT_OF_RANGE', `col_idx ${op.col_idx}`)
      if (op.card_idx < 0 || op.card_idx >= (src.cards?.length ?? 0)) {
        throw new OpError('OUT_OF_RANGE', `card_idx ${op.card_idx}`)
      }
      const dst = cols.find((c) => c.name === op.target_column)
      if (!dst) throw new OpError('NOT_FOUND', `column ${op.target_column}`)
      const [card] = src.cards!.splice(op.card_idx, 1)
      dst.cards = [...(dst.cards ?? []), card!]
      return b
    }
    case 'reorder_card': {
      const cols = b.columns ?? []
      const src = cols[op.col_idx]
      if (!src) throw new OpError('OUT_OF_RANGE', `col_idx ${op.col_idx}`)
      if (op.card_idx < 0 || op.card_idx >= (src.cards?.length ?? 0)) {
        throw new OpError('OUT_OF_RANGE', `card_idx ${op.card_idx}`)
      }
      const dst = cols.find((c) => c.name === op.target_column)
      if (!dst) throw new OpError('NOT_FOUND', `column ${op.target_column}`)
      const [card] = src.cards!.splice(op.card_idx, 1)
      const target = dst.cards ?? (dst.cards = [])
      const insertAt = Math.max(0, Math.min(op.before_idx, target.length))
      target.splice(insertAt, 0, card!)
      return b
    }
    case 'edit_card': {
      const card = cardAt(b, op.col_idx, op.card_idx)
      card.title = op.title
      card.body = op.body
      card.tags = op.tags
      card.priority = op.priority
      card.due = op.due
      card.assignee = op.assignee
      return b
    }
    case 'delete_card': {
      const col = colAt(b, op.col_idx)
      if (op.card_idx < 0 || op.card_idx >= (col.cards?.length ?? 0)) {
        throw new OpError('OUT_OF_RANGE', `card_idx ${op.card_idx}`)
      }
      col.cards!.splice(op.card_idx, 1)
      return b
    }
    case 'complete_card': {
      const card = cardAt(b, op.col_idx, op.card_idx)
      card.completed = !card.completed
      return b
    }
    case 'tag_card': {
      const card = cardAt(b, op.col_idx, op.card_idx)
      card.tags = op.tags
      return b
    }
```

Add at the bottom of the file:

```ts
function colAt(b: Board, idx: number) {
  const col = (b.columns ?? [])[idx]
  if (!col) throw new OpError('OUT_OF_RANGE', `col_idx ${idx}`)
  return col
}

function cardAt(b: Board, colIdx: number, cardIdx: number) {
  const col = colAt(b, colIdx)
  const card = (col.cards ?? [])[cardIdx]
  if (!card) throw new OpError('OUT_OF_RANGE', `card_idx ${cardIdx}`)
  return card
}
```

**Caveat for `complete_card`:** Go's `ApplyCompleteCard` toggles the `Completed` boolean. If the vector starts with `completed: undefined`, the toggle sets it to `true`. Go JSON-marshals `false` as an absent field (because of `omitempty`), so the vector's `board_after` must omit `completed` when it's false. Match Go's exact output.

- [ ] **Step 5: Run TS vitest**

Run:
```bash
cd web/shared && pnpm test
```

Expected: all seven new vectors pass + `add_card_happy` still passes.

- [ ] **Step 6: Commit**

```bash
git add web/shared/src/boardOps.ts testdata/mutations/
git commit -m "feat(parity): implement card-index ops in TS + vectors"
```

---

## Task 9: Column ops — TS impl + vectors

**Files:**
- Modify: `web/shared/src/boardOps.ts`
- Create: `testdata/mutations/add_column_happy.json`
- Create: `testdata/mutations/rename_column_happy.json`
- Create: `testdata/mutations/delete_column_happy.json`
- Create: `testdata/mutations/move_column_happy.json`
- Create: `testdata/mutations/sort_column_by_priority.json`
- Create: `testdata/mutations/toggle_column_collapse_happy.json`

- [ ] **Step 1: Read the Go source**

Read `ApplyAddColumn`, `ApplyRenameColumn`, `ApplyDeleteColumn`, `ApplyMoveColumn`, `ApplySortColumn`, `ApplyToggleColumnCollapse` in `internal/board/board.go`. Pay attention to:
- `ApplyMoveColumn` — does `after_col=""` mean "move to front"? Check.
- `ApplySortColumn` — what are the valid `sort_by` values? `priority`, `due`, `title`, `assignee`? Order semantics (stable? priority order mapping)?
- `ApplyToggleColumnCollapse` — writes to `board.list_collapse[col_idx]`, not to the column struct directly. Mirror exactly.

- [ ] **Step 2: Write vectors**

Create `testdata/mutations/add_column_happy.json`:

```json
{
  "name": "add_column_happy",
  "board_before": { "columns": [{ "name": "Todo", "cards": [] }] },
  "op": { "type": "add_column", "name": "Doing" },
  "board_after": {
    "columns": [
      { "name": "Todo", "cards": [] },
      { "name": "Doing", "cards": [] }
    ]
  }
}
```

Create `testdata/mutations/rename_column_happy.json`:

```json
{
  "name": "rename_column_happy",
  "board_before": { "columns": [{ "name": "Todo", "cards": [] }] },
  "op": { "type": "rename_column", "old_name": "Todo", "new_name": "Backlog" },
  "board_after": { "columns": [{ "name": "Backlog", "cards": [] }] }
}
```

Create `testdata/mutations/delete_column_happy.json`:

```json
{
  "name": "delete_column_happy",
  "board_before": {
    "columns": [
      { "name": "Todo", "cards": [] },
      { "name": "Done", "cards": [] }
    ]
  },
  "op": { "type": "delete_column", "name": "Done" },
  "board_after": { "columns": [{ "name": "Todo", "cards": [] }] }
}
```

Create `testdata/mutations/move_column_happy.json`:

```json
{
  "name": "move_column_happy",
  "description": "Move 'Done' after 'Todo' when it was already there — no-op equivalent for simple case. Check Go semantics before finalizing after_col value.",
  "board_before": {
    "columns": [
      { "name": "Todo", "cards": [] },
      { "name": "Doing", "cards": [] },
      { "name": "Done", "cards": [] }
    ]
  },
  "op": { "type": "move_column", "name": "Done", "after_col": "Todo" },
  "board_after": {
    "columns": [
      { "name": "Todo", "cards": [] },
      { "name": "Done", "cards": [] },
      { "name": "Doing", "cards": [] }
    ]
  }
}
```

**Implementer:** run this vector through the Go runner first; if Go produces a different `board_after`, update the vector to match Go. This is the canonical order.

Create `testdata/mutations/sort_column_by_priority.json`:

```json
{
  "name": "sort_column_by_priority",
  "description": "Sort cards in column 0 by priority (critical > high > medium > low). Implementer must confirm the exact ordering Go uses before pinning board_after.",
  "board_before": {
    "columns": [
      {
        "name": "Todo",
        "cards": [
          { "title": "c", "priority": "low" },
          { "title": "a", "priority": "critical" },
          { "title": "b", "priority": "medium" }
        ]
      }
    ]
  },
  "op": { "type": "sort_column", "col_idx": 0, "sort_by": "priority" },
  "board_after": {
    "columns": [
      {
        "name": "Todo",
        "cards": [
          { "title": "a", "priority": "critical" },
          { "title": "b", "priority": "medium" },
          { "title": "c", "priority": "low" }
        ]
      }
    ]
  }
}
```

Create `testdata/mutations/toggle_column_collapse_happy.json`:

```json
{
  "name": "toggle_column_collapse_happy",
  "description": "Toggle collapse flag for col_idx=1 in a board with three columns.",
  "board_before": {
    "columns": [
      { "name": "A", "cards": [] },
      { "name": "B", "cards": [] },
      { "name": "C", "cards": [] }
    ],
    "list_collapse": [false, false, false]
  },
  "op": { "type": "toggle_column_collapse", "col_idx": 1 },
  "board_after": {
    "columns": [
      { "name": "A", "cards": [] },
      { "name": "B", "cards": [] },
      { "name": "C", "cards": [] }
    ],
    "list_collapse": [false, true, false]
  }
}
```

- [ ] **Step 3: Run Go runner; reconcile**

```bash
go test ./internal/parity/ -race -v
```

For any vector where Go disagrees, update the vector to match Go's output (Go is canonical). In particular `move_column_happy` and `sort_column_by_priority` are most likely to need reconciliation.

- [ ] **Step 4: Implement column ops in TS**

Extend the switch in `web/shared/src/boardOps.ts`:

```ts
    case 'add_column': {
      b.columns = [...(b.columns ?? []), { name: op.name, cards: [] }]
      return b
    }
    case 'rename_column': {
      const col = (b.columns ?? []).find((c) => c.name === op.old_name)
      if (!col) throw new OpError('NOT_FOUND', `column ${op.old_name}`)
      col.name = op.new_name
      return b
    }
    case 'delete_column': {
      const cols = b.columns ?? []
      const idx = cols.findIndex((c) => c.name === op.name)
      if (idx < 0) throw new OpError('NOT_FOUND', `column ${op.name}`)
      cols.splice(idx, 1)
      return b
    }
    case 'move_column': {
      const cols = b.columns ?? []
      const srcIdx = cols.findIndex((c) => c.name === op.name)
      if (srcIdx < 0) throw new OpError('NOT_FOUND', `column ${op.name}`)
      const [col] = cols.splice(srcIdx, 1)
      const afterIdx = op.after_col === '' ? -1 : cols.findIndex((c) => c.name === op.after_col)
      if (op.after_col !== '' && afterIdx < 0) {
        throw new OpError('NOT_FOUND', `column ${op.after_col}`)
      }
      cols.splice(afterIdx + 1, 0, col!)
      return b
    }
    case 'sort_column': {
      const col = colAt(b, op.col_idx)
      col.cards = sortCards(col.cards ?? [], op.sort_by)
      return b
    }
    case 'toggle_column_collapse': {
      colAt(b, op.col_idx) // bounds-check
      const flags = b.list_collapse ?? []
      while (flags.length <= op.col_idx) flags.push(false)
      flags[op.col_idx] = !flags[op.col_idx]
      b.list_collapse = flags
      return b
    }
```

Add at the bottom:

```ts
function sortCards(cards: Card[], sortBy: string): Card[] {
  // Match Go's ApplySortColumn. Read internal/board/board.go for the exact
  // comparator. At minimum, priority order is: critical, high, medium, low, "".
  const priorityRank: Record<string, number> = {
    critical: 0,
    high: 1,
    medium: 2,
    low: 3,
    '': 4,
  }
  const copy = [...cards]
  switch (sortBy) {
    case 'priority':
      copy.sort((a, b) =>
        (priorityRank[a.priority ?? ''] ?? 5) - (priorityRank[b.priority ?? ''] ?? 5))
      return copy
    case 'due':
      copy.sort((a, b) => (a.due ?? '').localeCompare(b.due ?? ''))
      return copy
    case 'title':
      copy.sort((a, b) => a.title.localeCompare(b.title))
      return copy
    case 'assignee':
      copy.sort((a, b) => (a.assignee ?? '').localeCompare(b.assignee ?? ''))
      return copy
    default:
      throw new OpError('INVALID', `unknown sort_by: ${sortBy}`)
  }
}
```

Add `Card` to the import in boardOps.ts if not already imported.

**Implementer:** after a first green pass, read `ApplySortColumn` in `internal/board/board.go` and confirm the TS comparators match Go byte-for-byte (stable sort? tiebreakers?). The vector test will catch mismatches, but understanding *why* Go chose a specific order makes drift prevention cheaper.

- [ ] **Step 5: Run both runners**

```bash
go test ./internal/parity/ -race
cd web/shared && pnpm test
```

Expected: all vectors pass on both sides.

- [ ] **Step 6: Commit**

```bash
git add web/shared/src/boardOps.ts testdata/mutations/
git commit -m "feat(parity): implement column ops in TS + vectors"
```

---

## Task 10: Board-level ops — TS impl + vectors

**Files:**
- Modify: `web/shared/src/boardOps.ts`
- Create: `testdata/mutations/update_board_meta_happy.json`
- Create: `testdata/mutations/update_board_members_happy.json`
- Create: `testdata/mutations/update_board_icon_happy.json`
- Create: `testdata/mutations/update_board_settings_happy.json`

Last four ops; they're whole-field replaces on the board struct.

- [ ] **Step 1: Write vectors**

Create `testdata/mutations/update_board_meta_happy.json`:

```json
{
  "name": "update_board_meta_happy",
  "board_before": { "name": "Old", "description": "d", "tags": ["x"] },
  "op": {
    "type": "update_board_meta",
    "name": "New",
    "description": "new desc",
    "tags": ["a", "b"]
  },
  "board_after": { "name": "New", "description": "new desc", "tags": ["a", "b"] }
}
```

Create `testdata/mutations/update_board_members_happy.json`:

```json
{
  "name": "update_board_members_happy",
  "board_before": { "members": ["alice"] },
  "op": { "type": "update_board_members", "members": ["bob", "carol"] },
  "board_after": { "members": ["bob", "carol"] }
}
```

Create `testdata/mutations/update_board_icon_happy.json`:

```json
{
  "name": "update_board_icon_happy",
  "board_before": { "icon": "🚀" },
  "op": { "type": "update_board_icon", "icon": "🎯" },
  "board_after": { "icon": "🎯" }
}
```

Create `testdata/mutations/update_board_settings_happy.json`:

```json
{
  "name": "update_board_settings_happy",
  "description": "Settings replace: new struct wins, nil fields clear overrides.",
  "board_before": {
    "settings": { "show_checkbox": true, "view_mode": "board" }
  },
  "op": {
    "type": "update_board_settings",
    "settings": { "view_mode": "calendar" }
  },
  "board_after": {
    "settings": { "view_mode": "calendar" }
  }
}
```

- [ ] **Step 2: Run Go runner**

```bash
go test ./internal/parity/ -race -v
```

Expected: pass. Reconcile `board_after` against Go output if anything diverges. The settings vector in particular: Go's `BoardSettings` uses pointer fields, so a "missing" field in JSON unmarshals to `nil`. The round-trip comparison should normalize to the same shape.

- [ ] **Step 3: Implement in TS**

Extend switch:

```ts
    case 'update_board_meta': {
      b.name = op.name
      b.description = op.description
      b.tags = op.tags
      return b
    }
    case 'update_board_members': {
      b.members = op.members
      return b
    }
    case 'update_board_icon': {
      b.icon = op.icon
      return b
    }
    case 'update_board_settings': {
      b.settings = op.settings
      return b
    }
```

- [ ] **Step 4: Verify exhaustiveness**

The `default` branch should now be unreachable. TypeScript's narrowing will flag a missing case if one slipped. Ensure `pnpm typecheck` passes.

- [ ] **Step 5: Run both runners**

```bash
go test ./internal/parity/ -race
cd web/shared && pnpm test
```

Expected: all 18 vectors pass.

- [ ] **Step 6: Commit**

```bash
git add web/shared/src/boardOps.ts testdata/mutations/
git commit -m "feat(parity): implement board-level ops in TS + vectors"
```

---

## Task 11: Error vectors

**Files:**
- Create: `testdata/mutations/add_card_column_not_found.json`
- Create: `testdata/mutations/move_card_out_of_range.json`
- Create: `testdata/mutations/rename_column_not_found.json`
- Create: `testdata/mutations/delete_card_out_of_range.json`
- Create: `testdata/mutations/sort_column_invalid_sort_by.json`

At least one error case per error code to lock the contract.

- [ ] **Step 1: Write vectors**

Create `testdata/mutations/add_card_column_not_found.json`:

```json
{
  "name": "add_card_column_not_found",
  "board_before": { "columns": [{ "name": "Todo", "cards": [] }] },
  "op": { "type": "add_card", "column": "Nope", "title": "x" },
  "expected_error": "NOT_FOUND"
}
```

Create `testdata/mutations/move_card_out_of_range.json`:

```json
{
  "name": "move_card_out_of_range",
  "board_before": {
    "columns": [{ "name": "Todo", "cards": [{ "title": "a" }] }]
  },
  "op": { "type": "move_card", "col_idx": 0, "card_idx": 99, "target_column": "Todo" },
  "expected_error": "OUT_OF_RANGE"
}
```

Create `testdata/mutations/rename_column_not_found.json`:

```json
{
  "name": "rename_column_not_found",
  "board_before": { "columns": [{ "name": "Todo", "cards": [] }] },
  "op": { "type": "rename_column", "old_name": "Missing", "new_name": "X" },
  "expected_error": "NOT_FOUND"
}
```

Create `testdata/mutations/delete_card_out_of_range.json`:

```json
{
  "name": "delete_card_out_of_range",
  "board_before": { "columns": [{ "name": "Todo", "cards": [] }] },
  "op": { "type": "delete_card", "col_idx": 0, "card_idx": 0 },
  "expected_error": "OUT_OF_RANGE"
}
```

Create `testdata/mutations/sort_column_invalid_sort_by.json`:

```json
{
  "name": "sort_column_invalid_sort_by",
  "board_before": { "columns": [{ "name": "Todo", "cards": [] }] },
  "op": { "type": "sort_column", "col_idx": 0, "sort_by": "not_a_real_key" },
  "expected_error": "INVALID"
}
```

**Implementer:** the `INVALID` mapping is the softest — check that Go's `ApplySortColumn` returns an error wrapping `board.ErrInvalidInput` for an unknown sort key. If Go silently no-ops or returns a different error, change the vector (or accept that this one test stays out of the suite).

- [ ] **Step 2: Run both runners**

```bash
go test ./internal/parity/ -race -v
cd web/shared && pnpm test
```

Expected: all error vectors pass; `sentinelCode` in Go maps cleanly, and `OpError.code` in TS matches.

- [ ] **Step 3: Commit**

```bash
git add testdata/mutations/
git commit -m "test(parity): add error-case vectors"
```

---

## Task 12: CI integration

**Files:**
- Modify: `.github/workflows/ci.yml`

Add a job that runs the TS test suite. Gate merges on both Go and TS parity.

- [ ] **Step 1: Add the TS job**

Append to `.github/workflows/ci.yml` after the existing `test` job:

```yaml
  ts-parity:
    name: TS Parity Tests
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v6

      - name: Setup pnpm
        uses: pnpm/action-setup@v4
        with:
          version: 9

      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: pnpm
          cache-dependency-path: web/shared/pnpm-lock.yaml

      - name: Install TS deps
        working-directory: web/shared
        run: pnpm install --frozen-lockfile

      - name: Typecheck
        working-directory: web/shared
        run: pnpm typecheck

      - name: Run vitest
        working-directory: web/shared
        run: pnpm test
```

- [ ] **Step 2: Verify locally that the commands work**

Run:
```bash
cd /Users/htruong/code/htruong/liveboard/web/shared
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
```

Expected: all pass. If `--frozen-lockfile` fails, commit the current `pnpm-lock.yaml` first.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "chore(ci): add TS parity test job"
```

---

## Task 13: README pointer to parity docs

**Files:**
- Modify: `README.md`

One line so future contributors find the parity doc.

- [ ] **Step 1: Find the right place**

Look for the section that already links to `docs/api/v1.md` (added in P1 Task 13). Add the parity doc link next to it.

- [ ] **Step 2: Add the link**

Something like:

```markdown
- REST API reference: [`docs/api/v1.md`](docs/api/v1.md)
- Go/TS parity vectors: [`docs/parity.md`](docs/parity.md)
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: link parity vectors from README"
```

---

## Task 14: Final verification

- [ ] **Step 1: Full Go suite with race detector**

Run:
```bash
go test ./... -race
```
Expected: all green.

- [ ] **Step 2: Go lint**

Run:
```bash
make lint
```
Expected: no new lint issues introduced by P2 files. (P2 adds only one Go file — `internal/parity/runner_test.go` — and renames one function. If lint flags something new, fix it.)

- [ ] **Step 3: TS typecheck + tests**

Run:
```bash
cd web/shared && pnpm typecheck && pnpm test
```
Expected: green.

- [ ] **Step 4: Vector coverage audit**

Confirm the vector suite covers every `MutationOp` variant:

- `add_card`, `move_card`, `reorder_card` (same + cross), `edit_card`, `delete_card`, `complete_card`, `tag_card`
- `add_column`, `rename_column`, `delete_column`, `move_column`, `sort_column`, `toggle_column_collapse`
- `update_board_meta`, `update_board_members`, `update_board_icon`, `update_board_settings`
- Error cases: `NOT_FOUND`, `OUT_OF_RANGE`, `INVALID` each have at least one vector

If a variant is missing a vector, add one before calling P2 done.

- [ ] **Step 5: Open PR**

Title: `feat(parity): P2 Go/TS mutation vector suite`
Body: link to the spec and plan-of-plans, summarize the vector count and what CI now checks, note that P3 can begin once this merges.

---

## Spec coverage check

Plan-of-plans P2 scope → task mapping:

- Define `MutationOp` union shape → already done in P1 (Task 6 of P1). P2 reuses it via the exported `Apply`.
- Go types matching schema → reused from `pkg/models` + P1 `MutationOp`. Task 1 exports `Apply`.
- TS types in shared module → Task 5.
- `boardOps.ts` pure functions → Tasks 6–10.
- Vector suite → Tasks 2, 8, 9, 10, 11.
- Go test runner → Task 3.
- TS test runner → Task 6.
- CI both runners → Task 12.

No gaps.

## Known follow-ups (not P2)

- `move_card_to_board` — cross-board op; not in `MutationOp`. If P4 needs it, add the Go Apply variant, the TS mirror, and vectors as a dedicated follow-up.
- Property-based testing — could generate vectors automatically. Nice future addition but the hand-maintained vectors are the contract.
- Codegen from a single schema — explicitly deferred by the spec; vectors are the guarantee instead.
- Normalization of `inline_tags` extracted from card titles — if `ApplyEditCard` re-runs title parsing, that's a behavior the vectors should cover. Add a vector specifically for a title containing `#inline` tags if this turns out to matter.

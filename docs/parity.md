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

## Adapters

The `LocalAdapter` at `web/shared/src/adapters/local.ts` consumes `applyOp` from the parity module. Its tests (`local.test.ts`) verify adapter-specific behavior (version conflicts, seed, BroadcastChannel); the underlying mutation correctness is covered by the vector suite above.

## Renderer

The default React renderer at `web/renderer/default/` consumes the Client from `web/shared/src/client.ts`. It is a read-only board viewer (P4a scope). Mutations come in P4b. Component-level tests live alongside each component and cover the query-invalidation path end-to-end against a stubbed in-memory Broker.

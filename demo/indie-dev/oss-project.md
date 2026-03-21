---
name: OSS Tracker
description: "liveboard/liveboard — open source project board"
icon: "\U0001F4E6"
tags:
    - oss
    - github
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: board
---

## Triage

- [ ] Bug: cards reorder incorrectly when column is collapsed
  tags: bug, frontend
  priority: high
  Reported in #182. Drag-and-drop index calculation is off when the source column is collapsed. Steps to reproduce in the issue.

- [ ] Feature request: export board to JSON
  tags: feature, api
  priority: medium
  Issue #175 — multiple users asking for this. Should be straightforward since we already have the models.

- [ ] Bug: SSE connection drops after 30min idle
  tags: bug, backend
  priority: medium
  Issue #191. Browser closes the EventSource. Need server-side keepalive pings.

- [ ] Feature request: keyboard shortcuts for card actions
  tags: feature, ux
  priority: low
  Issue #168. Vim-style navigation requested. `j`/`k` to move between cards, `e` to edit, `d` to toggle done.

## Accepted

- [ ] Add Markdown table support in card bodies
  tags: feature, parser
  assignee: contributor-alex
  priority: medium
  Issue #156. Parser currently strips tables. Need to preserve them in roundtrip.

- [ ] ARM64 Docker image
  tags: infra, docker
  priority: high
  Issue #160. M1/M2 Mac users can't use the published image. Add multi-arch build to CI.

- [ ] Improve error messages for malformed frontmatter
  tags: dx, parser
  assignee: contributor-sam
  priority: low
  Issue #163. Currently panics on invalid YAML. Should return a helpful parse error with line number.

## In Progress

- [ ] WebSocket support as SSE alternative
  tags: feature, backend
  assignee: contributor-alex
  priority: medium
  PR #189. Draft PR up, needs tests. Optional flag `--transport=ws` to switch from SSE.

- [ ] Add board-level access control
  tags: feature, auth
  assignee: hieu
  priority: high
  PR #192. Per-board read/write permissions stored in frontmatter. Working on middleware.

## Released

- [x] v0.9.2 — Fix concurrent write race condition
  tags: bugfix, backend
  Mutex was per-process, not per-board. Fixed with sync.Map keyed by board path.

- [x] v0.9.1 — Add column sorting (by priority, due date, name)
  tags: feature, frontend

- [x] v0.9.0 — Table view mode
  tags: feature, frontend
  Alternate view rendering boards as sortable tables instead of kanban columns.

- [x] v0.8.5 — Git auto-commit on every mutation
  tags: feature, backend

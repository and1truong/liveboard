---
version: 23
name: Sprint 14
description: Two-week sprint — ship billing and onboarding
icon: "\U0001F3AF"
tags:
    - sprint
    - engineering
members:
    - Maya
    - htruong
    - Jane Doe
list-collapse:
    - false
    - false
    - false
    - false
settings:
    expand-columns: true
    view-mode: board
---

## Backlog

- [ ] Rate limiting middleware
  tags: backend, api
  priority: medium
  Add token bucket rate limiter to public API endpoints. 100 req/min per API key.

- [ ] Webhook retry with exponential backoff
  tags: backend, webhooks
  assignee: Jane Doe
  priority: low
  due: 2026-03-08
  Currently we fire-and-forget. Need retry queue with 3 attempts.

- [ ] Dark mode toggle in user settings
  tags: frontend, ui
  priority: low

- [ ] Add OpenAPI spec generation
  tags: backend, docs
  priority: medium

## Review

- [ ] Add CSRF token to all mutation endpoints
  tags: backend, security
  assignee: Maya
  priority: high
  PR #247 — needs security review before merge

- [ ] Refactor card parser to handle nested markdown
  tags: backend, parser
  assignee: Hieu
  priority: medium
  PR #243 — passes all tests, needs code review

## In Progress

- [ ] Stripe checkout session integration
  tags: backend, billing
  assignee: Hieu
  priority: critical
  due: 2026-03-24
  Wire up checkout sessions for pro plan. Use price_1ABC from Stripe dashboard. Need to handle webhooks for payment success/failure.

- [ ] Onboarding wizard — step 1: workspace setup
  tags: frontend, onboarding
  assignee: Maya
  priority: high
  due: 2026-03-25
  Three-step wizard. **First** step collects workspace name and invites. Use the stepper component from the design system.

- [ ] Fix N+1 query in board listing endpoint
  tags: backend, performance
  assignee: Hieu
  priority: high
  `GET /api/boards` fires a query per board for member count. Batch it.

## Done

- [x] Set up CI pipeline with GitHub Actions
  tags: devops

- [x] Implement JWT refresh token rotation
  tags: backend, auth

- [x] Add drag-and-drop card reordering
  tags: frontend, ux

- [x] Write integration tests for board CRUD
  tags: backend, testing

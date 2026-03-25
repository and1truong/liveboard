---
name: Sample Project
description: A realistic project board to show LiveBoard in action
icon: "\U0001F504"
tags:
    - workflows
tag-colors:
    api: '#4080c4'
    auth: '#e05252'
    database: '#607080'
    frontend: '#8060c4'
    testing: '#a07040'
    deploy: '#4caf76'
list-collapse:
    - false
    - false
    - false
    - false
settings:
    view-mode: board
    card-display-mode: trim
    expand-columns: false
---

## Backlog

- [ ] Add password reset flow #auth
  tags: frontend, api
  priority: medium
  Need email template, reset token generation, and a new /reset-password page. Check how the signup flow handles token expiry — reuse that pattern.

- [ ] Migrate user table to support OAuth providers #auth
  tags: database
  priority: high
  Add provider and provider_id columns. Keep backward compat with email/password users. Migration script in /db/migrations.

- [ ] Write API docs for v2 endpoints
  tags: api
  priority: low
  assignee: sam
  OpenAPI spec for /api/v2/*. Use the existing v1 spec as a starting point.

## In Progress

- [ ] Build notification preferences page #frontend
  tags: frontend
  assignee: alice
  priority: high
  due: 2026-03-28
  Figma mockup approved. Three sections: email, push, in-app. Each has per-event toggles. Use the existing settings page layout.

- [ ] Set up staging environment #deploy
  tags: deploy
  assignee: bob
  priority: critical
  due: 2026-03-26
  Docker Compose for staging. Needs: app server, Postgres, Redis, Nginx. Mirror prod config but with debug logging enabled.

## Review

- [ ] Rate limiting middleware #api
  tags: api, auth
  assignee: alice
  priority: high
  PR #47 — 100 req/min per IP for public endpoints, 1000 for authenticated. Using token bucket. Needs load test before merge.

- [x] Fix duplicate webhook deliveries #api
  tags: api
  assignee: bob
  priority: medium
  PR #45 merged. Added idempotency key check. Webhook table now has a unique constraint on (event_id, endpoint_url).

## Done

- [x] Database connection pooling #database
  tags: database
  assignee: bob
  priority: high
  Switched from single connection to pgxpool. Max 25 connections, min 5. Reduced p99 latency from 240ms to 85ms.

- [x] CI pipeline with GitHub Actions #deploy
  tags: deploy, testing
  assignee: alice
  priority: medium
  Lint → test → build → deploy-preview. Runs on push to main and PRs. Takes ~3 min. Added Slack notification on failure.

- [x] User avatar upload #frontend
  tags: frontend, api
  assignee: sam
  priority: medium
  S3 presigned URL flow. Client uploads directly to S3, then sends the key to the API. Max 2MB, jpg/png only. Crops to 256x256.

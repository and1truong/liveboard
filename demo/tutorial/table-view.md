---
version: 1
name: Table View
description: See your cards as a sortable table
icon: "\U0001F4CB"
tags:
    - views
tag-colors:
    bug: '#e05252'
    docs: '#a07040'
    feature: '#4caf76'
    infra: '#607080'
    ux: '#45aab5'
settings:
    view-mode: table
    card-display-mode: full
---

## To Do

- [ ] Write API documentation for v2 endpoints
  tags: docs, feature
  assignee: alice
  priority: high
  due: 2026-04-01
  The new endpoints need full OpenAPI specs and example requests before the public launch.

- [ ] Fix pagination bug on mobile
  tags: bug, ux
  priority: critical
  due: 2026-03-27
  assignee: bob
  Cards beyond page 2 fail to load on iOS Safari. Likely a fetch offset issue.

- [ ] Add dark mode toggle to settings page
  tags: feature, ux
  priority: medium
  due: 2026-04-10
  assignee: carol
  Users have been requesting this since launch. Needs to respect system preference by default.

- [ ] Migrate CI to GitHub Actions
  tags: infra
  priority: low
  due: 2026-04-15
  assignee: dave
  Current Jenkins setup is flaky. GHA would simplify the pipeline and reduce maintenance.

## In Progress

- [ ] Redesign the onboarding flow
  tags: ux, feature
  assignee: carol
  priority: high
  due: 2026-03-30
  New users drop off at step 3. Simplify to two steps with inline previews.

- [ ] Set up error tracking with Sentry
  tags: infra
  assignee: dave
  priority: medium
  due: 2026-03-28
  Need stack traces and breadcrumbs for production issues. Free tier covers our volume.

## Done

- [x] Launch landing page
  tags: feature, ux
  assignee: alice
  priority: high
  Went live last Thursday. Conversion rate looks solid at 4.2%.

- [x] Configure rate limiting on public API
  tags: infra
  assignee: bob
  priority: critical
  Set to 100 req/min per key. Burst allowance of 20.

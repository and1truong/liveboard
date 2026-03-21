---
name: Docs Sprint
description: API documentation overhaul for a client's developer portal
icon: "\U0001F4DA"
tags:
    - docs
    - technical-writing
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: board
---

## Outline

- [ ] Authentication and API keys guide
  tags: auth, guide
  priority: high
  Cover: API key generation, OAuth2 flow, token refresh, rate limits. Include code samples in Python, Node, and cURL.

- [ ] Webhooks integration guide
  tags: webhooks, guide
  priority: medium
  Event types, payload schemas, retry behavior, signature verification. Add troubleshooting section.

- [ ] Error codes reference
  tags: reference
  priority: medium
  Full table of error codes with descriptions, causes, and suggested fixes. Currently scattered across Notion pages.

## Drafting

- [ ] Getting started tutorial — "Your first API call in 5 minutes"
  tags: tutorial
  assignee: Jules
  priority: critical
  due: 2026-03-26
  Step-by-step from signup to first successful request. Include runnable code snippets. Test in a fresh environment.

- [ ] REST API reference — Payments endpoints
  tags: reference, api
  assignee: Priya
  priority: high
  due: 2026-03-28
  12 endpoints. Each needs: description, parameters, request/response examples, error codes. Generate from OpenAPI spec where possible.

## Review

- [ ] REST API reference — Users endpoints
  tags: reference, api
  priority: medium
  Draft complete. Need technical review from client's backend team. 8 endpoints documented.

- [ ] Migration guide: v2 to v3
  tags: guide, migration
  priority: high
  due: 2026-03-24
  Breaking changes list, before/after code samples, deprecation timeline. Client wants this before v3 GA.

## Published

- [x] Developer portal landing page copy
  tags: content

- [x] SDK installation guides (Python, Node, Go)
  tags: sdk, guide

- [x] Changelog page template and first 3 entries
  tags: changelog

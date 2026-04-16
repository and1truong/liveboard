---
version: 16
name: Product Launch
description: Shipping v1 of FormBot — AI form builder SaaS
icon: "\U0001F680"
tags:
    - product
    - launch
tag-colors:
    auth: '#e05252'
    backend: '#607080'
    billing: '#4caf76'
    content: '#c060a0'
    devops: '#d4722c'
    docs: '#a07040'
    email: '#c060a0'
    infra: '#607080'
    marketing: '#c060a0'
    seo: '#c060a0'
    ui: '#45aab5'
    ux: '#45aab5'
list-collapse:
    - false
    - false
    - false
    - true
settings:
    show-checkbox: true
    expand-columns: true
    card-display-mode: normal
---

## Backlog

- [ ] Add SEO meta tags to all pages
  tags: marketing, seo
  priority: low

- [ ] Customer onboarding flow
  tags: ux
  priority: medium

- [ ] Beta invite email sequence
  tags: marketing, email
  priority: medium

- [ ] Write changelog for v1.0
  tags: docs

## This Week

- [ ] Landing page with waitlist form
  tags: marketing
  priority: high

- [ ] Set up Plausible analytics
  tags: infra

- [ ] Write launch blog post
  tags: marketing, content
  priority: medium

## In Progress

- [ ] Stripe billing integration
  tags: billing
  priority: critical

- [ ] Submit to Product Hunt
  tags: marketing
  priority: high
  due: 2026-04-01

## Shipped

- [x] Auth with magic links
  tags: auth

- [x] Database schema and migrations
  tags: backend

- [x] Deploy to Fly.io
  tags: infra, devops

- [x] Responsive dashboard layout
  tags: ui

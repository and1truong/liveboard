---
name: On-Call Week
description: Tracking incidents, follow-ups, and handoff for on-call rotation
icon: "\U0001F514"
tags:
    - oncall
    - ops
list-collapse:
    - true
    - false
    - false
    - false
---

## Active Incidents

- [ ] Elevated 5xx on /api/checkout since deploy v2.14.3
  tags: incident, api
  assignee: John Doe
  priority: high
  Rolled back to v2.14.2. Root cause TBD — suspect connection pool exhaustion.

- [ ] Redis cluster memory pressure on prod-cache-03
  tags: incident, redis
  assignee: John Doe
  priority: critical
  due: 2026-03-20
  OOM kills started at 03:12 UTC. Scaled to 8GB, monitoring.

## Investigating

- [ ] Flaky health checks on k8s node pool us-east1-b
  tags: infra, kubernetes
  priority: medium
  Node cordoned. 3 pods rescheduled. Checking kubelet logs.

- [ ] Grafana alert: p99 latency > 800ms on search-service
  tags: performance, search
  priority: medium

- [ ] Sentry spike: null pointer in payment-service v3.8.1
  tags: bug, payments
  priority: low
  Only hitting one merchant. Non-blocking, queued for next sprint.

## Follow-Ups

- [ ] Write postmortem for Monday DB failover
  tags: postmortem
  priority: high
  due: 2026-03-21

- [ ] Update runbook for Redis OOM procedure
  tags: runbook, redis
  priority: medium

- [ ] Ping platform team RE: node pool auto-repair config
  tags: infra, kubernetes

- [ ] File ticket to increase checkout connection pool size
  tags: api, config
  priority: medium

## Handed Off

- [x] CDN cache invalidation stuck for /static/js
  tags: cdn, resolved
  Purged manually. Root cause: origin header mismatch after Cloudflare migration.

- [x] PagerDuty escalation policy not routing to secondary
  tags: pagerduty, resolved
  Fixed rotation schedule — was pointing at old team slug.

- [x] Cert expiry warning on internal-api.corp.net
  tags: tls, resolved
  Renewed via cert-manager. Added 30-day alert to monitoring.

---
name: Postmortems
description: Incident reviews, action items, and follow-through tracking
icon: "\U0001F50D"
tags:
    - postmortem
    - incidents
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: board
---

## Draft

- [ ] INC-4462: Payment webhook outage — 30min P1
  tags: payments, p1
  assignee: Ravi Patel
  priority: critical
  due: 2026-03-22
  Expired TLS cert on Stripe webhook endpoint. 30min total outage, 12min detection time. Impact: ~$45k in delayed transactions, 312 affected orders. Need to document: timeline, root cause, detection gap, remediation.

- [ ] INC-4471: Cart service OOM — recurring P1
  tags: cart, p1
  priority: high
  due: 2026-03-25
  Third OOM in 2 weeks. Goroutine leak in websocket handler. Each incident causes 5-8min of degraded checkout. Need to document: pattern analysis, why we didn't catch it earlier, memory profiling gaps.

## Action Items

- [ ] Add cert-manager auto-renewal for all webhook endpoints
  tags: tls, automation
  priority: critical
  due: 2026-03-24
  From INC-4462. Currently 4 webhook endpoints use manually managed certs. Migrate all to cert-manager with 30-day renewal window.

- [ ] Implement goroutine leak detection in CI
  tags: testing, ci
  priority: high
  due: 2026-03-28
  From INC-4471. Add goleak to integration test suite. Fail build if goroutine count grows across test runs.

- [ ] Reduce MTTD for payment path failures to < 2min
  tags: monitoring, detection
  assignee: Lisa Park
  priority: high
  due: 2026-04-01
  From INC-4462. Current MTTD was 12min. Add Datadog synthetic check for Stripe webhook. Alert on 2 consecutive failures.

- [ ] Add memory usage alerting per pod (not just node)
  tags: monitoring, kubernetes
  priority: medium
  From INC-4471. Current alerts are node-level only. Add per-pod memory usage alert at 80% of limit.

- [ ] Document Stripe webhook failover procedure
  tags: runbook, payments
  priority: medium
  From INC-4462. If primary webhook is down, how to activate backup endpoint and replay missed events.

## Reviewed

- [ ] INC-4389: Search index corruption after ES upgrade
  tags: search, p2
  priority: low
  Postmortem published. 2/3 action items complete. Remaining: add pre-upgrade index validation script.

- [ ] INC-4352: DNS propagation delay after zone migration
  tags: dns, p2
  priority: low
  All action items complete. Monitoring TTL compliance dashboard added. Closing next review cycle.

## Closed

- [x] INC-4301: Redis sentinel split-brain
  tags: redis, p1
  All 4 action items completed. Sentinel topology hardened. Runbook updated. No recurrence in 6 weeks.

- [x] INC-4287: CI/CD pipeline deploying to wrong environment
  tags: cicd, p2
  Root cause: environment variable collision in GitHub Actions matrix. Fixed with explicit env pinning. Guardrail added.

- [x] INC-4256: Cascading failure from health check thundering herd
  tags: kubernetes, p1
  Staggered health check intervals. Added jitter to readiness probes. Load test confirmed fix.

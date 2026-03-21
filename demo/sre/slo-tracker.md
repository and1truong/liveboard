---
name: SLO Tracker
description: Service level objectives — Q1 2026 burn rate and action items
icon: "\U0001F4CA"
tags:
    - slo
    - reliability
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: table
---

## Burning Error Budget

- [ ] checkout-service availability: 99.82% (target 99.95%)
  tags: checkout, availability
  priority: critical
  due: 2026-03-25
  Burned 3.6x monthly error budget in first 2 weeks. Root cause: Monday payment outage (INC-4462) + cart OOM incidents. Need hotfix and capacity review.

- [ ] search-service p99 latency: 620ms (target 400ms)
  tags: search, latency
  priority: high
  due: 2026-03-28
  Degraded after ES cluster rebalance. Index shard count needs optimization. Also seeing GC pressure on data nodes.

- [ ] order-processor end-to-end latency: 4.2s (target 2s)
  tags: orders, latency
  priority: high
  Kafka consumer lag is the bottleneck. Batch processing config needs tuning after payload format change.

## Needs Attention

- [ ] auth-service error rate: 0.08% (target 0.1%)
  tags: auth, errors
  priority: medium
  Close to budget. Mostly 429s from aggressive rate limiting on mobile clients. Consider per-device rate limit instead of per-IP.

- [ ] image-cdn cache hit ratio: 89% (target 95%)
  tags: cdn, performance
  priority: medium
  Dropped after origin URL scheme change. Cache keys include origin path — need purge and re-warm.

- [ ] notification-service delivery rate: 97.1% (target 99%)
  tags: notifications, delivery
  priority: medium
  FCM push failures on Android. Token refresh not propagating from mobile app v3.1. Client fix is in next release.

## Healthy

- [ ] api-gateway availability: 99.99% (target 99.95%)
  tags: gateway, availability
  priority: low
  Well within budget. No action needed.

- [ ] user-service p50 latency: 12ms (target 50ms)
  tags: users, latency
  priority: low
  Solid. Connection pooling tuning from last sprint paid off.

- [ ] payment-service success rate: 99.94% (target 99.9%)
  tags: payments, success
  priority: low
  Recovered after INC-4462 fix. Stripe failover to secondary PSP working as designed.

## Closed Actions

- [x] Fix api-gateway retry storm during upstream outages
  tags: gateway, reliability
  Added circuit breaker with 50% threshold, 30s half-open window. Error amplification eliminated.

- [x] Reduce search-service cold start time from 45s to 8s
  tags: search, performance
  Pre-warmed ES connection pool and loaded ML model at init. Autoscaler can now respond to traffic spikes.

- [x] Add error budget alerting to Slack #sre-alerts
  tags: monitoring, slo
  Datadog SLO monitors now page at 50% budget consumed, alert Slack at 30%.

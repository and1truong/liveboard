---
name: "On-Call: Mar 17–23"
description: Week 12 rotation — primary oncall handoff tracker
icon: "\U0001F4DF"
tags:
    - oncall
    - week-12
list-collapse:
    - true
    - false
    - false
    - false
settings:
    expand-columns: false
    view-mode: board
---

## Active Incidents

- [ ] INC-4471: Cart service OOMKilled in us-east-1
  tags: incident, p1
  assignee: Ravi Patel
  priority: critical
  due: 2026-03-21
  Pod restarts every 8min. Heap dump shows unbounded goroutine leak in websocket handler. Temporary mitigation: bumped memory limit to 4Gi. Need hotfix in cart-service v2.31.

- [ ] INC-4473: Kafka consumer lag > 500k on order-events topic
  tags: incident, kafka, p2
  assignee: Ravi Patel
  priority: high
  Consumer group `order-processor` stuck. Rebalancing loop. Suspect partition reassignment after broker-3 restart at 02:00 UTC.

## Investigating

- [ ] Grafana alert: disk usage > 85% on postgres-primary-01
  tags: database, capacity
  priority: high
  WAL segments not being cleaned up. `pg_archiver` last ran 6 hours ago. Check archive_command and WAL archive destination.

- [ ] Intermittent 503s on /api/search — 0.3% error rate
  tags: api, search
  priority: medium
  Started after search-service v4.2.0 deploy. No rollback yet — error rate is low. Correlates with Elasticsearch GC pauses.

- [ ] Sentry: deadline exceeded in inventory-service gRPC calls
  tags: grpc, timeout
  priority: medium
  p99 latency spiked from 120ms to 900ms. Only affects `GetStock` RPC. Suspect slow query on inventory DB.

## Follow-Ups

- [ ] Write postmortem for INC-4462 (Monday payment outage)
  tags: postmortem, p1
  priority: high
  due: 2026-03-22
  30-min outage. Root cause: Stripe webhook endpoint certificate expired. Detection time was 12min — need to improve.

- [ ] Update PagerDuty escalation policy for platform-infra
  tags: pagerduty, process
  priority: medium
  Secondary rotation still points to Alex who moved to ML team. Update to new team roster.

- [ ] Add synthetic monitoring for checkout critical path
  tags: monitoring, synthetics
  priority: medium
  We detected the Monday outage from customer complaints, not monitoring. Add Datadog Synthetic for: login → add to cart → checkout → confirmation.

- [ ] Tune Kafka consumer batch size for order-processor
  tags: kafka, tuning
  priority: low
  Current batch size 500 is too aggressive for the new payload format. Try 200 and measure throughput.

## Handed Off

- [x] INC-4462: Payment webhook endpoint returning 502
  tags: incident, p1, resolved
  Cert renewed. Added cert-manager annotation for auto-renewal. PagerDuty timeline documented.

- [x] INC-4468: Redis sentinel failover flapping
  tags: incident, redis, resolved
  Sentinel quorum was 2/3 but one sentinel was on a failing node. Moved to dedicated sentinel pods.

- [x] Noisy PagerDuty alerts from staging environment
  tags: pagerduty, noise, resolved
  Staging was routing to prod PagerDuty service. Fixed routing key in Terraform.

---
name: Runbook Backlog
description: Runbook gaps, updates, and documentation debt
icon: "\U0001F4D6"
tags:
    - runbooks
    - docs
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: board
---

## Missing Runbooks

- [ ] Kafka cluster recovery — full broker failure
  tags: kafka, critical-gap
  priority: critical
  due: 2026-03-26
  No documented procedure for broker loss + partition reassignment. Last incident we winged it. Need: detection, isolation, partition rebalance, consumer group recovery.

- [ ] Elasticsearch cluster red status
  tags: elasticsearch, critical-gap
  priority: high
  Current runbook only covers yellow. Red status (unassigned primary shards) needs: shard allocation, forced reroute, data recovery from snapshot.

- [ ] Database failover — Postgres primary to replica promotion
  tags: database, critical-gap
  priority: high
  due: 2026-03-28
  We have automated failover via Patroni but no runbook for manual intervention when automation fails. Include: fencing, timeline check, pgBouncer reconnect.

## Needs Update

- [ ] Redis sentinel failover runbook
  tags: redis, update
  assignee: Ravi Patel
  priority: high
  due: 2026-03-24
  After INC-4468 we learned the sentinel topology section is wrong. Update pod locations, quorum config, and add Kubernetes-specific steps.

- [ ] SSL/TLS certificate renewal runbook
  tags: tls, update
  priority: medium
  Doesn't mention cert-manager. Still references manual certbot steps. Add: cert-manager troubleshooting, manual override, and verification commands.

- [ ] Kubernetes node drain procedure
  tags: kubernetes, update
  priority: medium
  Missing PDB considerations. Add: check PodDisruptionBudgets, verify replica count, handle stateful workloads.

- [ ] Datadog agent troubleshooting
  tags: monitoring, update
  priority: low
  Agent v7 migration changed config paths. Update flare collection and log locations.

## In Review

- [ ] Incident commander checklist — v2
  tags: process, incident
  assignee: Lisa Park
  priority: high
  due: 2026-03-23
  Revised after Q1 retro. Added: stakeholder comms template, severity classification matrix, handoff checklist.

- [ ] Rollback procedure for Argo CD deployments
  tags: argocd, gitops
  priority: medium
  Covers: app sync revert, image tag rollback, manual helm override. Needs SRE team review.

## Published

- [x] PagerDuty triage flowchart
  tags: pagerduty, process
  Decision tree for severity classification. Linked from PagerDuty webhook.

- [x] Kubernetes pod crash loop debugging
  tags: kubernetes, debugging
  Step-by-step: check events, describe pod, logs (previous), resource limits, liveness probes.

- [x] AWS EKS node group scaling runbook
  tags: aws, kubernetes
  Manual and automated scaling. Includes spot instance interruption handling.

- [x] Grafana dashboard provisioning guide
  tags: monitoring, grafana
  JSON model management, folder structure, alerting rules.

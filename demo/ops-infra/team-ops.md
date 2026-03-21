---
name: Team Ops
description: Platform team operations — hiring, OKRs, vendor eval
icon: "\U0001F465"
tags:
    - team
    - ops
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: table
---

## Active

- [ ] Hire senior SRE — final round interviews
  tags: hiring
  assignee: Lisa Park
  priority: high
  due: 2026-03-28
  3 candidates in final round. Schedule panel interviews this week. Comp band approved by HR.

- [ ] Evaluate Teleport vs Tailscale for zero-trust access
  tags: vendor, security
  assignee: Mike Torres
  priority: medium
  due: 2026-04-05
  POC both in staging. Key criteria: SSO integration, audit logging, session recording.

- [ ] Q1 OKR scoring — platform reliability
  tags: okrs
  priority: high
  due: 2026-03-31
  KR1: 99.95% uptime (actual: 99.92%). KR2: MTTR < 30min (actual: 22min). KR3: Zero P0 incidents (actual: 1).

- [ ] Draft Q2 OKR proposals
  tags: okrs
  priority: medium
  due: 2026-04-07
  Focus areas: observability improvements, cost optimization, developer experience.

## Backlog

- [ ] Run team retrospective for Q1
  tags: process
  priority: medium
  Book 90-min slot. Prepare Miro board with timeline of major incidents and wins.

- [ ] Consolidate monitoring vendors — Datadog vs Grafana Cloud
  tags: vendor, cost
  priority: low
  Current Datadog bill is $8k/mo. Grafana Cloud estimate is $3.5k. Need feature parity analysis.

- [ ] Update on-call compensation policy
  tags: hr, oncall
  priority: medium
  Current policy is outdated. Research industry benchmarks for on-call pay.

## Waiting On

- [ ] Budget approval for additional staging cluster
  tags: infra, budget
  priority: medium
  Submitted to finance 2026-03-15. Expected approval by end of month.

- [ ] Legal review of Teleport enterprise license
  tags: vendor, legal
  priority: low
  Sent to legal 2026-03-18. Standard SaaS terms but need data residency clause.

## Done

- [x] Onboard new junior SRE — Alex Kim
  tags: hiring, onboarding
  Completed 2-week ramp. Assigned to observability squad.

- [x] Migrate CI from Jenkins to GitHub Actions
  tags: ci, devops
  All 23 pipelines migrated. Jenkins decommissioned.

- [x] Negotiate Datadog contract renewal
  tags: vendor, cost
  Locked in 15% discount for annual commitment.

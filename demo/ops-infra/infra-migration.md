---
name: K8s Migration
description: Migrating payment-service from ECS to Kubernetes
icon: "\u2699\uFE0F"
tags:
    - infra
    - kubernetes
    - migration
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: board
---

## Planning

- [ ] Define resource requests and limits for all containers
  tags: capacity, k8s
  priority: high
  Pull last 30 days of ECS metrics from CloudWatch. Map to k8s resource specs. Don't forget sidecar containers.

- [ ] Write Helm chart for payment-service
  tags: helm, k8s
  priority: high
  Base off the api-gateway chart. Need: deployment, service, HPA, PDB, configmap, secrets.

- [ ] Plan database connection string migration
  tags: database, secrets
  priority: medium
  Currently in ECS task definition env vars. Move to ExternalSecrets with AWS Secrets Manager backend.

## Executing

- [ ] Deploy to staging k8s cluster
  tags: k8s, staging
  assignee: Mike Torres
  priority: critical
  due: 2026-03-26
  Staging cluster is ready. Need to deploy, run smoke tests, validate metrics in Grafana.

- [ ] Set up Istio service mesh for payment-service
  tags: networking, istio
  assignee: Sarah Chen
  priority: high
  mTLS between payment-service and downstream services. Configure retry policy and circuit breaker.

- [ ] Migrate Datadog dashboards from ECS to k8s metrics
  tags: monitoring, datadog
  priority: medium
  ECS metrics (CPU/mem utilization, task count) need to be replaced with k8s equivalents (pod CPU/mem, replica count, HPA status).

## Validating

- [ ] Load test payment-service on k8s at 2x peak traffic
  tags: performance, testing
  priority: high
  Use k6 scripts from perf-tests repo. Target: p99 < 200ms at 500 RPS. Compare with ECS baseline.

- [ ] Verify PCI DSS network controls in k8s
  tags: security, pci
  priority: critical
  Network policies must isolate payment namespace. Auditor needs evidence of segmentation.

## Complete

- [x] Provision EKS cluster with Terraform
  tags: terraform, aws
  3 node groups: system, app, spot. Cluster autoscaler configured.

- [x] Set up Argo CD for GitOps deployments
  tags: gitops, argocd
  App-of-apps pattern. Syncs from infra-k8s repo main branch.

- [x] Configure Prometheus + Grafana monitoring stack
  tags: monitoring, prometheus
  kube-prometheus-stack deployed. Custom dashboards for SLOs.

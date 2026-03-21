---
name: Security Audit
description: SOC2 Type II audit findings and remediation tracking
icon: "\U0001F512"
tags:
    - security
    - compliance
list-collapse:
    - false
    - false
    - false
    - true
settings:
    expand-columns: false
    view-mode: board
---

## Open Findings

- [ ] SA-027: Service accounts with overly broad IAM roles
  tags: iam, gcp
  priority: critical
  due: 2026-03-28
  12 service accounts have Editor role. Auditor flagged as excessive. Need to scope down to least-privilege custom roles.

- [ ] SA-031: No encryption at rest for staging database
  tags: encryption, database
  priority: high
  due: 2026-04-01
  Prod uses CMEK, but staging RDS instance has default encryption disabled.

- [ ] SA-034: Missing MFA enforcement for admin console
  tags: auth, admin
  priority: high
  3 admin accounts without MFA. Need org-wide policy enforcement.

- [ ] SA-038: Audit logs not shipped to immutable storage
  tags: logging, compliance
  priority: medium
  CloudTrail logs go to S3 but bucket has no object lock. Auditor wants WORM.

## Remediation

- [ ] SA-022: Rotate all API keys older than 90 days
  tags: secrets, rotation
  assignee: Sarah Chen
  priority: high
  due: 2026-03-25
  14 keys identified. Script drafted in infra-tools repo. Need to coordinate with app teams for zero-downtime rotation.

- [ ] SA-029: Enable VPC flow logs in production
  tags: networking, logging
  assignee: Mike Torres
  priority: medium
  Terraform PR up. Estimated $200/mo additional cost. Approved by infra lead.

## Verification

- [ ] SA-019: Verify SSH key rotation completed across fleet
  tags: ssh, verification
  assignee: Sarah Chen
  priority: medium
  Keys rotated last week. Need to confirm no old keys remain on any host.

- [ ] SA-021: Confirm container image scanning in CI
  tags: containers, ci
  priority: low
  Trivy added to pipeline. Verify it blocks deploys on critical CVEs.

## Closed

- [x] SA-015: Enable database audit logging
  tags: database, logging
  pgAudit enabled on all Postgres instances. Logs ship to CloudWatch.

- [x] SA-018: Implement network segmentation for PCI scope
  tags: networking, pci
  VPC peering removed. Transit gateway with security groups in place.

- [x] SA-020: Add SIEM integration for auth events
  tags: auth, monitoring
  Auth0 logs streaming to Datadog. Alert rules configured for brute force.

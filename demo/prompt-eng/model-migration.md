---
name: Model Migration
description: Tracking prompt updates for Claude 4.5 to 4.6 migration
icon: "\U0001F504"
tags:
    - migration
    - claude-4.6
members:
    - alex
    - dana
    - marco
    - priya
list-collapse:
    - false
    - false
    - false
    - true
settings:
    view-mode: board
---

## Not Started

- [ ] Migrate: customer churn prediction
  tags: analytics
  priority: medium
  assignee: dana
  Baseline eval on 4.5 before switching. Watch for changes in risk_score calibration.

- [ ] Migrate: SQL query generator
  tags: code-gen
  priority: medium
  assignee: marco
  Re-run adversarial suite on 4.6. New model may handle complex joins better but verify mutation refusal still holds.

- [ ] Migrate: competitor analysis (10-K)
  tags: research
  priority: low
  assignee: priya
  Low risk — output is advisory. Quick smoke test should suffice.

## In Progress

- [ ] Migrate: support ticket classifier
  tags: classification
  priority: high
  assignee: dana
  due: 2026-03-26
  4.6 shows +3% accuracy on category classification in early tests. Urgency scoring regressed slightly — investigating.

- [ ] Migrate: product description writer
  tags: copywriting
  priority: high
  assignee: alex
  due: 2026-03-25
  Tone control is tighter on 4.6. Luxury tone improved significantly. Casual tone needs prompt tweak — comes across too stiff.

- [ ] Migrate: code review assistant
  tags: dev-tools
  priority: medium
  assignee: marco
  due: 2026-03-28
  4.6 reduced false positives on style by 40%. Security detection unchanged. Running full eval suite now.

## Validated

- [ ] Migrate: email draft generator
  tags: support
  priority: high
  assignee: priya
  due: 2026-03-23
  4.6 scores higher on empathy (4.8 vs 4.6). No regressions. Ready to swap in production.

- [ ] Migrate: legal clause summarizer
  tags: legal
  priority: medium
  assignee: dana
  4.6 output is more concise. Legal team re-approved. Disclaimer injection still works correctly.

## Done

- [x] Migrate: blog post generator v3
  tags: copywriting
  Swapped to 4.6 on 2026-03-15. SEO metrics stable. Readability improved slightly.

- [x] Migrate: knowledge base Q&A
  tags: rag
  Swapped to 4.6 on 2026-03-12. Retrieval unchanged (RAG layer). Generation quality up — fewer hedging phrases.

- [x] Migrate: invoice extractor
  tags: extraction
  Swapped to 4.6 on 2026-03-10. All 12 formats still pass. Extraction speed improved.

- [x] Migrate: sentiment analysis
  tags: analytics
  Swapped to 4.6 on 2026-03-08. Sentiment polarity more nuanced. Dashboard updated.

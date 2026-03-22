---
name: Eval Pipeline
description: Evaluation runs, regression tracking, and quality gates
icon: "\U0001F9EA"
tags:
    - evals
    - quality
list-collapse:
    - false
    - false
    - false
    - true
settings:
    view-mode: board
---

## Queued

- [ ] Eval: churn prediction prompt on Q1 customer data
  tags: analytics, eval
  priority: high
  assignee: dana
  due: 2026-03-26
  500 labeled examples from CS team. Split 80/20 train/test. Measure precision, recall, and calibration of risk_score.

- [ ] Eval: SQL generator — adversarial inputs
  tags: code-gen, security, eval
  priority: high
  assignee: marco
  Test SQL injection attempts, ambiguous table names, joins on 5+ tables. Must refuse all mutation queries.

- [ ] Eval: meeting summarizer v2 vs v1 A/B
  tags: summarization, eval
  priority: medium
  assignee: alex
  Side-by-side comparison on 30 real meeting transcripts. Human raters score completeness, conciseness, action-item capture.

## Running

- [ ] Eval: product description writer — brand consistency
  tags: copywriting, eval
  priority: high
  assignee: alex
  due: 2026-03-24
  200 products across 5 brands. Measuring tone match, factual accuracy, and keyword inclusion. 60% complete — ETA tomorrow.

- [ ] Eval: support ticket classifier — edge cases
  tags: classification, eval
  priority: medium
  assignee: dana
  Testing multi-label tickets, non-English, extremely short messages. Currently at 150/300 cases.

## Review Results

- [ ] Results: email draft generator — empathy + accuracy
  tags: support, eval
  priority: high
  assignee: priya
  due: 2026-03-23
  Empathy: 4.6/5, Accuracy: 4.8/5, Hallucination rate: 0.4%. Green across all gates. Recommending promotion to production.

- [ ] Results: code review assistant — false positive rate
  tags: dev-tools, eval
  priority: medium
  assignee: marco
  FP rate on style issues: 34% (too high). Security and perf flags are solid at 6% FP. Need to recalibrate severity thresholds before re-eval.

## Completed

- [x] Eval: blog generator v3 — SEO and readability
  tags: copywriting, eval
  Flesch-Kincaid: 58 avg (target 55-65). Keyword density: 1.8% (target 1-2%). Passed all gates.

- [x] Eval: knowledge base Q&A — retrieval accuracy
  tags: rag, eval
  Golden set: 93% accuracy. Latency acceptable. Failure mode: questions spanning multiple docs.

- [x] Eval: invoice extractor — format coverage
  tags: extraction, eval
  12/12 formats passing. Added 3 new edge-case formats from APAC vendors — all green.

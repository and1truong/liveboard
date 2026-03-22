---
name: Prompt Library
description: Core prompt catalog — drafting, testing, and shipping to production
icon: "\U0001F4DA"
tags:
    - prompts
    - library
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

- [ ] Customer churn prediction — analyst prompt #classification #analytics
  tags: analytics, ml
  priority: high
  assignee: dana
  Prompt that takes customer usage data and predicts churn risk with reasoning. Needs to output structured JSON with risk_score, factors, and recommended_actions.

- [ ] Meeting notes summarizer v2 #summarization
  tags: productivity, summarization
  priority: medium
  assignee: alex
  Rewrite from scratch. v1 loses action items when meetings are longer than 45 min. Try chain-of-thought extraction before final summary.

- [ ] SQL query generator from natural language #code-gen
  tags: code-gen, sql
  priority: medium
  Schema-aware prompt that generates safe SELECT queries. Must refuse UPDATE/DELETE. Include dialect param for postgres vs mysql.

- [ ] Competitor analysis from 10-K filings #research
  tags: research, finance
  priority: low
  assignee: priya
  Extract key metrics, risks, and strategic moves. Output as markdown table comparing up to 4 companies.

## Testing

- [ ] Product description writer — e-commerce #copywriting
  tags: copywriting, ecommerce
  priority: high
  assignee: alex
  due: 2026-03-24
  Tone knob works (casual/professional/luxury). Testing edge cases: very short input, non-English product names, products with no category. Eval score: 4.1/5 on brand consistency.

- [ ] Support ticket classifier + router #classification
  tags: support, classification
  priority: high
  assignee: dana
  due: 2026-03-25
  Classifies into billing/technical/feature-request/other and assigns urgency. Running against 500-ticket eval set. Accuracy so far: 91% on category, 84% on urgency.

- [ ] Code review assistant #code-review
  tags: dev-tools, code-review
  priority: medium
  assignee: marco
  Focuses on security, performance, and readability. Testing across Python, TypeScript, and Go. Tends to over-flag trivial style issues — needs calibration.

## Reviewed

- [ ] Customer email draft generator #copywriting
  tags: support, copywriting
  priority: high
  assignee: priya
  due: 2026-03-23
  Passed eval: empathy score 4.6/5, accuracy 4.8/5. Handles refunds, shipping delays, product defects. Ready for A/B test in production.

- [ ] Legal clause summarizer #summarization
  tags: legal, summarization
  priority: medium
  assignee: dana
  Reviewed by legal team. Approved with caveat: must include disclaimer that output is not legal advice. Add disclaimer as system-level postfix.

## Production

- [x] Blog post generator v3 #copywriting
  tags: copywriting, content
  Deployed 2026-03-10. Handles SEO keywords, tone control, and target word count. Monitoring via weekly eval sample.

- [x] Internal knowledge base Q&A #rag
  tags: rag, internal
  Deployed 2026-03-05. RAG pipeline with reranking. Latency p95: 2.1s. Accuracy on golden set: 93%.

- [x] Invoice data extractor #extraction
  tags: extraction, finance
  Deployed 2026-02-20. Extracts vendor, line items, totals, due date from PDF invoices. Handles 12 invoice formats.

- [x] Sentiment analysis — social mentions #classification
  tags: analytics, classification
  Deployed 2026-02-15. Tracks brand sentiment across Twitter, Reddit, HN. Feeds dashboard via webhook.

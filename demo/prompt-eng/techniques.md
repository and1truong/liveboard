---
version: 3
name: Techniques & Patterns
description: Reusable prompt engineering patterns and research notes
icon: "\U0001F9E0"
tags:
    - techniques
    - research
list-collapse:
    - false
    - false
    - false
settings:
    view-mode: board
    card-display-mode: full
---

## To Investigate

- [ ] Constitutional AI self-critique chains
  tags: alignment, alignment, safety
  assignee: alex
  priority: high
  Can we use critique-revision loops to reduce hallucination in our RAG prompts? Paper suggests 2-3 rounds optimal.

- [ ] Prompt caching strategies for long system prompts
  tags: optimization, optimization, cost
  assignee: marco
  priority: high
  Our system prompts average 2k tokens. Caching could cut costs 30-40% on high-volume endpoints.

- [ ] Multi-turn vs single-turn for complex extraction
  tags: extraction, extraction, architecture
  assignee: dana
  priority: medium
  Invoice extractor uses single-turn. Hypothesis: multi-turn with verification step catches more edge cases but doubles latency.

- [ ] Structured output with tool_use vs JSON mode
  tags: architecture, architecture, reliability
  assignee: priya
  priority: medium
  Compare reliability of tool_use-based structured output vs asking for JSON in prompt. Track parse failure rates.

## Active Research

- [ ] Few-shot example selection — dynamic vs static
  tags: few-shot, few-shot, retrieval
  assignee: alex
  priority: high
  due: 2026-03-28
  Building retrieval-based example selector. Embed user query, find 3 nearest examples from bank. Early results: +8% on classification tasks vs static few-shot.

- [ ] Chain-of-thought prompting for multi-step reasoning
  tags: reasoning, reasoning, cot
  assignee: marco
  priority: medium
  Testing explicit CoT vs letting model reason implicitly. CoT adds ~200 tokens but improves accuracy 12% on complex SQL generation.

- [ ] Prompt compression — removing filler without quality loss
  tags: optimization, optimization, cost
  assignee: dana
  priority: medium
  Systematically removing polite filler, redundant instructions. Goal: 30% token reduction with <1% quality drop. At 22% reduction so far with no measurable quality change.

## Documented

- [x] Persona-based system prompts — best practices
  tags: personas, system-prompt
  Documented pattern: role + constraints + examples + output format. Shared in Notion wiki.

- [x] Temperature tuning guide per task type
  tags: optimization, parameters
  Classification: 0.0-0.1. Creative writing: 0.7-0.9. Extraction: 0.0. Summarization: 0.3-0.5.

- [x] Eval framework — golden set methodology
  tags: evals, methodology
  50+ labeled examples minimum. Stratified by difficulty. Human agreement baseline required before automation.

- [x] Prompt versioning — naming conventions
  tags: process, versioning
  Format: {task}-v{major}.{minor}. Major = behavioral change. Minor = wording tweak. All versions in git.

---
version: 4
name: Feature Showcase
description: Everything a card can do
icon: ✨
tags:
    - features
tag-colors:
    backend: '#607080'
    bug: '#e05252'
    design: '#8060c4'
    docs: '#a07040'
    feature: '#4caf76'
    frontend: '#4080c4'
    urgent: '#e05252'
    ux: '#45aab5'
settings:
    expand-columns: false
    view-mode: board
    card-display-mode: full
    week-start: sunday
---

## Card Features

- [ ] Tags help you categorize
  tags: frontend, design, ux
  priority: medium
  Tags can be added inline with #hashtags in the title, or as metadata below. Both get merged together. You can assign colors to tags in the board settings.

- [ ] Priorities set urgency levels
  tags: feature
  priority: critical
  due: 2026-03-26
  Four levels: **critical**, **high**, **medium**, **low**. This card is critical priority with a due date of tomorrow. Priorities show as colored indicators on the card.

- [ ] Assign cards to team members
  tags: feature
  assignee: alice
  priority: high
  The assignee field shows who's responsible. Great for team collaboration. Assignees appear as avatars on the card.

- [ ] Due dates keep you on track
  tags: feature
  priority: medium
  due: 2026-04-01
  Set a due date and LiveBoard shows how much time remains. Overdue cards get highlighted so nothing slips through.

- [ ] Card bodies support Markdown
  tags: docs
  priority: low
  You can write **bold**, *italic*, `inline code`, and more in card bodies. The body is everything below the metadata lines.

## Organization

- [ ] Custom metadata on any card
  tags: feature
  priority: medium
  estimate: 3 hours
  custom-field: any value you want
  Cards support arbitrary key-value metadata. Add any field you need — it'll be preserved in the markdown file.

- [ ] Inline tags are extracted automatically
  tags: backend, docs
  priority: low
  Hashtags in the card title get extracted and merged with the tags field. Write `#bug` in a title and it becomes a filterable tag.

- [x] Completed cards keep their data
  tags: feature
  assignee: bob
  priority: low
  Even when checked off, all metadata is preserved. Nothing is lost.

## Views

- [ ] Board view — what you see now
  tags: ux
  priority: medium
  The default Kanban layout. Columns side by side, cards stacked vertically. Drag and drop to reorder.

- [ ] Calendar view for date-based work
  tags: feature
  priority: medium
  Switch to calendar view to see cards plotted by their due dates. Great for deadline-driven workflows. Toggle it from the view menu.

- [ ] Compact mode for dense boards
  tags: ux
  priority: low
  Too many cards? Switch to compact or trim display mode in board settings to show less detail per card.

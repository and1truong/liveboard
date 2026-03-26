---
name: Tips & Tricks
description: Power user features and keyboard shortcuts
icon: "\U0001F4A1"
tags:
    - tips
list-collapse:
    - false
    - false
    - false
settings:
    view-mode: board
    card-display-mode: full
---

## Keyboard Shortcuts

- [ ] Command Palette — Cmd+K / Ctrl+K
  priority: high
  The fastest way to navigate. Opens a search bar to jump between boards, create new boards, or run actions. Start typing and select.

- [ ] Quick card creation — just start typing
  priority: medium
  Focus a column and press Enter to create a new card inline. Hit Escape to cancel.

- [ ] Navigate with keyboard
  priority: medium
  Arrow keys to move between cards. Enter to open a card. Escape to close. Tab to move between columns.

- [ ] Toggle theme — keyboard shortcut
  priority: low
  Switch between light and dark mode instantly. Your preference is saved and persists across sessions.

## Power Features

- [ ] Drag and drop everything
  priority: high
  Cards between columns, cards within a column to reorder, even columns themselves. Grab and move.

- [ ] Collapse columns to save space
  priority: medium
  Click the column header chevron to collapse it. Collapsed state is saved per-board. Great for hiding "Done" columns.

- [ ] Real-time collaboration via SSE
  priority: medium
  Open the same board in two browser tabs. Edit in one — watch the other update instantly. No refresh needed. Works across devices on the same network.

- [ ] Your data is just Markdown
  priority: high
  Every board is a `.md` file. Edit it in VS Code, commit to Git, grep it, back it up — it's your data. No database, no lock-in.

## Customization

- [ ] Change the site name
  priority: low
  Edit `settings.json` in your workspace root. Set `site_name` to whatever you want — it shows in the header and browser tab.

- [ ] Color your tags
  priority: medium
  Add `tag-colors` in a board's frontmatter to assign hex colors to tags. Colors appear as badges on cards.

- [ ] Adjust column width
  priority: low
  Set `column_width` in settings.json (default 512px). Wider for detailed cards, narrower for compact overview.

- [ ] Choose your card display mode
  priority: low
  Three modes: **full** (show everything), **trim** (truncate long bodies), **compact** (title + tags only). Set globally or per-board.

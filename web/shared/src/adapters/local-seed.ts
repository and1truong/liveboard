import type { Board } from '../types.js'

export const WELCOME_BOARD: Board = {
  version: 1,
  name: 'Welcome',
  description: 'This is your demo board. Data stays in this browser.',
  icon: '👋',
  tags: ['demo'],
  columns: [
    {
      name: 'Todo',
      cards: [
        { title: 'Try dragging this card to Done' },
        { title: 'Double-click the board title to rename it' },
      ],
    },
    { name: 'Doing', cards: [{ title: 'Build something awesome' }] },
    { name: 'Done', cards: [{ title: 'Read the intro' }] },
  ],
}

export const WORKSPACE_NAME = 'Demo'

import { describe, expect, it, mock } from 'bun:test'
import { scheduleDelete } from './undoable.js'

describe('scheduleDelete', () => {
  it('fires after timeout when not cancelled', async () => {
    const fire = mock(() => {})
    scheduleDelete(fire, 30)
    await new Promise((r) => setTimeout(r, 60))
    expect(fire).toHaveBeenCalledTimes(1)
  })

  it('does not fire when cancelled before timeout', async () => {
    const fire = mock(() => {})
    const h = scheduleDelete(fire, 50)
    h.cancel()
    await new Promise((r) => setTimeout(r, 80))
    expect(fire).not.toHaveBeenCalled()
  })
})

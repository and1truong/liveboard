import { toast } from '../toast.js'

const UNDO_MS = 5000

export function scheduleDelete(
  fire: () => void,
  ms: number = UNDO_MS,
): { cancel: () => void } {
  let done = false
  const timer = setTimeout(() => {
    if (!done) fire()
  }, ms)
  return {
    cancel: () => {
      done = true
      clearTimeout(timer)
    },
  }
}

export function stageDelete(fire: () => void, label: string): void {
  const handle = scheduleDelete(fire)
  toast(`Deleted ${label}`, {
    duration: UNDO_MS,
    action: { label: 'Undo', onClick: handle.cancel },
  })
}

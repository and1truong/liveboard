import type { MutationOp } from '@shared/types.js'
import type { UseMutationResult } from '@tanstack/react-query'
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

export function stageDelete(
  mutation: UseMutationResult<unknown, unknown, MutationOp, unknown>,
  op: MutationOp,
  label: string,
): void {
  const handle = scheduleDelete(() => mutation.mutate(op))
  toast(`Deleted ${label}`, {
    duration: UNDO_MS,
    action: { label: 'Undo', onClick: handle.cancel },
  })
}

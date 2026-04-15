import { toast, Toaster } from 'sonner'

export { toast, Toaster }

export function errorToast(code: string): void {
  const copy: Record<string, string> = {
    VERSION_CONFLICT: 'Board changed elsewhere — refreshed',
    NOT_FOUND: 'Card or column not found',
    OUT_OF_RANGE: 'Index out of range',
    INVALID: 'Invalid input',
    INTERNAL: 'Server error — try again',
    ALREADY_EXISTS: 'A board with that name already exists',
  }
  toast.error(copy[code] ?? code)
}

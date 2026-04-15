import { Suspense, lazy, useEffect, useRef, useState } from 'react'

const CommandPalette = lazy(() =>
  import('./CommandPalette.js').then((m) => ({ default: m.CommandPalette })),
)

export function CommandPaletteHost(): JSX.Element | null {
  const [open, setOpen] = useState(false)
  const hasBeenOpen = useRef(false)
  if (open) hasBeenOpen.current = true

  useEffect(() => {
    const handler = (e: KeyboardEvent): void => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  if (!hasBeenOpen.current) return null

  return (
    <Suspense fallback={null}>
      <CommandPalette open={open} onOpenChange={setOpen} />
    </Suspense>
  )
}

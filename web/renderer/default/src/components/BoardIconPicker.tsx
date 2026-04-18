import { useEffect, useLayoutEffect, useRef, useState, type JSX } from 'react'
import { createPortal } from 'react-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { BoardIcon } from './BoardIcon.js'
import { BOARD_ICON_SLUGS } from '../icons/boardIcons.js'
import { ICON_COLORS, resolveIconColor } from '../icons/iconColors.js'

export function BoardIconPicker({
  boardId,
  icon,
  iconColor,
}: {
  boardId: string
  icon?: string
  iconColor?: string
}): JSX.Element {
  const [open, setOpen] = useState(false)
  // Track color locally so swatch clicks retint the grid preview immediately.
  const [currentColor, setCurrentColor] = useState<string>(resolveIconColor(iconColor))
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const popoverRef = useRef<HTMLDivElement>(null)
  const mutation = useBoardMutation(boardId)
  const qc = useQueryClient()

  useEffect(() => {
    setCurrentColor(resolveIconColor(iconColor))
  }, [iconColor])

  useLayoutEffect(() => {
    if (!open || !buttonRef.current) return
    const updatePos = (): void => {
      const r = buttonRef.current?.getBoundingClientRect()
      if (r) setPos({ top: r.top, left: r.right + 8 })
    }
    updatePos()
    window.addEventListener('resize', updatePos)
    window.addEventListener('scroll', updatePos, true)
    return () => {
      window.removeEventListener('resize', updatePos)
      window.removeEventListener('scroll', updatePos, true)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const onMouseDown = (e: MouseEvent): void => {
      const target = e.target as Node
      if (
        !buttonRef.current?.contains(target) &&
        !popoverRef.current?.contains(target)
      ) {
        setOpen(false)
      }
    }
    const onKey = (e: KeyboardEvent): void => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const invalidate = (): void => {
    void qc.invalidateQueries({ queryKey: ['boards'] })
  }

  const pickColor = (key: string): void => {
    setCurrentColor(key)
    mutation.mutate(
      { type: 'update_board_icon', icon_color: key },
      { onSuccess: invalidate },
    )
  }

  const pickIcon = (slug: string | null): void => {
    setOpen(false)
    // Commit both so the visible tint in the grid is what gets saved, even if
    // the user never clicked a swatch separately.
    mutation.mutate(
      slug === null
        ? { type: 'update_board_icon', icon: null }
        : { type: 'update_board_icon', icon: slug, icon_color: currentColor },
      { onSuccess: invalidate },
    )
  }

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          setOpen((p) => !p)
        }}
        title="Change icon"
        aria-label="Change board icon"
        className={`lb-row__icon${icon ? '' : ' lb-row__icon--placeholder'}`}
      >
        <BoardIcon icon={icon} color={iconColor} size="sm" />
      </button>
      {open && pos && createPortal(
        <div
          ref={popoverRef}
          role="dialog"
          aria-label="Choose board icon"
          onClick={(e) => e.stopPropagation()}
          className="lb-popover lb-popover--icons"
          style={{ position: 'fixed', top: pos.top, left: pos.left }}
        >
          <div className="lb-iconpicker-swatches" role="radiogroup" aria-label="Icon colour">
            {ICON_COLORS.map((c) => (
              <button
                key={c.key}
                type="button"
                aria-label={c.label}
                aria-pressed={currentColor === c.key}
                title={c.label}
                className={`lb-iconpicker-swatch lb-bicon--c-${c.key}`}
                style={{ background: 'var(--lb-icon-bg)' }}
                onClick={() => pickColor(c.key)}
              />
            ))}
          </div>
          <div className="lb-iconpicker-grid">
            <button
              type="button"
              onClick={() => pickIcon(null)}
              title="Remove icon"
              aria-label="Remove icon"
              className="lb-iconpicker-cell lb-iconpicker-cell--clear"
            >
              ✕
            </button>
            {BOARD_ICON_SLUGS.map((slug) => (
              <button
                key={slug}
                type="button"
                onClick={() => pickIcon(slug)}
                title={slug}
                aria-pressed={icon === slug}
                className="lb-iconpicker-cell"
              >
                <BoardIcon icon={slug} color={currentColor} size="sm" />
              </button>
            ))}
          </div>
        </div>,
        document.body,
      )}
    </>
  )
}

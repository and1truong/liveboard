import { useEffect, useLayoutEffect, useRef, useState, type JSX } from 'react'
import { createPortal } from 'react-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { BoardIcon } from './BoardIcon.js'
import { BOARD_EMOJI_ICONS, BOARD_ICON_SLUGS, isEmojiIcon } from '../icons/boardIcons.js'
import { ICON_COLORS, resolveIconColor } from '../icons/iconColors.js'

type Tab = 'symbol' | 'emoji'

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
  const [tab, setTab] = useState<Tab>(() => (icon && isEmojiIcon(icon) ? 'emoji' : 'symbol'))
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

  const pickIcon = (value: string | null): void => {
    setOpen(false)
    // Commit both so the visible tint in the grid is what gets saved, even if
    // the user never clicked a swatch separately.
    mutation.mutate(
      value === null
        ? { type: 'update_board_icon', icon: null }
        : { type: 'update_board_icon', icon: value, icon_color: currentColor },
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
          <section className="lb-iconpicker-section lb-iconpicker-section--colour">
            <div className="lb-iconpicker-section__label">Colour</div>
            <div className="lb-iconpicker-swatches" role="radiogroup" aria-label="Icon colour">
              {ICON_COLORS.map((c) => (
                <button
                  key={c.key}
                  type="button"
                  aria-label={c.label}
                  aria-pressed={currentColor === c.key}
                  title={c.label}
                  className="lb-iconpicker-swatch"
                  style={{ background: c.swatch }}
                  onClick={() => pickColor(c.key)}
                />
              ))}
            </div>
          </section>

          <div className="lb-iconpicker-divider" aria-hidden />

          <section className="lb-iconpicker-section lb-iconpicker-section--icon">
            <div className="lb-iconpicker-section__header">
              <div className="lb-iconpicker-tabs" role="tablist">
                <button
                  type="button"
                  role="tab"
                  aria-selected={tab === 'symbol'}
                  className="lb-iconpicker-tab"
                  onClick={() => setTab('symbol')}
                >
                  Symbol
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={tab === 'emoji'}
                  className="lb-iconpicker-tab"
                  onClick={() => setTab('emoji')}
                >
                  Emoji
                </button>
              </div>
              <button
                type="button"
                onClick={() => pickIcon(null)}
                className="lb-iconpicker-clear"
                title="Remove icon"
              >
                Clear
              </button>
            </div>
            <div className="lb-iconpicker-grid">
              {tab === 'symbol'
                ? BOARD_ICON_SLUGS.map((slug) => (
                    <button
                      key={slug}
                      type="button"
                      onClick={() => pickIcon(slug)}
                      title={slug}
                      aria-pressed={icon === slug}
                      className="lb-iconpicker-cell"
                    >
                      <BoardIcon icon={slug} color={currentColor} size="md" />
                    </button>
                  ))
                : BOARD_EMOJI_ICONS.map((emoji) => (
                    <button
                      key={emoji}
                      type="button"
                      onClick={() => pickIcon(emoji)}
                      title={emoji}
                      aria-pressed={icon === emoji}
                      className="lb-iconpicker-cell"
                    >
                      <BoardIcon icon={emoji} size="md" />
                    </button>
                  ))}
            </div>
          </section>
        </div>,
        document.body,
      )}
    </>
  )
}

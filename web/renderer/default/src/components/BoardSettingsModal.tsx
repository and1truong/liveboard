import { forwardRef, useEffect, useRef, useState, type FormEvent, type ReactNode, type Ref, type SelectHTMLAttributes } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { useBoardSettings, useUpdateSettings } from '../queries/useBoardSettings.js'
import { useAvailableTags } from '../queries/useAvailableTags.js'
import { useDeleteBoard } from '../mutations/useBoardCrud.js'
import { useBoardMutation } from '../mutations/useBoardMutation.js'
import { stageDelete } from '../mutations/undoable.js'
import { useBoard } from '../queries.js'
import { TAG_PALETTE, tagChipStyle } from '../utils/tagColor.js'

type ViewMode = 'board' | 'list' | 'calendar'
const VIEW_MODES: { value: ViewMode; label: string; glyph: string }[] = [
  { value: 'board', label: 'Board', glyph: '▦' },
  { value: 'list', label: 'List', glyph: '☰' },
  { value: 'calendar', label: 'Calendar', glyph: '▤' },
]

function normalizeViewMode(v: string | null | undefined): ViewMode {
  if (v === 'list' || v === 'calendar') return v
  return 'board'
}

type WeekStart = 'sunday' | 'monday'
const WEEK_STARTS: { value: WeekStart; label: string }[] = [
  { value: 'sunday', label: 'Sunday' },
  { value: 'monday', label: 'Monday' },
]

function normalizeWeekStart(v: string | null | undefined): WeekStart {
  return v === 'sunday' ? 'sunday' : 'monday'
}

export function BoardSettingsModal({
  boardId,
  boardName,
  open,
  onOpenChange,
}: {
  boardId: string
  boardName: string
  open: boolean
  onOpenChange: (next: boolean) => void
}): JSX.Element {
  const settings = useBoardSettings(boardId)
  const mutation = useUpdateSettings(boardId)
  const deleteMut = useDeleteBoard()
  const boardQuery = useBoard(boardId)
  const board = boardQuery.data
  const boardMut = useBoardMutation(boardId)
  const checkboxRef = useRef<HTMLInputElement>(null)
  const expandRef = useRef<HTMLInputElement>(null)
  const modeRef = useRef<HTMLSelectElement>(null)
  const [viewMode, setViewMode] = useState<ViewMode>(() => normalizeViewMode(settings.view_mode))
  const [weekStart, setWeekStart] = useState<WeekStart>(() => normalizeWeekStart(settings.week_start))
  const [confirmDelete, setConfirmDelete] = useState(false)
  const savedTagColors = board?.tag_colors ?? {}
  const [tagColors, setTagColors] = useState<Record<string, string>>(savedTagColors)
  // When the modal reopens for a board, reset all drafts from the latest saved state.
  useEffect(() => {
    if (open) {
      setTagColors(board?.tag_colors ?? {})
      setViewMode(normalizeViewMode(settings.view_mode))
      setWeekStart(normalizeWeekStart(settings.week_start))
    }
  }, [open, board?.tag_colors, settings.view_mode, settings.week_start])

  const availableTags = useAvailableTags(board)

  const setTagColor = (tag: string, color: string): void => {
    setTagColors((prev) => {
      const next = { ...prev }
      if (next[tag] === color) delete next[tag]
      else next[tag] = color
      return next
    })
  }
  const clearTagColor = (tag: string): void => {
    setTagColors((prev) => {
      if (!(tag in prev)) return prev
      const next = { ...prev }
      delete next[tag]
      return next
    })
  }
  const tagColorsChanged = (): boolean => {
    const keys = new Set([...Object.keys(savedTagColors), ...Object.keys(tagColors)])
    for (const k of keys) if (savedTagColors[k] !== tagColors[k]) return true
    return false
  }

  const requestDelete = (): void => {
    if (!confirmDelete) {
      setConfirmDelete(true)
      return
    }
    setConfirmDelete(false)
    onOpenChange(false)
    stageDelete(() => deleteMut.mutate(boardId), boardName)
  }

  const submit = (e: FormEvent): void => {
    e.preventDefault()
    if (tagColorsChanged()) {
      boardMut.mutate({ type: 'update_tag_colors', tag_colors: tagColors })
    }
    mutation.mutate(
      {
        show_checkbox: checkboxRef.current?.checked ?? true,
        expand_columns: expandRef.current?.checked ?? false,
        card_display_mode: (modeRef.current?.value as 'normal' | 'compact') ?? 'normal',
        view_mode: viewMode,
        week_start: weekStart,
      },
      {
        onSuccess: () => onOpenChange(false),
      },
    )
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="lb-settings__overlay" />
        <Dialog.Content
          key={String(open)}
          aria-label="Board settings"
          aria-describedby={undefined}
          className="lb-settings"
        >
          <header className="lb-settings__header">
            <div className="lb-settings__heading">
              <span className="lb-settings__icon" aria-hidden="true">
                {board?.icon ?? '⚙'}
              </span>
              <div className="lb-settings__heading-text">
                <Dialog.Title className="lb-settings__title">{boardName}</Dialog.Title>
                <p className="lb-settings__subtitle">Board preferences</p>
              </div>
            </div>
            <Dialog.Close asChild>
              <button type="button" aria-label="Close" className="lb-settings__close">
                ×
              </button>
            </Dialog.Close>
          </header>

          <form onSubmit={submit} className="lb-settings__form">
            <div className="lb-settings__scroll">
              {/* Appearance — view mode + week start */}
              <Section label="Appearance">
                <Row label="View" hint="How this board is laid out.">
                  <SegmentedControl
                    ariaLabel="view mode"
                    value={viewMode}
                    options={VIEW_MODES}
                    onChange={(v) => setViewMode(v as ViewMode)}
                  />
                </Row>
                {viewMode === 'calendar' && (
                  <Row label="Week starts on">
                    <SegmentedControl
                      ariaLabel="week start"
                      value={weekStart}
                      options={WEEK_STARTS}
                      onChange={(v) => setWeekStart(v as WeekStart)}
                    />
                  </Row>
                )}
                <Row label="Card density">
                  <SelectField
                    aria-label="card display mode"
                    ref={modeRef}
                    defaultValue={settings.card_display_mode}
                  >
                    <option value="normal">Comfortable</option>
                    <option value="compact">Compact</option>
                  </SelectField>
                </Row>
              </Section>

              {/* Behavior — toggles */}
              <Section label="Cards & Columns">
                <ToggleRow
                  ref={checkboxRef}
                  ariaLabel="show complete checkbox"
                  defaultChecked={settings.show_checkbox}
                  title="Show complete checkbox"
                  hint="Reveal a checkbox on each card to mark it done."
                />
                <ToggleRow
                  ref={expandRef}
                  ariaLabel="expand columns"
                  defaultChecked={settings.expand_columns}
                  title="Expand columns"
                  hint="Stretch columns to fill the available width."
                />
              </Section>

              {/* Tag colors */}
              <Section
                label="Tag colors"
                aside={availableTags.length > 0 ? `${availableTags.length}` : undefined}
              >
                {availableTags.length === 0 ? (
                  <div className="lb-settings__empty">No tags on this board yet.</div>
                ) : (
                  <ul className="lb-settings__tags">
                    {availableTags.map((tag) => {
                      const current = tagColors[tag]
                      return (
                        <li key={tag} className="lb-settings__tag-row">
                          <span
                            style={tagChipStyle(current)}
                            className={`lb-settings__tag-chip${current ? ' lb-settings__tag-chip--colored' : ''}`}
                          >
                            {tag}
                          </span>
                          <div
                            className="lb-settings__swatches"
                            role="group"
                            aria-label={`color swatches for ${tag}`}
                          >
                            {TAG_PALETTE.map((c) => (
                              <button
                                key={c}
                                type="button"
                                onClick={() => setTagColor(tag, c)}
                                aria-label={`set ${tag} color ${c}`}
                                aria-pressed={current === c}
                                className={`lb-settings__swatch${current === c ? ' lb-settings__swatch--active' : ''}`}
                                style={{ backgroundColor: c }}
                              />
                            ))}
                            <button
                              type="button"
                              onClick={() => clearTagColor(tag)}
                              aria-label={`clear ${tag} color`}
                              disabled={!current}
                              className="lb-settings__swatch-clear"
                            >
                              ×
                            </button>
                          </div>
                        </li>
                      )
                    })}
                  </ul>
                )}
              </Section>

              {/* Danger zone */}
              <Section label="Danger zone" tone="danger">
                {!confirmDelete ? (
                  <div className="lb-settings__danger-row">
                    <div className="lb-settings__danger-text">
                      <p className="lb-settings__row-title">Delete this board</p>
                      <p className="lb-settings__row-hint">
                        Permanently removes the markdown file from your workspace.
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={requestDelete}
                      className="lb-settings__btn lb-settings__btn--danger-ghost"
                    >
                      Delete board…
                    </button>
                  </div>
                ) : (
                  <div className="lb-settings__danger-confirm">
                    <p className="lb-settings__row-title">
                      Delete <strong>{boardName}</strong>?
                    </p>
                    <p className="lb-settings__row-hint">
                      You&rsquo;ll have 5 seconds to undo before the file is removed.
                    </p>
                    <div className="lb-settings__danger-actions">
                      <button
                        type="button"
                        onClick={() => setConfirmDelete(false)}
                        className="lb-settings__btn lb-settings__btn--ghost"
                      >
                        Keep board
                      </button>
                      <button
                        type="button"
                        onClick={requestDelete}
                        className="lb-settings__btn lb-settings__btn--danger"
                      >
                        Confirm delete
                      </button>
                    </div>
                  </div>
                )}
              </Section>
            </div>

            <footer className="lb-settings__footer">
              <button
                type="button"
                onClick={() => onOpenChange(false)}
                className="lb-settings__btn lb-settings__btn--ghost"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={mutation.isPending}
                className="lb-settings__btn lb-settings__btn--primary"
              >
                {mutation.isPending ? 'Saving…' : 'Save'}
              </button>
            </footer>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

/* ---------- Internal building blocks ---------- */

function Section({
  label,
  aside,
  tone,
  children,
}: {
  label: string
  aside?: string
  tone?: 'danger'
  children: ReactNode
}): JSX.Element {
  return (
    <section className={`lb-settings__section${tone === 'danger' ? ' lb-settings__section--danger' : ''}`}>
      <div className="lb-settings__section-head">
        <span className="lb-settings__section-label">{label}</span>
        {aside !== undefined && <span className="lb-settings__section-aside">{aside}</span>}
      </div>
      <div className="lb-settings__group">{children}</div>
    </section>
  )
}

function Row({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: ReactNode
}): JSX.Element {
  return (
    <div className="lb-settings__row">
      <div className="lb-settings__row-text">
        <span className="lb-settings__row-title">{label}</span>
        {hint && <span className="lb-settings__row-hint">{hint}</span>}
      </div>
      <div className="lb-settings__row-control">{children}</div>
    </div>
  )
}

function SegmentedControl<T extends string>({
  ariaLabel,
  value,
  options,
  onChange,
}: {
  ariaLabel: string
  value: T
  options: { value: T; label: string; glyph?: string }[]
  onChange: (next: T) => void
}): JSX.Element {
  return (
    <div role="radiogroup" aria-label={ariaLabel} className="lb-segmented">
      {options.map((m) => {
        const active = value === m.value
        return (
          <button
            key={m.value}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={`${ariaLabel} ${m.value}`}
            onClick={() => onChange(m.value)}
            className={`lb-segmented__btn${active ? ' lb-segmented__btn--active' : ''}`}
          >
            {m.glyph && <span className="lb-segmented__glyph" aria-hidden="true">{m.glyph}</span>}
            <span>{m.label}</span>
          </button>
        )
      })}
    </div>
  )
}

const SelectField = forwardRef(function SelectField(
  { children, ...rest }: SelectHTMLAttributes<HTMLSelectElement>,
  ref: Ref<HTMLSelectElement>,
): JSX.Element {
  return (
    <span className="lb-select">
      <select ref={ref} className="lb-select__el" {...rest}>
        {children}
      </select>
      <span className="lb-select__chevron" aria-hidden="true">
        ⌄
      </span>
    </span>
  )
})

const ToggleRow = forwardRef(function ToggleRow(
  {
    ariaLabel,
    defaultChecked,
    title,
    hint,
  }: { ariaLabel: string; defaultChecked: boolean; title: string; hint?: string },
  ref: Ref<HTMLInputElement>,
): JSX.Element {
  return (
    <label className="lb-settings__row lb-settings__row--toggle">
      <div className="lb-settings__row-text">
        <span className="lb-settings__row-title">{title}</span>
        {hint && <span className="lb-settings__row-hint">{hint}</span>}
      </div>
      <span className="lb-toggle">
        <input
          ref={ref}
          aria-label={ariaLabel}
          type="checkbox"
          role="switch"
          defaultChecked={defaultChecked}
          className="lb-toggle__input"
        />
        <span className="lb-toggle__track" aria-hidden="true">
          <span className="lb-toggle__thumb" />
        </span>
      </span>
    </label>
  )
})

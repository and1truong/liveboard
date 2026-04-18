import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Command } from 'cmdk'
import { Plus, Pencil, Settings, LayoutGrid, List, CalendarDays, Trash2 } from 'lucide-react'
import { useBoardList } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useOptionalBoardFocus } from '../contexts/BoardFocusContext.js'
import { useSearch } from '../queries/useSearch.js'
import { sanitize } from './markdownPreview.js'
import {
  useCreateBoard,
  useRenameBoard,
  useDeleteBoard,
} from '../mutations/useBoardCrud.js'
import { stageDelete } from '../mutations/undoable.js'
import { useBoardSettings, useUpdateSettings } from '../queries/useBoardSettings.js'
import { useBoardSettingsContext } from '../contexts/BoardSettingsContext.js'
import { useGlobalSettingsContext } from '../contexts/GlobalSettingsContext.js'
import { BoardIcon } from './BoardIcon.js'

const VIEW_MODES: { value: 'board' | 'list' | 'calendar'; label: string }[] = [
  { value: 'board', label: 'Board' },
  { value: 'list', label: 'List' },
  { value: 'calendar', label: 'Calendar' },
]

const VIEW_ICONS: Record<string, ReactNode> = {
  board: <LayoutGrid size={15} />,
  list: <List size={15} />,
  calendar: <CalendarDays size={15} />,
}

const ICON_CLS = 'shrink-0 mr-2.5 text-[color:var(--color-text-muted)]'

type Page = 'list' | 'create' | 'rename'

interface CommandPaletteProps {
  open: boolean
  onOpenChange: (next: boolean) => void
}

const ITEM_BASE =
  'group cursor-pointer border-l-2 border-transparent rounded-r px-3 py-2.5 text-sm ' +
  'text-[color:var(--color-text-primary)] flex items-center ' +
  'aria-selected:bg-[color:var(--color-column-bg)] aria-selected:border-[var(--accent-500)]'

const ARROW = (
  <span
    aria-hidden
    className="ml-auto pl-3 opacity-0 group-aria-selected:opacity-100 text-[color:var(--color-text-muted)]"
  >
    ›
  </span>
)

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps): JSX.Element {
  const [page, setPage] = useState<Page>('list')
  const [cmdValue, setCmdValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const cmdListRef = useRef<HTMLDivElement>(null)
  const committedRef = useRef(false)

  const [query, setQuery] = useState('')
  const hits = useSearch(query)
  const focusCtx = useOptionalBoardFocus()

  const boards = useBoardList()
  const { active, setActive } = useActiveBoard()
  const createMut = useCreateBoard()
  const renameMut = useRenameBoard()
  const deleteMut = useDeleteBoard()
  const settings = useBoardSettings(active)
  const updateSettingsMut = useUpdateSettings(active ?? '')
  const { openSettings } = useBoardSettingsContext()
  const { openSettings: openGlobalSettings } = useGlobalSettingsContext()

  const activeBoard = boards.data?.find((b) => b.id === active) ?? null
  const activeName = activeBoard?.name ?? ''

  // Reset to list page on every (re)open.
  useEffect(() => {
    if (open) {
      setPage('list')
      setQuery('')
      setCmdValue('')
      committedRef.current = false
    }
  }, [open])

  // Focus input on page change to create/rename.
  useEffect(() => {
    if (open && (page === 'create' || page === 'rename')) {
      committedRef.current = false
      // Defer to next tick so the input has mounted.
      setTimeout(() => {
        inputRef.current?.focus()
        inputRef.current?.select()
      }, 0)
    }
  }, [open, page])

  const close = (): void => {
    onOpenChange(false)
  }

  const submitCreate = (e: FormEvent): void => {
    e.preventDefault()
    if (committedRef.current) return
    committedRef.current = true
    const name = (inputRef.current?.value ?? '').trim()
    if (name) createMut.mutate(name)
    close()
  }

  const submitRename = (e: FormEvent): void => {
    e.preventDefault()
    if (committedRef.current) return
    committedRef.current = true
    const next = (inputRef.current?.value ?? '').trim()
    if (active && next && next !== activeName) {
      renameMut.mutate({ boardId: active, newName: next })
    }
    close()
  }

  const handleCmdKeyDown = (e: React.KeyboardEvent): void => {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
    const list = cmdListRef.current
    if (!list) return
    const items = Array.from(list.querySelectorAll<HTMLElement>('[cmdk-item]:not([hidden])'))
    if (items.length === 0) return
    const selectedIdx = items.findIndex((el) => el.getAttribute('aria-selected') === 'true')
    if (e.key === 'ArrowDown' && selectedIdx === items.length - 1) {
      e.preventDefault()
      e.stopPropagation()
      const val = items[0].getAttribute('data-value')
      if (val) { setCmdValue(val); list.scrollTop = 0 }
    } else if (e.key === 'ArrowUp' && selectedIdx === 0) {
      e.preventDefault()
      e.stopPropagation()
      const val = items[items.length - 1].getAttribute('data-value')
      if (val) { setCmdValue(val); list.scrollTop = list.scrollHeight }
    }
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm" />
        <Dialog.Content
          aria-label="Command palette"
          aria-describedby={undefined}
          className="fixed left-1/2 top-[20%] z-50 w-full max-w-lg -translate-x-1/2 rounded-lg border border-[color:var(--color-border)] bg-[color:var(--color-surface)] p-3 shadow-[var(--shadow-raised)]"
        >
          <Dialog.Title className="sr-only">Command palette</Dialog.Title>
          {page === 'list' && (
            <Command label="Command palette" className="flex flex-col" value={cmdValue} onValueChange={setCmdValue}>
              <div className="border-b border-[color:var(--color-border)]">
                <Command.Input
                  value={query}
                  onValueChange={setQuery}
                  onKeyDown={handleCmdKeyDown}
                  placeholder="Type a command, board, or card…"
                  className="w-full rounded px-3 py-2.5 text-base text-[color:var(--color-text-primary)] outline-none placeholder:text-[color:var(--color-text-muted)] bg-transparent"
                />
              </div>
              <Command.List ref={cmdListRef} className="max-h-80 overflow-y-auto py-2">
                <Command.Empty className="px-3 py-2 text-sm text-[color:var(--color-text-muted)]">
                  No matches.
                </Command.Empty>
                {boards.data && boards.data.length > 0 && (
                  <Command.Group heading="Boards" className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:font-bold [&_[cmdk-group-heading]]:text-[color:var(--color-text-muted)]">
                    {boards.data.map((b) => (
                      <Command.Item
                        key={b.id}
                        value={`board ${b.name}`}
                        onSelect={() => {
                          setActive(b.id)
                          close()
                        }}
                        className={ITEM_BASE}
                      >
                        <span className="flex-1 min-w-0 flex items-center gap-2">
                          {b.icon && <BoardIcon icon={b.icon} color={b.icon_color} size="sm" />}
                          {b.name}
                        </span>
                        {ARROW}
                      </Command.Item>
                    ))}
                  </Command.Group>
                )}
                <Command.Group heading="Actions" className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:font-bold [&_[cmdk-group-heading]]:text-[color:var(--color-text-muted)]">
                  <Command.Item
                    value="action create board"
                    onSelect={() => setPage('create')}
                    className={ITEM_BASE}
                  >
                    <span aria-hidden className={ICON_CLS}><Plus size={15} /></span>
                    <span className="flex-1 min-w-0">Create board</span>
                    {ARROW}
                  </Command.Item>
                  {active !== null && (
                    <>
                      <Command.Item
                        value="action rename current board"
                        onSelect={() => setPage('rename')}
                        className={ITEM_BASE}
                      >
                        <span aria-hidden className={ICON_CLS}><Pencil size={15} /></span>
                        <span className="flex-1 min-w-0">Rename current board</span>
                        {ARROW}
                      </Command.Item>
                      <Command.Item
                        value="action board settings"
                        onSelect={() => { openSettings(); close() }}
                        className={ITEM_BASE}
                      >
                        <span aria-hidden className={ICON_CLS}><Settings size={15} /></span>
                        <span className="flex-1 min-w-0">Board settings</span>
                        {ARROW}
                      </Command.Item>
                      {VIEW_MODES.filter((m) => m.value !== settings.view_mode).map((m) => (
                        <Command.Item
                          key={`view-${m.value}`}
                          value={`action switch view ${m.label}`}
                          onSelect={() => {
                            updateSettingsMut.mutate({ view_mode: m.value })
                            close()
                          }}
                          className={ITEM_BASE}
                        >
                          <span aria-hidden className={ICON_CLS}>{VIEW_ICONS[m.value]}</span>
                          <span className="flex-1 min-w-0">Switch view: {m.label}</span>
                          {ARROW}
                        </Command.Item>
                      ))}
                      <Command.Item
                        value="action delete current board"
                        onSelect={() => {
                          stageDelete(() => deleteMut.mutate(active), activeName)
                          close()
                        }}
                        className={
                          'group cursor-pointer border-l-2 border-transparent rounded-r px-3 py-2.5 text-sm ' +
                          'flex items-center text-red-600 ' +
                          'aria-selected:bg-red-50 aria-selected:border-red-500 ' +
                          'dark:text-red-400 dark:aria-selected:bg-red-900/30 dark:aria-selected:border-red-500'
                        }
                      >
                        <span aria-hidden className="shrink-0 mr-2.5"><Trash2 size={15} /></span>
                        <span className="flex-1 min-w-0">Delete current board</span>
                        <span
                          aria-hidden
                          className="ml-auto pl-3 opacity-0 group-aria-selected:opacity-100 text-red-400"
                        >
                          ›
                        </span>
                      </Command.Item>
                    </>
                  )}
                  <Command.Item
                    value="action app settings workspace preferences"
                    onSelect={() => { openGlobalSettings(); close() }}
                    className={ITEM_BASE}
                  >
                    <span aria-hidden className={ICON_CLS}><Settings size={15} /></span>
                    <span className="flex-1 min-w-0">App settings</span>
                    {ARROW}
                  </Command.Item>
                </Command.Group>
                {hits.length > 0 && (
                  <Command.Group heading="Cards" className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:font-bold [&_[cmdk-group-heading]]:text-[color:var(--color-text-muted)]">
                    {hits.map((h) => (
                      <Command.Item
                        key={`${h.boardId}:${h.colIdx}:${h.cardIdx}`}
                        value={`card ${h.cardTitle} ${query}`}
                        onSelect={() => {
                          setActive(h.boardId)
                          Promise.resolve().then(() => focusCtx?.setFocused({ colIdx: h.colIdx, cardIdx: h.cardIdx }))
                          close()
                        }}
                        className={ITEM_BASE}
                      >
                        <span className="flex-1 min-w-0">
                          <span className="font-semibold">{h.cardTitle}</span>
                          <span className="ml-2 text-xs text-[color:var(--color-text-muted)]">in {h.boardName}</span>
                          {h.snippet && (
                            <span
                              className="block text-xs text-[color:var(--color-text-muted)]"
                              dangerouslySetInnerHTML={{ __html: sanitize(h.snippet) }}
                            />
                          )}
                        </span>
                        {ARROW}
                      </Command.Item>
                    ))}
                  </Command.Group>
                )}
              </Command.List>
            </Command>
          )}

          {page === 'create' && (
            <form onSubmit={submitCreate} className="flex flex-col gap-1">
              <div className="px-3 pt-1 text-xs uppercase text-[color:var(--color-text-muted)]">New board</div>
              <input
                ref={inputRef}
                aria-label="new board name"
                defaultValue=""
                placeholder="Board name…"
                onKeyDown={(e) => {
                  if (e.key === 'Escape') { e.preventDefault(); close() }
                }}
                className="w-full rounded px-3 py-2 text-base text-[color:var(--color-text-primary)] outline-none placeholder:text-[color:var(--color-text-muted)] bg-transparent"
              />
            </form>
          )}

          {page === 'rename' && (
            <form onSubmit={submitRename} className="flex flex-col gap-1">
              <div className="px-3 pt-1 text-xs uppercase text-[color:var(--color-text-muted)]">Rename board</div>
              <input
                ref={inputRef}
                aria-label="rename current board"
                defaultValue={activeName}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') { e.preventDefault(); close() }
                }}
                className="w-full rounded px-3 py-2 text-base outline-none"
              />
            </form>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

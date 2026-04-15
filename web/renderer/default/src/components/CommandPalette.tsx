import { useEffect, useRef, useState, type FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Command } from 'cmdk'
import { useBoardList } from '../queries.js'
import { useActiveBoard } from '../contexts/ActiveBoardContext.js'
import {
  useCreateBoard,
  useRenameBoard,
  useDeleteBoard,
} from '../mutations/useBoardCrud.js'
import { stageDelete } from '../mutations/undoable.js'

type Page = 'list' | 'create' | 'rename'

export function CommandPalette(): JSX.Element {
  const [open, setOpen] = useState(false)
  const [page, setPage] = useState<Page>('list')
  const inputRef = useRef<HTMLInputElement>(null)
  const committedRef = useRef(false)

  const boards = useBoardList()
  const { active, setActive } = useActiveBoard()
  const createMut = useCreateBoard()
  const renameMut = useRenameBoard()
  const deleteMut = useDeleteBoard()

  const activeBoard = boards.data?.find((b) => b.id === active) ?? null
  const activeName = activeBoard?.name ?? ''

  // Global Cmd+K / Ctrl+K toggle.
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

  // Reset to list page on every (re)open.
  useEffect(() => {
    if (open) {
      setPage('list')
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
    setOpen(false)
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

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          aria-label="Command palette"
          className="fixed left-1/2 top-[20%] z-50 w-full max-w-lg -translate-x-1/2 rounded-lg bg-white p-2 shadow-xl dark:bg-slate-800"
        >
          <Dialog.Title className="sr-only">Command palette</Dialog.Title>
          {page === 'list' && (
            <Command label="Command palette" className="flex flex-col gap-1">
              <Command.Input
                placeholder="Type a command or board name…"
                className="w-full rounded px-3 py-2 text-base outline-none placeholder:text-slate-400"
              />
              <Command.List className="max-h-80 overflow-y-auto">
                <Command.Empty className="px-3 py-2 text-sm text-slate-400">
                  No matches.
                </Command.Empty>
                {boards.data && boards.data.length > 0 && (
                  <Command.Group heading="Boards" className="text-xs uppercase text-slate-400 [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1">
                    {boards.data.map((b) => (
                      <Command.Item
                        key={b.id}
                        value={`board ${b.name}`}
                        onSelect={() => {
                          setActive(b.id)
                          close()
                        }}
                        className="cursor-pointer rounded px-3 py-1.5 text-sm text-slate-800 aria-selected:bg-slate-100"
                      >
                        {b.icon && <span aria-hidden className="mr-2">{b.icon}</span>}
                        {b.name}
                      </Command.Item>
                    ))}
                  </Command.Group>
                )}
                <Command.Group heading="Actions" className="text-xs uppercase text-slate-400 [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1">
                  <Command.Item
                    value="action create board"
                    onSelect={() => setPage('create')}
                    className="cursor-pointer rounded px-3 py-1.5 text-sm text-slate-800 aria-selected:bg-slate-100"
                  >
                    Create board
                  </Command.Item>
                  {active !== null && (
                    <>
                      <Command.Item
                        value="action rename current board"
                        onSelect={() => setPage('rename')}
                        className="cursor-pointer rounded px-3 py-1.5 text-sm text-slate-800 aria-selected:bg-slate-100"
                      >
                        Rename current board
                      </Command.Item>
                      <Command.Item
                        value="action delete current board"
                        onSelect={() => {
                          stageDelete(() => deleteMut.mutate(active), activeName)
                          close()
                        }}
                        className="cursor-pointer rounded px-3 py-1.5 text-sm text-red-600 aria-selected:bg-red-50"
                      >
                        Delete current board
                      </Command.Item>
                    </>
                  )}
                </Command.Group>
              </Command.List>
            </Command>
          )}

          {page === 'create' && (
            <form onSubmit={submitCreate} className="flex flex-col gap-1">
              <div className="px-3 pt-1 text-xs uppercase text-slate-400">New board</div>
              <input
                ref={inputRef}
                aria-label="new board name"
                defaultValue=""
                placeholder="Board name…"
                onKeyDown={(e) => {
                  if (e.key === 'Escape') { e.preventDefault(); close() }
                }}
                className="w-full rounded px-3 py-2 text-base outline-none placeholder:text-slate-400"
              />
            </form>
          )}

          {page === 'rename' && (
            <form onSubmit={submitRename} className="flex flex-col gap-1">
              <div className="px-3 pt-1 text-xs uppercase text-slate-400">Rename board</div>
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

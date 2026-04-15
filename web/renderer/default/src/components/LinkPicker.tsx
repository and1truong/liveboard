import { useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Command } from 'cmdk'
import { useSearch } from '../queries/useSearch.js'

export function LinkPicker({
  open,
  onOpenChange,
  onPick,
  excludeBoardId,
  excludeCardId,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  onPick: (target: string) => void
  excludeBoardId?: string
  excludeCardId?: string
}): JSX.Element {
  const [query, setQuery] = useState('')
  const hits = useSearch(query)
  const filtered = hits.filter(
    (h) => !(h.boardId === excludeBoardId && h.cardId === excludeCardId),
  )

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          aria-label="Link picker"
          aria-describedby={undefined}
          className="fixed left-1/2 top-1/4 z-50 w-full max-w-md -translate-x-1/2 rounded-lg bg-white dark:bg-slate-800 p-2 shadow-xl"
        >
          <Dialog.Title className="sr-only">Link a card</Dialog.Title>
          <Command label="Pick a card to link" shouldFilter={false}>
            <Command.Input
              value={query}
              onValueChange={setQuery}
              autoFocus
              placeholder="Search a card to link…"
              className="w-full rounded px-3 py-2 text-base outline-none placeholder:text-slate-400 dark:bg-slate-800 dark:text-slate-100"
            />
            <Command.List className="max-h-64 overflow-y-auto">
              {query.length === 0 && (
                <Command.Empty className="px-3 py-2 text-sm text-slate-400">
                  Type to search for a card.
                </Command.Empty>
              )}
              {filtered.map((h) => (
                <Command.Item
                  key={`${h.boardId}:${h.cardId}`}
                  value={`${h.cardTitle} ${h.boardName}`}
                  onSelect={() => {
                    onPick(`${h.boardId}:${h.cardId}`)
                    setQuery('')
                  }}
                  className="cursor-pointer rounded px-3 py-1.5 text-sm aria-selected:bg-slate-100 dark:aria-selected:bg-slate-700"
                >
                  <span className="font-semibold">{h.cardTitle}</span>
                  <span className="ml-2 text-xs text-slate-400">in {h.boardName}</span>
                </Command.Item>
              ))}
            </Command.List>
          </Command>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

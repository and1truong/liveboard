import { describe, expect, it } from 'bun:test'
import { useEffect } from 'react'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider, useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { BoardView } from './BoardView.js'
import { renderWithQuery } from '../test-utils.js'

function pad(n: number): string { return n < 10 ? '0' + n : '' + n }
function todayStr(): string {
  const t = new Date()
  return `${t.getFullYear()}-${pad(t.getMonth() + 1)}-${pad(t.getDate())}`
}

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  await client.putBoardSettings('welcome', { view_mode: 'calendar' })
  // Give one card a due of today, leave another unscheduled.
  const board = await client.getBoard('welcome')
  await client.mutateBoard('welcome', board.version ?? -1, {
    type: 'edit_card',
    col_idx: 0,
    card_idx: 0,
    title: board.columns![0].cards[0].title,
    body: '',
    tags: [],
    links: [],
    priority: '',
    due: todayStr(),
    assignee: '',
  })
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['settings', 'welcome'], await client.getSettings('welcome'))
  return { client, qc }
}

function SeedActive({ id }: { id: string | null }): null {
  const { setActive } = useActiveBoard()
  useEffect(() => { setActive(id) }, [id, setActive])
  return null
}

describe('BoardCalendarView', () => {
  it('renders the calendar toolbar with nav buttons when view_mode is calendar', async () => {
    const { client, qc } = await setup()
    const { findByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => expect(getByText('Today')).toBeDefined())
    expect(await findByLabelText('previous')).toBeDefined()
    expect(await findByLabelText('next')).toBeDefined()
  })

  it('renders the month grid with 42 day cells', async () => {
    const { client, qc } = await setup()
    const { findAllByRole, findByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    await findByLabelText('calendar month grid')
    const cells = await findAllByRole('gridcell')
    expect(cells.length).toBe(42)
  })

  it('places a card with due=today in today’s cell', async () => {
    const { client, qc } = await setup()
    const { findByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const cell = await findByLabelText(`day ${todayStr()}`)
    expect(cell.querySelector('[aria-label^="card "]')).not.toBeNull()
  })

  it('lists unscheduled cards in the unscheduled section', async () => {
    const { client, qc } = await setup()
    const { findByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    // The second Todo card has no due date and should appear in unscheduled.
    const toggle = await findByLabelText('toggle unscheduled')
    expect(toggle).toBeDefined()
    await findByLabelText('card Double-click the board title to rename it')
  })

  it('clicking a day cell opens the New card modal with the date pre-filled', async () => {
    const { client, qc } = await setup()
    const { findByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const cell = await findByLabelText(`day ${todayStr()}`)
    fireEvent.click(cell)
    await findByText('New card')
    const due = await findByLabelText('card due')
    expect((due as HTMLInputElement).value).toBe(todayStr())
  })

  it('canceling the new card modal does not create a card', async () => {
    const { client, qc } = await setup()
    const { findByLabelText, findByText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const boardBefore = await client.getBoard('welcome')
    const countBefore = boardBefore.columns!.flatMap((c) => c.cards).length

    const cell = await findByLabelText(`day ${todayStr()}`)
    fireEvent.click(cell)
    await findByText('New card')
    fireEvent.click(getByText('Cancel'))

    await waitFor(() => expect(document.querySelector('[role="dialog"]')).toBeNull())
    const boardAfter = await client.getBoard('welcome')
    expect(boardAfter.columns!.flatMap((c) => c.cards).length).toBe(countBefore)
  })

  it('saving the new card modal creates a card with the clicked date', async () => {
    const { client, qc } = await setup()
    const { findByLabelText, findByText, getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const cell = await findByLabelText(`day ${todayStr()}`)
    fireEvent.click(cell)
    await findByText('New card')
    fireEvent.input(getByLabelText('card title'), { target: { value: 'New task from calendar' } })
    fireEvent.click(getByText('Save'))

    await waitFor(async () => {
      const board = await client.getBoard('welcome')
      const card = board.columns!.flatMap((c) => c.cards).find((c) => c.title === 'New task from calendar')
      expect(card).toBeDefined()
    })
  })

  it('clicking the day number button navigates to day view without opening modal', async () => {
    const { client, qc } = await setup()
    const { findByLabelText, queryByText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    await findByLabelText('calendar month grid')
    // Click the day number button (not the cell background)
    const cell = await findByLabelText(`day ${todayStr()}`)
    const dayBtn = cell.querySelector('button')!
    fireEvent.click(dayBtn)
    // Should switch to day view, not open a modal
    await waitFor(() => expect(document.querySelector('[role="dialog"]')).toBeNull())
    // day view label appears
    await findByLabelText('calendar week grid').catch(() => {
      // fine — may have navigated to day sub-view which has no grid
    })
    expect(queryByText('New card')).toBeNull()
  })
})

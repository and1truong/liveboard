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

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  // Persist view_mode=list in the board frontmatter so any refetch sees it.
  await client.putBoardSettings('welcome', { view_mode: 'list' })
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['settings', 'welcome'], await client.getSettings('welcome'))
  return { client, qc }
}

function SeedActive({ id }: { id: string | null }): null {
  const { setActive } = useActiveBoard()
  useEffect(() => { setActive(id) }, [id, setActive])
  return null
}

describe('BoardListView', () => {
  it('renders vertical sections for each column when view_mode is list', async () => {
    const { client, qc } = await setup()
    const { getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => expect(getByLabelText('section Todo')).toBeDefined())
    expect(getByLabelText('section Doing')).toBeDefined()
    expect(getByLabelText('section Done')).toBeDefined()
  })

  it('has the + Add list affordance when no column is focused', async () => {
    const { client, qc } = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(await findByText('+ Add list')).toBeDefined()
  })

  it('collapses a section when the chevron is clicked', async () => {
    const { client, qc } = await setup()
    const { findByLabelText, getByLabelText, queryByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const collapseBtn = await findByLabelText('collapse section Todo')
    expect(collapseBtn).toBeDefined()
    fireEvent.click(collapseBtn)
    await waitFor(() => expect(getByLabelText('expand section Todo')).toBeDefined())
    // Quick-add input for Todo disappears when collapsed.
    expect(queryByLabelText('new item in Todo')).toBeNull()
  })
})

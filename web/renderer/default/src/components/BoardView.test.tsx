import { describe, expect, it } from 'bun:test'
import { useEffect } from 'react'
import { fireEvent, waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider, useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { BoardView } from './BoardView.js'
import { renderWithQuery } from '../test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

function SeedActive({ id }: { id: string | null }): null {
  const { setActive } = useActiveBoard()
  useEffect(() => { setActive(id) }, [id, setActive])
  return null
}

describe('BoardView', () => {
  it('renders empty state when no board selected', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id={null} />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
    )
    expect(getByText('Select a board')).toBeDefined()
  })

  it('renders columns from the welcome board', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    expect(getByText('Doing')).toBeDefined()
    expect(getByText('Done')).toBeDefined()
  })

  it('self-recovers on NOT_FOUND (active becomes null, empty state appears)', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="nope" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
    )
    // Effect fires: setActive(null) → empty state appears.
    await waitFor(() => expect(getByText('Select a board')).toBeDefined())
  })

  it('renders only the focused column and an exit bar while in focus mode', async () => {
    const client = await setup()
    const { getByText, queryByText, getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    // Enter focus mode via the Todo column menu.
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    fireEvent.click(await findByText('Focus'))
    // Exit bar is visible.
    await waitFor(() => expect(getByText(/Focusing:/)).toBeDefined())
    // Other columns are no longer in the DOM.
    expect(queryByText('Doing')).toBeNull()
    expect(queryByText('Done')).toBeNull()
    // "Add list" button is hidden.
    expect(queryByText('+ Add list')).toBeNull()
  })

  it('exits focus mode via the exit bar button', async () => {
    const client = await setup()
    const { getByText, queryByText, getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <BoardView client={client} onToggleSidebar={() => {}} />
        </ActiveBoardProvider>
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    fireEvent.click(await findByText('Focus'))
    await waitFor(() => expect(getByText(/Focusing:/)).toBeDefined())
    fireEvent.click(getByText(/Exit Focus/))
    // All three columns back.
    await waitFor(() => expect(getByText('Doing')).toBeDefined())
    expect(getByText('Done')).toBeDefined()
    expect(queryByText(/Focusing:/)).toBeNull()
  })
})

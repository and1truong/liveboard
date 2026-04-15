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
import { renderWithQuery } from '../test-utils.js'
import { CommandPaletteHost } from './CommandPaletteHost.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['boards'], await client.listBoards())
  return { client, qc }
}

function SeedActive({ id }: { id: string | null }): null {
  const { setActive } = useActiveBoard()
  useEffect(() => { setActive(id) }, [id, setActive])
  return null
}

function ActiveProbe({ onChange }: { onChange: (id: string | null) => void }): null {
  const { active } = useActiveBoard()
  useEffect(() => { onChange(active) }, [active, onChange])
  return null
}

describe('CommandPalette', () => {
  it('Cmd+K opens the palette', async () => {
    const { client, qc } = await setup()
    const { queryByPlaceholderText, findByPlaceholderText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <CommandPaletteHost />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(queryByPlaceholderText('Type a command or board name…')).toBeNull()
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByPlaceholderText('Type a command or board name…')
  })

  it('lists boards from cache', async () => {
    const { client, qc } = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <CommandPaletteHost />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByText('Welcome')
  })

  it('selecting a board sets active and closes', async () => {
    const { client, qc } = await setup()
    let activeSeen: string | null = null
    const { findByText, queryByPlaceholderText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <ActiveProbe onChange={(v) => { activeSeen = v }} />
          <CommandPaletteHost />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    const item = await findByText('Welcome')
    fireEvent.click(item)
    await waitFor(() => expect(activeSeen).toBe('welcome'))
    await waitFor(() => expect(queryByPlaceholderText('Type a command or board name…')).toBeNull())
  })

  it('Rename current board hidden when no active board', async () => {
    const { client, qc } = await setup()
    const { queryByText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <CommandPaletteHost />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByText('Create board')
    expect(queryByText('Rename current board')).toBeNull()
    expect(queryByText('Delete current board')).toBeNull()
  })

  it('Rename current board visible when active is set', async () => {
    const { client, qc } = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <SeedActive id="welcome" />
          <CommandPaletteHost />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.keyDown(window, { key: 'k', metaKey: true })
    await findByText('Rename current board')
  })
})

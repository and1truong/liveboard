import { describe, expect, it } from 'bun:test'
import { waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import type { BackendAdapter } from '@shared/adapter.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import { ThemeProvider } from '../contexts/ThemeContext.js'
import { BoardSidebar } from './BoardSidebar.js'
import { renderWithQuery } from '../test-utils.js'

async function setup(adapter?: BackendAdapter): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, adapter ?? new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

// Wraps LocalAdapter to additionally advertise the 'folders' capability, so
// the sidebar shows folder grouping UI as it would against the real server.
function withFolderCapability(a: BackendAdapter): BackendAdapter {
  return new Proxy(a, {
    get(target, prop, receiver) {
      if (prop === 'capabilities') return () => [...target.capabilities(), 'folders']
      const v = Reflect.get(target, prop, receiver)
      return typeof v === 'function' ? v.bind(target) : v
    },
  })
}

describe('BoardSidebar', () => {
  it('lists boards from the adapter', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ThemeProvider>
          <ActiveBoardProvider>
            <BoardSidebar />
          </ActiveBoardProvider>
        </ThemeProvider>
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
  })

  it('renders the Board pill affordance', async () => {
    const client = await setup()
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ThemeProvider>
          <ActiveBoardProvider>
            <BoardSidebar />
          </ActiveBoardProvider>
        </ThemeProvider>
      </ClientProvider>,
    )
    await findByText('Board')
  })

  it('renders the Folder pill affordance when backend has the folders capability', async () => {
    const adapter = withFolderCapability(new LocalAdapter(new MemoryStorage()))
    const client = await setup(adapter)
    const { findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ThemeProvider>
          <ActiveBoardProvider>
            <BoardSidebar />
          </ActiveBoardProvider>
        </ThemeProvider>
      </ClientProvider>,
    )
    await findByText('Folder')
  })

  it('hides the Folder pill affordance when backend lacks the folders capability', async () => {
    // LocalAdapter does not advertise 'folders' — the UI should stay flat.
    const client = await setup()
    const { findByText, queryByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ThemeProvider>
          <ActiveBoardProvider>
            <BoardSidebar />
          </ActiveBoardProvider>
        </ThemeProvider>
      </ClientProvider>,
    )
    await findByText('Board')
    expect(queryByText('Folder')).toBeNull()
  })

  it('groups nested boards under a folder header when capability is present', async () => {
    const base = new LocalAdapter(new MemoryStorage())
    await base.createBoard('Ideas', 'Work')
    const client = await setup(withFolderCapability(base))

    const { findByText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ThemeProvider>
          <ActiveBoardProvider>
            <BoardSidebar />
          </ActiveBoardProvider>
        </ThemeProvider>
      </ClientProvider>,
    )
    // The folder header shows the folder name.
    await findByText('Work')
    // And the nested board shows its display name.
    await waitFor(() => expect(getByText('Ideas')).toBeDefined())
  })
})

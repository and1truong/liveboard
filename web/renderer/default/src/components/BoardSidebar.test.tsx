import { describe, expect, it } from 'bun:test'
import { waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import { ThemeProvider } from '../contexts/ThemeContext.js'
import { BoardSidebar } from './BoardSidebar.js'
import { renderWithQuery } from '../test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
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

  it('renders + New board affordance', async () => {
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
    await findByText('+ New board')
  })
})

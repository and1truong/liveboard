import { describe, expect, it } from 'bun:test'
import { waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { BoardView } from './BoardView.js'
import { renderWithQuery } from '../test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  const adapter = new LocalAdapter(new MemoryStorage())
  new Broker(shellT, adapter, { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

describe('BoardView', () => {
  it('renders empty state when no board selected', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardView boardId={null} client={client} />
      </ClientProvider>,
    )
    expect(getByText('Select a board')).toBeDefined()
  })

  it('renders columns from the welcome board', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardView boardId="welcome" client={client} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
    expect(getByText('Doing')).toBeDefined()
    expect(getByText('Done')).toBeDefined()
  })

  it('shows error state when board is missing', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardView boardId="nope" client={client} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Failed to load board')).toBeDefined())
  })
})

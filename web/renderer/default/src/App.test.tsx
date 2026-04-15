import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { App } from './App.js'
import { ClientProvider } from './queries.js'
import { renderWithQuery } from './test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

describe('App integration', () => {
  it('selecting a board renders its columns', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <App client={client} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    fireEvent.click(getByText('Welcome'))
    await waitFor(() => expect(getByText('Todo')).toBeDefined())
  })

  it('mutation bumps board version', async () => {
    const client = await setup()
    renderWithQuery(
      <ClientProvider client={client}>
        <App client={client} />
      </ClientProvider>,
    )
    await client.subscribe('welcome')
    await client.mutateBoard('welcome', 1, {
      type: 'add_card',
      column: 'Todo',
      title: 'LIVE-ADDED',
    })
    const list = await client.listBoards()
    const welcome = list.find((b) => b.id === 'welcome')!
    expect(welcome.version).toBeGreaterThanOrEqual(2)
  })
})

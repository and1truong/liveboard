import { describe, expect, it } from 'bun:test'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { Card } from './Card.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

describe('Card', () => {
  it('renders title', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}><Card card={{ title: 'Hello' }} /></ClientProvider>,
    )
    expect(getByText('Hello')).toBeDefined()
  })
  it('renders tags as pills', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}><Card card={{ title: 'x', tags: ['a', 'b'] }} /></ClientProvider>,
    )
    expect(getByText('a')).toBeDefined()
    expect(getByText('b')).toBeDefined()
  })
  it('shows priority dot when priority set', async () => {
    const client = await setup()
    const { getByLabelText } = renderWithQuery(
      <ClientProvider client={client}><Card card={{ title: 'x', priority: 'high' }} /></ClientProvider>,
    )
    expect(getByLabelText('priority high')).toBeDefined()
  })
  it('dims completed cards', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}><Card card={{ title: 'done', completed: true }} /></ClientProvider>,
    )
    expect(getByText('done').className).toContain('text-slate-400')
  })
})

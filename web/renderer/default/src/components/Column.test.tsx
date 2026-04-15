import { describe, expect, it } from 'bun:test'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { QueryClient } from '@tanstack/react-query'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { Column } from './Column.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('Column', () => {
  it('renders name and count', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <Column
          column={{ name: 'Todo', cards: [{ title: 'a' }, { title: 'b' }] }}
          colIdx={0}
          allColumnNames={['Todo']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('Todo')).toBeDefined()
    expect(getByText('2')).toBeDefined()
  })

  it('renders all cards', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <Column
          column={{ name: 'x', cards: [{ title: 'A' }, { title: 'B' }] }}
          colIdx={0}
          allColumnNames={['x']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('A')).toBeDefined()
    expect(getByText('B')).toBeDefined()
  })

  it('handles empty cards array', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <Column
          column={{ name: 'Empty', cards: [] }}
          colIdx={0}
          allColumnNames={['Empty']}
          boardId="welcome"
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('0')).toBeDefined()
  })
})

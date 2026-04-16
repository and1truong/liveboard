import { describe, expect, it } from 'bun:test'
import { fireEvent } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { ColumnHeader } from './ColumnHeader.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('ColumnHeader', () => {
  it('renders name, count, and menu trigger', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader name="Todo" cardCount={3} colIdx={0} allColumnNames={['Todo','Doing','Done']} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('Todo')).toBeDefined()
    expect(getByText('3')).toBeDefined()
    expect(getByLabelText('column menu Todo')).toBeDefined()
  })

  it('renders "Collapse" item when expanded', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo']} boardId="welcome" collapsed={false} />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    expect(await findByText('Collapse')).toBeDefined()
  })

  it('renders "Expand" item when collapsed', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo']} boardId="welcome" collapsed={true} />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    expect(await findByText('Expand')).toBeDefined()
  })
})

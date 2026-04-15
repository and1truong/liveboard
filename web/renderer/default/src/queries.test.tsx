import { describe, expect, it } from 'bun:test'
import { waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider, useBoard, useBoardList } from './queries.js'
import { renderWithQuery } from './test-utils.js'

function setup(): Client {
  const [iframeT, shellT] = createMemoryPair()
  const adapter = new LocalAdapter(new MemoryStorage())
  new Broker(shellT, adapter, { shellVersion: 't' })
  return new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
}

function BoardListProbe(): JSX.Element {
  const q = useBoardList()
  if (q.isLoading) return <p>loading</p>
  if (q.error) return <p>err</p>
  return <ul>{q.data?.map((b) => <li key={b.id}>{b.name}</li>)}</ul>
}

function BoardProbe({ id }: { id: string }): JSX.Element {
  const q = useBoard(id)
  if (q.isLoading) return <p>loading</p>
  if (q.error) return <p>err</p>
  return <h2>{q.data?.name}</h2>
}

describe('queries', () => {
  it('useBoardList returns the welcome board', async () => {
    const client = setup()
    await client.ready()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardListProbe />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
  })

  it('useBoard returns the board by id', async () => {
    const client = setup()
    await client.ready()
    const { getByRole } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardProbe id="welcome" />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByRole('heading', { name: 'Welcome' })).toBeDefined())
  })

  it('invalidates board query when board.updated fires', async () => {
    const client = setup()
    await client.ready()
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    client.on('board.updated', ({ boardId }) =>
      void qc.invalidateQueries({ queryKey: ['board', boardId] }),
    )
    await client.subscribe('welcome')

    const { rerender, queryByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardProbe id="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => expect(queryByText('Welcome')).toBeDefined())

    await client.mutateBoard('welcome', 1, { type: 'add_card', column: 'Todo', title: 'z' })
    rerender(
      <ClientProvider client={client}>
        <BoardProbe id="welcome" />
      </ClientProvider>,
    )
    await new Promise((r) => setTimeout(r, 20))
    expect(queryByText('Welcome')).toBeDefined()
  })
})

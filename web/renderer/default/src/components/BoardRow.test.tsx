import { describe, expect, it } from 'bun:test'
import { fireEvent } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import { renderWithQuery } from '../test-utils.js'
import { BoardRow } from './BoardRow.js'

async function setup() {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['boards'], await client.listBoards())
  return { client, qc }
}

const board = { id: 'foo', name: 'Foo', version: 1 }

describe('BoardRow', () => {
  it('renders name', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <ul><BoardRow board={board} /></ul>
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('Foo')).toBeDefined()
  })

  it('clicking the row attempts to set active (smoke)', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <ul><BoardRow board={board} /></ul>
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('Foo'))
  })
})

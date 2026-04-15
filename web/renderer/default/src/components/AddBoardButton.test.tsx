import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import { renderWithQuery } from '../test-utils.js'
import { AddBoardButton } from './AddBoardButton.js'

async function setup() {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['boards'], await client.listBoards())
  return { client, qc }
}

describe('AddBoardButton', () => {
  it('click reveals input', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <AddBoardButton />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ New board'))
    await waitFor(() => expect(getByLabelText('new board name')).toBeDefined())
  })

  it('blur with text creates a new board (sidebar list grows)', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <AddBoardButton />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ New board'))
    const input = await waitFor(() => getByLabelText('new board name')) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'Foo' } })
    fireEvent.blur(input)
    await waitFor(async () => {
      const list = await client.listBoards()
      expect(list.map((s) => s.id)).toContain('foo')
    })
  })
})

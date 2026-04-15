import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { AddColumnButton } from './AddColumnButton.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('AddColumnButton', () => {
  it('click reveals input', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddColumnButton boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add column'))
    await waitFor(() => expect(getByLabelText('new column name')).toBeDefined())
  })

  it('blur with text commits add_column', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddColumnButton boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add column'))
    const input = await waitFor(() => getByLabelText('new column name')) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'Review' } })
    fireEvent.blur(input)
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      expect(b.columns.some((c: any) => c.name === 'Review')).toBe(true)
    })
  })
})

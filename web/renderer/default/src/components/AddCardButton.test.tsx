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
import { AddCardButton } from './AddCardButton.js'

async function setup(): Promise<{ client: Client; qc: QueryClient; columnName: string }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const board = await client.getBoard('welcome')
  qc.setQueryData(['board', 'welcome'], board)
  return { client, qc, columnName: board.columns![0].name }
}

describe('AddCardButton', () => {
  it('click reveals input', async () => {
    const { client, qc, columnName } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddCardButton columnName={columnName} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add card'))
    await waitFor(() => expect(getByLabelText(`new card in ${columnName}`)).toBeDefined())
  })

  it('blur with text commits add_card', async () => {
    const { client, qc, columnName } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddCardButton columnName={columnName} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add card'))
    const input = await waitFor(() => getByLabelText(`new card in ${columnName}`)) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'NEW' } })
    fireEvent.blur(input)
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      const col = b.columns.find((c: any) => c.name === columnName)
      expect(col.cards.some((c: any) => c.title === 'NEW')).toBe(true)
    })
  })

  it('blur with empty input cancels without mutation', async () => {
    const { client, qc, columnName } = await setup()
    const before = qc.getQueryData<any>(['board', 'welcome'])
    const beforeCount = before.columns.find((c: any) => c.name === columnName).cards.length
    const { getByText, getByLabelText, queryByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <AddCardButton columnName={columnName} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByText('+ Add card'))
    const input = await waitFor(() => getByLabelText(`new card in ${columnName}`))
    fireEvent.blur(input)
    await waitFor(() => expect(queryByLabelText(`new card in ${columnName}`)).toBeNull())
    const after = qc.getQueryData<any>(['board', 'welcome'])
    expect(after.columns.find((c: any) => c.name === columnName).cards.length).toBe(beforeCount)
  })
})

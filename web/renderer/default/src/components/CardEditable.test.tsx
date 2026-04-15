import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { QueryClient } from '@tanstack/react-query'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { CardEditable } from './CardEditable.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('CardEditable', () => {
  it('double-click switches to edit mode', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable card={{ title: 'hello' }} colIdx={0} cardIdx={0} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.doubleClick(getByText('hello'))
    await waitFor(() => expect(getByLabelText('card title')).toBeDefined())
  })

  it('blur without change returns to view mode without mutation', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText, queryByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable card={{ title: 'hello' }} colIdx={0} cardIdx={0} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.doubleClick(getByText('hello'))
    const input = await waitFor(() => getByLabelText('card title'))
    // Blur without changing the value — should not mutate and return to view
    fireEvent.blur(input)
    await waitFor(() => expect(queryByLabelText('card title')).toBeNull())
    expect(getByText('hello')).toBeDefined()
  })

  it('blur after edit commits edit_card mutation', async () => {
    const { client, qc } = await setup()
    // Use the actual first card title from the seed so col_idx/card_idx 0/0 matches.
    const seed = qc.getQueryData<any>(['board', 'welcome'])
    const firstTitle = seed.columns[0].cards[0].title
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable card={{ title: firstTitle }} colIdx={0} cardIdx={0} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.doubleClick(getByText(firstTitle))
    const input = await waitFor(() => getByLabelText('card title'))
    // fireEvent.change sets the DOM value; the uncontrolled input reads it via ref on blur.
    fireEvent.change(input, { target: { value: 'NEW TITLE' } })
    fireEvent.blur(input)
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      expect(b?.columns?.[0]?.cards?.[0]?.title).toBe('NEW TITLE')
    })
  })

  it('complete button fires complete_card mutation', async () => {
    const { client, qc } = await setup()
    const seed = qc.getQueryData<any>(['board', 'welcome'])
    const firstTitle = seed.columns[0].cards[0].title
    const { getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardEditable card={{ title: firstTitle }} colIdx={0} cardIdx={0} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(getByLabelText('mark complete'))
    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      expect(b?.columns?.[0]?.cards?.[0]?.completed).toBe(true)
    })
  })
})

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
import { CardDetailModal } from './CardDetailModal.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

const seed = {
  title: 'Hello',
  body: 'orig body',
  tags: ['a', 'b'],
  priority: 'high',
  due: '2026-05-01',
  assignee: 'alice',
}

describe('CardDetailModal', () => {
  it('renders form seeded from card prop when open', async () => {
    const { client, qc } = await setup()
    const { getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={() => {}}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => expect((getByLabelText('card title') as HTMLInputElement).value).toBe('Hello'))
    expect((getByLabelText('card body') as HTMLTextAreaElement).value).toBe('orig body')
    expect((getByLabelText('card tags') as HTMLInputElement).value).toBe('a, b')
    expect((getByLabelText('card priority') as HTMLSelectElement).value).toBe('high')
    expect((getByLabelText('card due') as HTMLInputElement).value).toBe('2026-05-01')
    expect((getByLabelText('card assignee') as HTMLInputElement).value).toBe('alice')
  })

  it('Save fires edit_card with form values and closes modal', async () => {
    const { client, qc } = await setup()
    const calls: boolean[] = []
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={(next) => calls.push(next)}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => getByLabelText('card title'))
    fireEvent.input(getByLabelText('card title'), { target: { value: 'NEW TITLE' } })
    fireEvent.input(getByLabelText('card tags'), { target: { value: 'x, y, z' } })
    fireEvent.click(getByText('Save'))

    await waitFor(() => expect(calls).toContain(false))

    const b = qc.getQueryData<any>(['board', 'welcome'])
    const updated = b.columns[0].cards[0]
    expect(updated.title).toBe('NEW TITLE')
    expect(updated.tags).toEqual(['x', 'y', 'z'])
  })

  it('Cancel closes without firing mutation', async () => {
    const { client, qc } = await setup()
    const before = qc.getQueryData<any>(['board', 'welcome'])
    const beforeTitle = before.columns[0].cards[0].title
    const calls: boolean[] = []
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={(next) => calls.push(next)}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => getByLabelText('card title'))
    fireEvent.input(getByLabelText('card title'), { target: { value: 'WONT SAVE' } })
    fireEvent.click(getByText('Cancel'))

    expect(calls).toContain(false)
    const after = qc.getQueryData<any>(['board', 'welcome'])
    expect(after.columns[0].cards[0].title).toBe(beforeTitle)
  })

  it('empty title disables Save button', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardDetailModal
          card={seed}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          open={true}
          onOpenChange={() => {}}
        />
      </ClientProvider>,
      { queryClient: qc },
    )
    await waitFor(() => getByLabelText('card title'))
    fireEvent.input(getByLabelText('card title'), { target: { value: '   ' } })
    expect((getByText('Save') as HTMLButtonElement).disabled).toBe(true)
  })
})

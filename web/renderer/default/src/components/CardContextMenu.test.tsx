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
import { CardContextMenu } from './CardContextMenu.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

function openContextMenu(el: HTMLElement): void {
  fireEvent.contextMenu(el)
}

describe('CardContextMenu', () => {
  it('opens on right-click and shows actions', async () => {
    const { client, qc } = await setup()
    const { getByTestId, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardContextMenu
          card={{ title: 'Hi', body: '', tags: [], links: [], completed: false }}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          allColumnNames={['Todo', 'Doing', 'Done']}
          onQuickEdit={() => {}}
          onOpenDetail={() => {}}
        >
          <div data-testid="card">Hi</div>
        </CardContextMenu>
      </ClientProvider>,
      { queryClient: qc },
    )
    openContextMenu(getByTestId('card'))
    expect(await findByText('Quick edit')).toBeDefined()
    expect(await findByText('Open details')).toBeDefined()
    expect(await findByText('Mark complete')).toBeDefined()
    expect(await findByText('Delete')).toBeDefined()
  })

  it('fires onQuickEdit when Quick edit item selected', async () => {
    const { client, qc } = await setup()
    let fired = false
    const { getByTestId, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <CardContextMenu
          card={{ title: 'Hi', body: '', tags: [], links: [], completed: false }}
          colIdx={0}
          cardIdx={0}
          boardId="welcome"
          allColumnNames={['Todo']}
          onQuickEdit={() => { fired = true }}
          onOpenDetail={() => {}}
        >
          <div data-testid="card">Hi</div>
        </CardContextMenu>
      </ClientProvider>,
      { queryClient: qc },
    )
    openContextMenu(getByTestId('card'))
    const item = await findByText('Quick edit')
    fireEvent.click(item)
    expect(fired).toBe(true)
  })
})

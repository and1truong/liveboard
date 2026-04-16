import { describe, expect, it } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import * as ContextMenu from '@radix-ui/react-context-menu'
import { QueryClient } from '@tanstack/react-query'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { MoveToBoardSubmenu } from './MoveToBoardSubmenu.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  const storage = new MemoryStorage()
  new Broker(shellT, new LocalAdapter(storage), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  qc.setQueryData(['boards-lite'], await client.listBoardsLite())
  return { client, qc }
}

describe('MoveToBoardSubmenu', () => {
  it('renders the Move to board entry', async () => {
    const { client, qc } = await setup()
    // Seed a second board via client so lite listing has a target.
    await client.createBoard('other')
    qc.setQueryData(['boards-lite'], await client.listBoardsLite())

    const { getByTestId, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ContextMenu.Root>
          <ContextMenu.Trigger asChild>
            <div data-testid="t">trigger</div>
          </ContextMenu.Trigger>
          <ContextMenu.Portal>
            <ContextMenu.Content>
              <MoveToBoardSubmenu
                srcBoardId="welcome"
                colIdx={0}
                cardIdx={0}
                triggerCls=""
                contentCls=""
                itemCls=""
              />
            </ContextMenu.Content>
          </ContextMenu.Portal>
        </ContextMenu.Root>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.contextMenu(getByTestId('t'))
    await waitFor(async () => {
      expect(await findByText(/Move to board/)).toBeDefined()
    })
  })
})

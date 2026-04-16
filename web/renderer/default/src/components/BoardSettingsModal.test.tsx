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
import { BoardSettingsModal } from './BoardSettingsModal.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['settings', 'welcome'], await client.getSettings('welcome'))
  return { client, qc }
}

describe('BoardSettingsModal', () => {
  it('renders form seeded from settings cache', async () => {
    const { client, qc } = await setup()
    const { findByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <BoardSettingsModal
          boardId="welcome"
          boardName="Welcome"
          open={true}
          onOpenChange={() => {}}
          />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const checkbox = (await findByLabelText('show complete checkbox')) as HTMLInputElement
    const select = (await findByLabelText('card display mode')) as HTMLSelectElement
    expect(checkbox.checked).toBe(true)
    expect(select.value).toBe('normal')
  })

  it('Save persists the toggled values', async () => {
    const { client, qc } = await setup()
    const calls: boolean[] = []
    const { findByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <BoardSettingsModal
          boardId="welcome"
          boardName="Welcome"
          open={true}
          onOpenChange={(v) => calls.push(v)}
          />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const checkbox = (await findByLabelText('show complete checkbox')) as HTMLInputElement
    const select = (await findByLabelText('card display mode')) as HTMLSelectElement
    fireEvent.click(checkbox)
    fireEvent.change(select, { target: { value: 'compact' } })
    fireEvent.click(await findByText('Save'))

    await waitFor(() => expect(calls).toContain(false))

    const after = await client.getSettings('welcome')
    expect(after.show_checkbox).toBe(false)
    expect(after.card_display_mode).toBe('compact')
  })

  it('Save persists the chosen view mode', async () => {
    const { client, qc } = await setup()
    const { findByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <BoardSettingsModal
          boardId="welcome"
          boardName="Welcome"
          open={true}
          onOpenChange={() => {}}
          />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.click(await findByLabelText('view mode list'))
    fireEvent.click(await findByText('Save'))
    await waitFor(async () => {
      const after = await client.getSettings('welcome')
      expect(after.view_mode).toBe('list')
    })
  })

  it('Cancel closes without writing', async () => {
    const { client, qc } = await setup()
    const calls: boolean[] = []
    const before = await client.getSettings('welcome')
    const { findByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider>
          <BoardSettingsModal
          boardId="welcome"
          boardName="Welcome"
          open={true}
          onOpenChange={(v) => calls.push(v)}
          />
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    const checkbox = (await findByLabelText('show complete checkbox')) as HTMLInputElement
    fireEvent.click(checkbox)
    fireEvent.click(await findByText('Cancel'))
    expect(calls).toContain(false)
    const after = await client.getSettings('welcome')
    expect(after.show_checkbox).toBe(before.show_checkbox)
  })
})

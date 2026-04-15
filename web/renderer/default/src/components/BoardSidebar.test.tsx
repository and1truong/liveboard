import { describe, expect, it, mock } from 'bun:test'
import { fireEvent, waitFor } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { BoardSidebar } from './BoardSidebar.js'
import { renderWithQuery } from '../test-utils.js'

async function setup(): Promise<Client> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  return client
}

describe('BoardSidebar', () => {
  it('lists boards from the adapter', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSidebar activeId={null} onSelect={() => {}} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
  })

  it('fires onSelect with board id on click', async () => {
    const client = await setup()
    const onSelect = mock(() => {})
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSidebar activeId={null} onSelect={onSelect} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    fireEvent.click(getByText('Welcome'))
    expect(onSelect).toHaveBeenCalledWith('welcome')
  })

  it('highlights active board', async () => {
    const client = await setup()
    const { getByText } = renderWithQuery(
      <ClientProvider client={client}>
        <BoardSidebar activeId="welcome" onSelect={() => {}} />
      </ClientProvider>,
    )
    await waitFor(() => expect(getByText('Welcome')).toBeDefined())
    const btn = getByText('Welcome').closest('button')!
    expect(btn.className).toContain('bg-slate-200')
  })
})

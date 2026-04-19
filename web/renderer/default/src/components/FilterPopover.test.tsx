import { describe, expect, it, beforeEach } from 'bun:test'
import { useState } from 'react'
import { fireEvent } from '@testing-library/react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { QueryClient } from '@tanstack/react-query'
import { BoardFilterProvider, useBoardFilter } from '../contexts/BoardFilterContext.js'
import { ClientProvider } from '../queries.js'
import { renderWithQuery } from '../test-utils.js'
import { FilterPopover } from './FilterPopover.js'

const BOARD_TAGS = ['frontend', 'backend', 'urgent']

function Harness({ availableTags }: { availableTags: string[] }): JSX.Element {
  const [open, setOpen] = useState(false)
  return (
    <BoardFilterProvider boardId="test" availableTags={availableTags}>
      <FilterPopover
        availableTags={availableTags}
        open={open}
        onOpenChange={setOpen}
      />
      <FilterReadout />
      <button type="button" onClick={() => setOpen(true)}>force-open</button>
    </BoardFilterProvider>
  )
}

function FilterReadout(): JSX.Element {
  const { filter } = useBoardFilter()
  return (
    <div data-testid="readout">
      q={filter.query}|tags={filter.tags.join(',')}|prio={filter.priorities.join(',')}|hide={String(filter.hideCompleted)}
    </div>
  )
}

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return { client, qc }
}

function renderHarness(
  client: Client,
  qc: QueryClient,
  availableTags: string[],
): ReturnType<typeof renderWithQuery> {
  return renderWithQuery(
    <ClientProvider client={client}>
      <Harness availableTags={availableTags} />
    </ClientProvider>,
    { queryClient: qc },
  )
}

describe('FilterPopover', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('toggles a tag selection on click', async () => {
    const { client, qc } = await setup()
    const { getByText, getByRole, getByTestId } = renderHarness(client, qc, BOARD_TAGS)
    fireEvent.click(getByText('force-open'))
    const frontend = getByRole('checkbox', { name: 'frontend' })
    expect(frontend.getAttribute('aria-checked')).toBe('false')
    fireEvent.click(frontend)
    expect(getByTestId('readout').textContent).toContain('tags=frontend')
    fireEvent.click(frontend)
    expect(getByTestId('readout').textContent).toContain('tags=|')
  })

  it('toggles a priority selection on click', async () => {
    const { client, qc } = await setup()
    const { getByText, getByRole, getByTestId } = renderHarness(client, qc, BOARD_TAGS)
    fireEvent.click(getByText('force-open'))
    const high = getByRole('checkbox', { name: 'High' })
    expect(high.getAttribute('aria-checked')).toBe('false')
    fireEvent.click(high)
    expect(getByTestId('readout').textContent).toContain('prio=high')
    fireEvent.click(high)
    expect(getByTestId('readout').textContent).toContain('prio=|')
  })

  it('reset clears all filters', async () => {
    const { client, qc } = await setup()
    const { getByText, getByRole, getByTestId } = renderHarness(client, qc, BOARD_TAGS)
    fireEvent.click(getByText('force-open'))
    fireEvent.click(getByRole('checkbox', { name: 'urgent' }))
    fireEvent.click(getByRole('checkbox', { name: 'frontend' }))
    fireEvent.click(getByRole('checkbox', { name: 'High' }))
    fireEvent.click(getByRole('switch', { name: /Hide completed/i }))

    const readout = getByTestId('readout')
    expect(readout.textContent).toBe('q=|tags=urgent,frontend|prio=high|hide=true')

    fireEvent.click(getByRole('button', { name: 'Reset' }))
    expect(readout.textContent).toBe('q=|tags=|prio=|hide=false')
  })

  it('shows empty-state message when there are no tags', async () => {
    const { client, qc } = await setup()
    const { getByText } = renderHarness(client, qc, [])
    fireEvent.click(getByText('force-open'))
    expect(getByText('No tags on this board yet.')).toBeTruthy()
  })
})

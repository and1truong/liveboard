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
import { ColumnHeader } from './ColumnHeader.js'
import { useEffect } from 'react'
import type { Column } from '@shared/types.js'
import { ActiveBoardProvider } from '../contexts/ActiveBoardContext.js'
import {
  FocusedColumnProvider,
  useFocusedColumn,
} from '../contexts/FocusedColumnContext.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))
  return { client, qc }
}

describe('ColumnHeader', () => {
  it('renders name, count, and menu trigger', async () => {
    const { client, qc } = await setup()
    const { getByText, getByLabelText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader name="Todo" cardCount={3} colIdx={0} allColumnNames={['Todo','Doing','Done']} boardId="welcome" />
      </ClientProvider>,
      { queryClient: qc },
    )
    expect(getByText('Todo')).toBeDefined()
    expect(getByText('3')).toBeDefined()
    expect(getByLabelText('column menu Todo')).toBeDefined()
  })

  it('renders "Collapse" item when expanded', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo']} boardId="welcome" collapsed={false} />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    expect(await findByText('Collapse')).toBeDefined()
  })

  it('renders "Expand" item when collapsed', async () => {
    const { client, qc } = await setup()
    const { getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo']} boardId="welcome" collapsed={true} />
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    expect(await findByText('Expand')).toBeDefined()
  })

  it('shows a Focus menu item that sets the focused column', async () => {
    const { client, qc } = await setup()
    const colsList: Column[] = [
      { name: 'Todo', cards: [] },
      { name: 'Doing', cards: [] },
    ]
    let currentFocused: string | null = null
    function Spy(): null {
      currentFocused = useFocusedColumn().focused
      return null
    }
    const { getByLabelText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider initialBoardId="welcome">
          <FocusedColumnProvider columns={colsList}>
            <Spy />
            <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo','Doing']} boardId="welcome" />
          </FocusedColumnProvider>
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    fireEvent.click(await findByText('Focus'))
    expect(currentFocused as string | null).toBe('Todo')
  })

  it('hides the Focus item when this column is already focused', async () => {
    const { client, qc } = await setup()
    const colsList: Column[] = [{ name: 'Todo', cards: [] }]
    function Seed(): null {
      const { setFocused } = useFocusedColumn()
      useEffect(() => {
        setFocused('Todo')
      }, [setFocused])
      return null
    }
    const { getByLabelText, queryByText, findByText } = renderWithQuery(
      <ClientProvider client={client}>
        <ActiveBoardProvider initialBoardId="welcome">
          <FocusedColumnProvider columns={colsList}>
            <Seed />
            <ColumnHeader name="Todo" cardCount={0} colIdx={0} allColumnNames={['Todo']} boardId="welcome" />
          </FocusedColumnProvider>
        </ActiveBoardProvider>
      </ClientProvider>,
      { queryClient: qc },
    )
    fireEvent.pointerDown(getByLabelText('column menu Todo'), { button: 0, pointerType: 'mouse' })
    // Rename should still be present in both states.
    expect(await findByText('Rename')).toBeDefined()
    expect(queryByText('Focus')).toBeNull()
  })
})

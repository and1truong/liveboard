import { describe, expect, it } from 'bun:test'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { ActiveBoardProvider, useActiveBoard } from '../contexts/ActiveBoardContext.js'
import { useCreateBoard, useRenameBoard, useDeleteBoard } from './useBoardCrud.js'

async function setup() {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['boards'], await client.listBoards())
  const wrap = (children: ReactNode): JSX.Element => (
    <ClientProvider client={client}>
      <QueryClientProvider client={qc}>
        <ActiveBoardProvider>{children}</ActiveBoardProvider>
      </QueryClientProvider>
    </ClientProvider>
  )
  return { client, qc, wrap }
}

function combined() {
  return {
    create: useCreateBoard(),
    rename: useRenameBoard(),
    del: useDeleteBoard(),
    ab: useActiveBoard(),
  }
}

describe('useBoardCrud', () => {
  it('useCreateBoard sets new board active', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.create.isSuccess).toBe(true))
    expect(result.current.ab.active).toBe('foo')
  })

  it('useRenameBoard switches active to new id when active was renamed', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.ab.active).toBe('foo'))
    await act(async () => {
      result.current.rename.mutate({ boardId: 'foo', newName: 'Bar' })
    })
    await waitFor(() => expect(result.current.ab.active).toBe('bar'))
  })

  it('useRenameBoard leaves active untouched when renaming a different board', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.ab.active).toBe('foo'))
    await act(async () => {
      result.current.rename.mutate({ boardId: 'welcome', newName: 'Welcomed' })
    })
    await waitFor(() => expect(result.current.rename.isSuccess).toBe(true))
    expect(result.current.ab.active).toBe('foo')
  })

  it('useDeleteBoard switches active to first remaining when active was deleted', async () => {
    const { wrap, qc, client } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Foo')
    })
    await waitFor(() => expect(result.current.ab.active).toBe('foo'))
    qc.setQueryData(['boards'], await client.listBoards())
    await act(async () => {
      result.current.del.mutate('foo')
    })
    await waitFor(() => expect(result.current.del.isSuccess).toBe(true))
    expect(result.current.ab.active).toBe('welcome')
  })

  it('useCreateBoard surfaces ALREADY_EXISTS via toast (no throw)', async () => {
    const { wrap } = await setup()
    const { result } = renderHook(combined, { wrapper: ({ children }) => wrap(children) })
    await act(async () => {
      result.current.create.mutate('Welcome')
    })
    await waitFor(() => expect(result.current.create.isError).toBe(true))
  })
})

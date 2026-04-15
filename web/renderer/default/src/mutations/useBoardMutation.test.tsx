import { describe, expect, it } from 'bun:test'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Broker } from '@shared/broker.js'
import { Client } from '@shared/client.js'
import { LocalAdapter } from '@shared/adapters/local.js'
import { MemoryStorage } from '@shared/adapters/local-storage-driver.js'
import { createMemoryPair } from '@shared/transport.js'
import { ClientProvider } from '../queries.js'
import { useBoardMutation } from './useBoardMutation.js'

async function setup(): Promise<{ client: Client; qc: QueryClient }> {
  const [iframeT, shellT] = createMemoryPair()
  new Broker(shellT, new LocalAdapter(new MemoryStorage()), { shellVersion: 't' })
  const client = new Client(iframeT, { rendererId: 't', rendererVersion: '0' })
  await client.ready()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return { client, qc }
}

function wrap(client: Client, qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <ClientProvider client={client}>
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    </ClientProvider>
  )
}

describe('useBoardMutation', () => {
  it('optimistically applies add_card and server confirms', async () => {
    const { client, qc } = await setup()
    qc.setQueryData(['board', 'welcome'], await client.getBoard('welcome'))

    const { result } = renderHook(() => useBoardMutation('welcome'), { wrapper: wrap(client, qc) })

    result.current.mutate({ type: 'add_card', column: 'Todo', title: 'OPT' })

    await waitFor(() => {
      const b = qc.getQueryData<any>(['board', 'welcome'])
      const todo = b?.columns?.find((c: any) => c.name === 'Todo')
      expect(todo?.cards?.some((c: any) => c.title === 'OPT')).toBe(true)
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const final = qc.getQueryData<any>(['board', 'welcome'])
    expect(final.version).toBeGreaterThanOrEqual(2)
  })

  it('rolls back on VERSION_CONFLICT and invalidates', async () => {
    const { client, qc } = await setup()
    const real = await client.getBoard('welcome')
    qc.setQueryData(['board', 'welcome'], { ...real, version: 0 })

    const { result } = renderHook(() => useBoardMutation('welcome'), { wrapper: wrap(client, qc) })

    result.current.mutate({ type: 'add_card', column: 'Todo', title: 'BAD' })
    await waitFor(() => expect(result.current.isError).toBe(true))

    const after = qc.getQueryData<any>(['board', 'welcome'])
    expect(after.version).toBe(0)
  })
})

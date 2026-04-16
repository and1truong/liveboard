import { QueryClient } from '@tanstack/react-query'
import { Client } from '@shared/client.js'
import { iframeTransport } from '@shared/transports/post-message.js'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 60_000,
    },
  },
})

export function createClient(parentOrigin: string = window.location.origin): Client {
  const transport = iframeTransport(parentOrigin)
  const client = new Client(transport, {
    rendererId: 'default',
    rendererVersion: '0.1.0',
  })
  client.on('board.updated', ({ boardId }) => {
    void queryClient.invalidateQueries({ queryKey: ['board', boardId] })
    void queryClient.invalidateQueries({ queryKey: ['boards'] })
  })
  client.on('key.forward', (data) => {
    window.dispatchEvent(new KeyboardEvent('keydown', { ...data, bubbles: true, cancelable: true }))
  })
  return client
}

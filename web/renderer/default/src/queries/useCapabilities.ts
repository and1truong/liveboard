import { useQuery } from '@tanstack/react-query'
import { useClient } from '../queries.js'

export function useCapabilities(): string[] {
  const client = useClient()
  const q = useQuery({
    queryKey: ['capabilities'],
    queryFn: async () => (await client.ready()).capabilities,
    staleTime: Infinity,
  })
  return q.data ?? []
}

export function useHasCapability(name: string): boolean {
  return useCapabilities().includes(name)
}

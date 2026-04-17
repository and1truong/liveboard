import { useQuery } from '@tanstack/react-query'
import { useClient } from '../queries.js'

export function useExportUrl(): string | null {
  const client = useClient()
  const q = useQuery({
    queryKey: ['exportUrl'],
    queryFn: () => client.getExportUrl(),
    staleTime: Infinity,
  })
  return q.data?.url ?? null
}

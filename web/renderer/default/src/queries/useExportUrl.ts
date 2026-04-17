import { useQuery } from '@tanstack/react-query'
import type { ExportFormat } from '@shared/adapter.js'
import { useClient } from '../queries.js'

export function useExportUrl(format: ExportFormat): string | null {
  const client = useClient()
  const q = useQuery({
    queryKey: ['exportUrl', format],
    queryFn: () => client.getExportUrl(format),
    staleTime: Infinity,
  })
  return q.data?.url ?? null
}

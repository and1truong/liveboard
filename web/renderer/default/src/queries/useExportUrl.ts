import { useQuery } from '@tanstack/react-query'
import type { ExportFormat } from '@shared/adapter.js'
import { useClient } from '../queries.js'

export function useExportUrl(format: ExportFormat, includeAttachments = true): string | null {
  const client = useClient()
  const q = useQuery({
    queryKey: ['exportUrl', format, includeAttachments],
    queryFn: () => client.getExportUrl(format, { includeAttachments }),
    staleTime: Infinity,
  })
  return q.data?.url ?? null
}

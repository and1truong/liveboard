import { useQuery } from '@tanstack/react-query'
import type { BacklinkHit } from '@shared/adapter.js'
import { useClient } from '../queries.js'

export function useBacklinks(cardId: string | undefined): BacklinkHit[] {
  const client = useClient()
  const q = useQuery({
    queryKey: ['backlinks', cardId],
    queryFn: () => client.backlinks(cardId!),
    enabled: !!cardId,
  })
  return q.data ?? []
}

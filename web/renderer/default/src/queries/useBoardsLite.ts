import { useQuery, type UseQueryResult } from '@tanstack/react-query'
import type { BoardListLiteEntry } from '@shared/adapter.js'
import { useClient } from '../queries.js'

export function useBoardsLite(): UseQueryResult<BoardListLiteEntry[]> {
  const client = useClient()
  return useQuery({
    queryKey: ['boards-lite'],
    queryFn: () => client.listBoardsLite(),
  })
}

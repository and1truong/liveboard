import { useQuery, type UseQueryResult } from '@tanstack/react-query'
import type { Client } from '@shared/client.js'
import type { Board } from '@shared/types.js'
import type { BoardSummary, WorkspaceInfo } from '@shared/adapter.js'
import { createContext, useContext, type ReactNode } from 'react'

const ClientContext = createContext<Client | null>(null)

export function ClientProvider({
  client,
  children,
}: {
  client: Client
  children: ReactNode
}): ReactNode {
  return <ClientContext.Provider value={client}>{children}</ClientContext.Provider>
}

export function useClient(): Client {
  const c = useContext(ClientContext)
  if (!c) throw new Error('ClientProvider missing')
  return c
}

export function useBoardList(): UseQueryResult<BoardSummary[]> {
  const client = useClient()
  return useQuery({
    queryKey: ['boards'],
    queryFn: () => client.listBoards(),
  })
}

export function useBoard(boardId: string | null): UseQueryResult<Board> {
  const client = useClient()
  return useQuery({
    queryKey: ['board', boardId],
    queryFn: () => {
      if (!boardId) throw new Error('no board selected')
      return client.getBoard(boardId)
    },
    enabled: boardId !== null,
  })
}

export function useTagColors(boardId: string | null): Record<string, string> {
  return useBoard(boardId).data?.tag_colors ?? {}
}

export function useWorkspaceInfo(): UseQueryResult<WorkspaceInfo> {
  const client = useClient()
  return useQuery({
    queryKey: ['workspace'],
    queryFn: () => client.workspaceInfo(),
  })
}

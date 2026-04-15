import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useClient } from '../queries.js'

export function useBoardListEvents(): void {
  const client = useClient()
  const qc = useQueryClient()
  useEffect(() => {
    const off = client.on('board.list.updated', () => {
      void qc.invalidateQueries({ queryKey: ['boards'] })
    })
    return () => {
      off()
    }
  }, [client, qc])
}

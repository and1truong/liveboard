import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { SearchHit } from '@shared/adapter.js'
import { useClient } from '../queries.js'

function useDebouncedValue<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms)
    return () => clearTimeout(t)
  }, [value, ms])
  return v
}

export function useSearch(query: string): SearchHit[] {
  const client = useClient()
  const debounced = useDebouncedValue(query, 200)
  const q = useQuery({
    queryKey: ['search', debounced],
    queryFn: () => client.search(debounced, 20),
    enabled: debounced.trim().length > 0,
  })
  return q.data ?? []
}

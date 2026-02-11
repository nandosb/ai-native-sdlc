import { useState, useEffect, useCallback } from 'react'
import { fetchRuns, type RunSummary } from '../lib/api'

interface UseRunsResult {
  runs: RunSummary[]
  loading: boolean
  refresh: () => void
}

export function useRuns(): UseRunsResult {
  const [runs, setRuns] = useState<RunSummary[]>([])
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(() => {
    fetchRuns()
      .then((data) => {
        // Sort newest first
        const sorted = [...data].sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )
        setRuns(sorted)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    refresh()
    const interval = setInterval(refresh, 10_000)
    return () => clearInterval(interval)
  }, [refresh])

  return { runs, loading, refresh }
}

import { useState, useEffect, useCallback, useRef } from 'react'
import { fetchExecution, type Execution, type ExecutionMessage } from '../lib/api'
import type { WSEvent } from './useWebSocket'

interface UseExecutionOptions {
  executionId: string | null
  events: WSEvent[]
}

interface UseExecutionResult {
  execution: Execution | null
  messages: ExecutionMessage[]
  loading: boolean
  error: string | null
  refresh: () => void
}

export function useExecution({ executionId, events }: UseExecutionOptions): UseExecutionResult {
  const [execution, setExecution] = useState<Execution | null>(null)
  const [messages, setMessages] = useState<ExecutionMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const prevExecId = useRef<string | null>(null)
  const lastEventLen = useRef(0)

  const load = useCallback(() => {
    if (!executionId) {
      setExecution(null)
      setMessages([])
      return
    }
    setLoading(true)
    setError(null)
    fetchExecution(executionId)
      .then((exec) => {
        setExecution(exec)
        setMessages(exec.messages || [])
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [executionId])

  // Reset state when executionId changes
  useEffect(() => {
    if (executionId !== prevExecId.current) {
      prevExecId.current = executionId
      lastEventLen.current = events.length // skip old events
      setExecution(null)
      setMessages([])
      load()
    }
  }, [executionId, events.length, load])

  // Poll — 2s while running, 5s otherwise
  useEffect(() => {
    if (!executionId) return
    const isActive = execution?.status === 'running' || execution?.status === 'waiting_input'
    const interval = setInterval(load, isActive ? 2000 : 5000)
    return () => clearInterval(interval)
  }, [executionId, execution?.status, load])

  // WS events: only trigger immediate re-fetch on status changes for this execution
  useEffect(() => {
    if (!executionId || events.length <= lastEventLen.current) return

    const newEvents = events.slice(lastEventLen.current)
    lastEventLen.current = events.length

    let shouldRefetch = false
    for (const evt of newEvents) {
      const data = evt.data || {}
      if (data.execution_id !== executionId) continue

      // Status change events → immediate refetch to get full server state
      if (
        evt.type === 'execution.completed' ||
        evt.type === 'execution.failed' ||
        evt.type === 'execution.waiting_input' ||
        evt.type === 'execution.cancelled'
      ) {
        shouldRefetch = true
        break
      }
    }

    if (shouldRefetch) {
      load()
    }
  }, [events, executionId, load])

  return { execution, messages, loading, error, refresh: load }
}

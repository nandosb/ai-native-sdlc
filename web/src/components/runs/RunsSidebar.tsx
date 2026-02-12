import { useState, useEffect, useCallback } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  CheckCircle2,
  AlertCircle,
  XCircle,
  MessageSquare,
  Circle,
} from 'lucide-react'
import {
  fetchExecutions,
  type RunSummary,
  type Execution,
  type ExecutionStatus,
} from '../../lib/api'
import { phaseLabel } from '../../lib/phases'

interface Props {
  runs: RunSummary[]
  activeRunId: string | null
  selectedExecutionId: string | null
  onSelectExecution: (id: string) => void
  refreshKey?: number
}

export function RunsSidebar({ runs, activeRunId, selectedExecutionId, onSelectExecution, refreshKey }: Props) {
  // Track which runs are expanded
  const [expandedRuns, setExpandedRuns] = useState<Set<string>>(new Set())
  // Cache executions per run
  const [executionsByRun, setExecutionsByRun] = useState<Record<string, Execution[]>>({})
  const [loadingRuns, setLoadingRuns] = useState<Set<string>>(new Set())

  // Auto-expand the active run
  useEffect(() => {
    if (activeRunId) {
      setExpandedRuns((prev) => new Set(prev).add(activeRunId))
    }
  }, [activeRunId])

  const toggleRun = useCallback((runId: string) => {
    setExpandedRuns((prev) => {
      const next = new Set(prev)
      if (next.has(runId)) {
        next.delete(runId)
      } else {
        next.add(runId)
      }
      return next
    })
  }, [])

  // Invalidate execution cache when refreshKey changes (e.g., after creating a new execution)
  useEffect(() => {
    if (refreshKey !== undefined && refreshKey > 0) {
      setExecutionsByRun({})
    }
  }, [refreshKey])

  // Fetch executions when a run is expanded
  useEffect(() => {
    for (const runId of expandedRuns) {
      if (executionsByRun[runId] || loadingRuns.has(runId)) continue
      setLoadingRuns((prev) => new Set(prev).add(runId))
      fetchExecutions(runId)
        .then((execs) => {
          setExecutionsByRun((prev) => ({ ...prev, [runId]: execs }))
        })
        .catch(() => {})
        .finally(() => {
          setLoadingRuns((prev) => {
            const next = new Set(prev)
            next.delete(runId)
            return next
          })
        })
    }
  }, [expandedRuns, executionsByRun, loadingRuns])

  // Poll executions for expanded active run
  useEffect(() => {
    if (!activeRunId || !expandedRuns.has(activeRunId)) return
    const interval = setInterval(() => {
      fetchExecutions(activeRunId)
        .then((execs) => {
          setExecutionsByRun((prev) => ({ ...prev, [activeRunId]: execs }))
        })
        .catch(() => {})
    }, 5000)
    return () => clearInterval(interval)
  }, [activeRunId, expandedRuns])

  return (
    <div className="w-72 flex-shrink-0 bg-gray-900 border border-gray-800 rounded-lg flex flex-col h-full">
      <div className="p-3 border-b border-gray-800">
        <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider">Runs</h3>
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {runs.length === 0 && (
          <div className="text-center text-gray-500 text-xs py-8">No runs yet</div>
        )}

        {runs.map((run) => {
          const isActive = run.id === activeRunId
          const isExpanded = expandedRuns.has(run.id)
          const execs = executionsByRun[run.id] || []
          const isLoading = loadingRuns.has(run.id)

          // Group executions: parents + children
          const parents = execs.filter((e) => !e.parent_id)
          const childrenMap = new Map<string, Execution[]>()
          for (const e of execs) {
            if (e.parent_id) {
              const arr = childrenMap.get(e.parent_id) || []
              arr.push(e)
              childrenMap.set(e.parent_id, arr)
            }
          }
          parents.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())

          return (
            <div key={run.id}>
              {/* Run header */}
              <button
                onClick={() => toggleRun(run.id)}
                className={`w-full text-left px-2 py-2 rounded-md text-sm flex items-center gap-2 transition-colors ${
                  isActive
                    ? 'bg-gray-800 text-white'
                    : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
                }`}
              >
                {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                <RunStatusDot status={run.phase_status} />
                <span className="truncate font-mono text-xs">
                  {run.id.slice(0, 12)}
                </span>
                {isActive && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 ml-auto flex-shrink-0">
                    active
                  </span>
                )}
                {!isActive && (
                  <span className="text-[10px] text-gray-600 ml-auto flex-shrink-0">
                    {run.issue_count} issues
                  </span>
                )}
              </button>

              {/* Executions under this run */}
              {isExpanded && (
                <div className="ml-4 mt-1 space-y-0.5">
                  {isLoading && (
                    <div className="flex items-center gap-2 text-xs text-gray-500 py-1 pl-2">
                      <Loader2 size={12} className="animate-spin" />
                      Loading...
                    </div>
                  )}
                  {!isLoading && parents.length === 0 && (
                    <div className="text-xs text-gray-600 py-1 pl-2">No executions</div>
                  )}
                  {parents.map((exec) => {
                    const children = childrenMap.get(exec.id) || []
                    return (
                      <div key={exec.id}>
                        <ExecutionItem
                          execution={exec}
                          isSelected={selectedExecutionId === exec.id}
                          onSelect={onSelectExecution}
                          isChild={false}
                        />
                        {children.map((child) => (
                          <ExecutionItem
                            key={child.id}
                            execution={child}
                            isSelected={selectedExecutionId === child.id}
                            onSelect={onSelectExecution}
                            isChild={true}
                          />
                        ))}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function RunStatusDot({ status }: { status: string }) {
  const colors: Record<string, string> = {
    running: 'text-blue-400',
    completed: 'text-green-400',
    gate: 'text-yellow-400',
    failed: 'text-red-400',
  }
  if (status === 'running') {
    return <Loader2 size={12} className="text-blue-400 animate-spin flex-shrink-0" />
  }
  return <Circle size={8} className={`${colors[status] || 'text-gray-600'} fill-current flex-shrink-0`} />
}

function ExecutionItem({
  execution,
  isSelected,
  onSelect,
  isChild,
}: {
  execution: Execution
  isSelected: boolean
  onSelect: (id: string) => void
  isChild: boolean
}) {
  const StatusIcon = statusIcon(execution.status)
  const label =
    execution.type === 'issue' && execution.issue_id
      ? execution.issue_id
      : execution.phase

  return (
    <button
      onClick={() => onSelect(execution.id)}
      className={`w-full text-left px-2 py-1.5 rounded-md text-xs flex items-center gap-2 transition-colors ${
        isChild ? 'ml-3' : ''
      } ${
        isSelected
          ? 'bg-gray-700 text-white'
          : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
      }`}
    >
      <StatusIcon size={12} className={statusColor(execution.status)} />
      <span className="truncate">{execution.type === 'issue' ? label : phaseLabel(label)}</span>
      <span className="text-[10px] text-gray-600 ml-auto flex-shrink-0">
        {execution.type === 'issue' ? 'issue' : 'phase'}
      </span>
    </button>
  )
}

function statusIcon(status: ExecutionStatus) {
  switch (status) {
    case 'running': return Loader2
    case 'waiting_input': return MessageSquare
    case 'completed': return CheckCircle2
    case 'failed': return AlertCircle
    case 'cancelled': return XCircle
    default: return Loader2
  }
}

function statusColor(status: ExecutionStatus): string {
  switch (status) {
    case 'running': return 'text-blue-400 animate-spin'
    case 'waiting_input': return 'text-yellow-400'
    case 'completed': return 'text-green-400'
    case 'failed': return 'text-red-400'
    case 'cancelled': return 'text-gray-500'
    default: return 'text-gray-400'
  }
}

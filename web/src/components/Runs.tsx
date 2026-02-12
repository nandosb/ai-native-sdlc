import { useState, useEffect } from 'react'
import { Play, Plus, Loader2 } from 'lucide-react'
import { createExecution, type StatusResponse } from '../lib/api'
import type { WSEvent } from '../hooks/useWebSocket'
import { useRuns } from '../hooks/useRuns'
import { RunsSidebar } from './runs/RunsSidebar'
import { ExecutionPanel } from './runs/ExecutionPanel'
import { NewExecutionForm } from './runs/NewExecutionForm'

interface Props {
  status: StatusResponse | null
  events: WSEvent[]
}

export function Runs({ status, events }: Props) {
  const { runs, loading: runsLoading, refresh: refreshRuns } = useRuns()
  const [selectedId, setSelectedId] = useState<string | null>(() => {
    return localStorage.getItem('sdlc:selectedExecutionId')
  })
  const [showNewForm, setShowNewForm] = useState(false)
  const [execRefreshKey, setExecRefreshKey] = useState(0)

  // Persist selected execution ID to localStorage
  useEffect(() => {
    if (selectedId) {
      localStorage.setItem('sdlc:selectedExecutionId', selectedId)
    } else {
      localStorage.removeItem('sdlc:selectedExecutionId')
    }
  }, [selectedId])
  const handleNewExecution = async (phase: string, params: Record<string, string>) => {
    try {
      const result = await createExecution({
        run_id: status?.run_id,
        type: 'phase',
        phase,
        params,
      })
      setShowNewForm(false)
      setSelectedId(result.id)
      refreshRuns()
      setExecRefreshKey((k) => k + 1)
    } catch (err) {
      console.error('Failed to create execution:', err)
    }
  }

  return (
    <>
      <div className="flex h-[calc(100vh-8rem)] gap-4">
        {/* Left: sidebar with action buttons + runs list */}
        <div className="flex flex-col gap-3 w-72 flex-shrink-0">
          {/* Action buttons */}
          <div className="flex gap-2">
            <button
              onClick={() => setShowNewForm(true)}
              className="flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 transition-colors"
            >
              <Plus size={16} />
              New Phase
            </button>
          </div>

          {/* Runs sidebar */}
          <div className="flex-1 min-h-0">
            <RunsSidebar
              runs={runs}
              activeRunId={status?.run_id ?? null}
              selectedExecutionId={selectedId}
              onSelectExecution={setSelectedId}
              refreshKey={execRefreshKey}
            />
          </div>
        </div>

        {/* Right: execution panel */}
        <div className="flex-1 min-w-0">
          {selectedId ? (
            <ExecutionPanel executionId={selectedId} events={events} />
          ) : (
            <div className="bg-gray-900 border border-gray-800 rounded-lg flex flex-col items-center justify-center h-full text-gray-500">
              {runsLoading ? (
                <Loader2 size={32} className="mb-3 text-gray-600 animate-spin" />
              ) : (
                <>
                  <Play size={32} className="mb-3 text-gray-600" />
                  <p className="text-sm">Select an execution or start a new one</p>
                  <div className="flex gap-3 mt-4">
                    <button
                      onClick={() => setShowNewForm(true)}
                      className="px-4 py-2 rounded-lg text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 transition-colors"
                    >
                      New Phase Execution
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </div>

      {showNewForm && (
        <NewExecutionForm
          status={status}
          onClose={() => setShowNewForm(false)}
          onSubmit={handleNewExecution}
        />
      )}
    </>
  )
}

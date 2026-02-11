import { useState, useEffect, useRef } from 'react'
import { fetchRuns, selectRun, type RunSummary } from '../lib/api'
import { ChevronDown } from 'lucide-react'

interface RunSelectorProps {
  currentRunId: string
  onRunSelected: () => void
}

export function RunSelector({ currentRunId, onRunSelected }: RunSelectorProps) {
  const [runs, setRuns] = useState<RunSummary[]>([])
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchRuns().then(setRuns).catch(() => {})
  }, [currentRunId])

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  async function handleSelect(id: string) {
    if (id === currentRunId) {
      setOpen(false)
      return
    }
    try {
      await selectRun(id)
      setOpen(false)
      onRunSelected()
    } catch {
      // ignore
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="badge badge-blue flex items-center gap-1 cursor-pointer hover:opacity-80"
      >
        {currentRunId}
        {runs.length > 1 && <ChevronDown size={12} />}
      </button>

      {open && runs.length > 0 && (
        <div className="absolute top-full mt-1 left-0 bg-gray-900 border border-gray-700 rounded-md shadow-lg z-50 min-w-[240px] py-1">
          {runs.map((run) => (
            <button
              key={run.id}
              onClick={() => handleSelect(run.id)}
              className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-800 flex items-center justify-between ${
                run.id === currentRunId ? 'text-blue-400' : 'text-gray-300'
              }`}
            >
              <span className="font-mono">{run.id}</span>
              <span className="text-xs text-gray-500">
                {run.phase} &middot; {run.issue_count} issues
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

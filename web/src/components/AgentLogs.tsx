import { useState, useRef, useEffect } from 'react'
import type { WSEvent } from '../hooks/useWebSocket'
import { Pause, Play, Filter } from 'lucide-react'

interface Props {
  events: WSEvent[]
}

const agentTypes = ['all', 'coder', 'quality-reviewer', 'solution-designer', 'task-decomposer', 'doc-generator', 'feedback-writer']

export function AgentLogs({ events }: Props) {
  const [paused, setPaused] = useState(false)
  const [filter, setFilter] = useState('all')
  const [issueFilter, setIssueFilter] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  const filteredEvents = events.filter((evt) => {
    // Only show agent output and relevant events
    if (evt.type !== 'agent.output' && evt.type !== 'agent.spawned' && evt.type !== 'agent.completed') {
      return false
    }
    if (filter !== 'all' && evt.data?.agent !== filter) return false
    if (issueFilter && evt.data?.issue_id !== issueFilter) return false
    return true
  })

  const displayEvents = paused ? filteredEvents : filteredEvents.slice(-200)

  useEffect(() => {
    if (!paused && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [displayEvents.length, paused])

  // Collect unique issue IDs from events
  const issueIds = [...new Set(events.filter((e) => e.data?.issue_id).map((e) => String(e.data.issue_id)))]

  return (
    <div className="space-y-4">
      {/* Controls */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Agent Logs</h2>
        <div className="flex items-center gap-3">
          {/* Agent filter */}
          <div className="flex items-center gap-1.5">
            <Filter size={14} className="text-gray-500" />
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="bg-gray-800 border border-gray-700 rounded text-xs text-gray-300 px-2 py-1"
            >
              {agentTypes.map((t) => (
                <option key={t} value={t}>{t === 'all' ? 'All Agents' : t}</option>
              ))}
            </select>
          </div>

          {/* Issue filter */}
          {issueIds.length > 0 && (
            <select
              value={issueFilter}
              onChange={(e) => setIssueFilter(e.target.value)}
              className="bg-gray-800 border border-gray-700 rounded text-xs text-gray-300 px-2 py-1"
            >
              <option value="">All Issues</option>
              {issueIds.map((id) => (
                <option key={id} value={id}>{id}</option>
              ))}
            </select>
          )}

          {/* Pause/Resume */}
          <button
            onClick={() => setPaused(!paused)}
            className={`flex items-center gap-1 px-2 py-1 rounded text-xs ${
              paused ? 'bg-yellow-500/20 text-yellow-400' : 'bg-gray-800 text-gray-400'
            }`}
          >
            {paused ? <Play size={12} /> : <Pause size={12} />}
            {paused ? 'Resume' : 'Pause'}
          </button>
        </div>
      </div>

      {/* Log Output */}
      <div className="card font-mono text-xs bg-gray-950 border-gray-800 max-h-[600px] overflow-y-auto p-0">
        {displayEvents.length === 0 ? (
          <div className="p-6 text-center text-gray-600">
            No agent output yet. Start a run to see logs.
          </div>
        ) : (
          <div className="p-3 space-y-0.5">
            {displayEvents.map((evt, i) => (
              <LogLine key={i} event={evt} />
            ))}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      <div className="text-xs text-gray-600 text-right">
        {filteredEvents.length} events
        {paused && ' (paused)'}
      </div>
    </div>
  )
}

function LogLine({ event }: { event: WSEvent }) {
  const time = new Date(event.timestamp).toLocaleTimeString()
  const agent = String(event.data?.agent || '')
  const issueId = String(event.data?.issue_id || '')

  if (event.type === 'agent.spawned') {
    return (
      <div className="text-green-400 py-0.5">
        <span className="text-gray-600">[{time}]</span>{' '}
        <span className="text-green-500">START</span>{' '}
        {agent}
        {issueId && <span className="text-gray-500"> ({issueId})</span>}
      </div>
    )
  }

  if (event.type === 'agent.completed') {
    return (
      <div className="text-blue-400 py-0.5">
        <span className="text-gray-600">[{time}]</span>{' '}
        <span className="text-blue-500">DONE</span>{' '}
        {agent}
        {issueId && <span className="text-gray-500"> ({issueId})</span>}
      </div>
    )
  }

  // agent.output
  const raw = String(event.data?.raw || '')
  return (
    <div className="text-gray-400 py-0.5 whitespace-pre-wrap break-all">
      <span className="text-gray-600">[{time}]</span>{' '}
      {issueId && <span className="text-gray-600">[{issueId}]</span>}{' '}
      {raw.slice(0, 500)}
    </div>
  )
}

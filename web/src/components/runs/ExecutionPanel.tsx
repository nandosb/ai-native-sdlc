import { CheckCircle2, XCircle } from 'lucide-react'
import { useExecution } from '../../hooks/useExecution'
import { sendExecutionMessage, approveExecution, cancelExecution } from '../../lib/api'
import { phaseLabel } from '../../lib/phases'
import { ChatInterface } from './ChatInterface'
import type { WSEvent } from '../../hooks/useWebSocket'

interface Props {
  executionId: string
  events: WSEvent[]
}

export function ExecutionPanel({ executionId, events }: Props) {
  const { execution, messages, loading, error } = useExecution({ executionId, events })

  if (loading && !execution) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500 text-sm">
        Loading execution...
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full text-red-400 text-sm">
        Error: {error}
      </div>
    )
  }

  if (!execution) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500 text-sm">
        Execution not found
      </div>
    )
  }

  const handleSend = async (content: string) => {
    try {
      await sendExecutionMessage(executionId, content)
    } catch (err) {
      console.error('Failed to send message:', err)
    }
  }

  const handleApprove = async () => {
    try {
      await approveExecution(executionId)
    } catch (err) {
      console.error('Failed to approve:', err)
    }
  }

  const handleCancel = async () => {
    try {
      await cancelExecution(executionId)
    } catch (err) {
      console.error('Failed to cancel:', err)
    }
  }

  const elapsed = execution.created_at
    ? Math.round((Date.now() - new Date(execution.created_at).getTime()) / 1000)
    : 0

  const formatElapsed = (s: number) => {
    if (s < 60) return `${s}s`
    if (s < 3600) return `${Math.floor(s / 60)}m ${s % 60}s`
    return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`
  }

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg flex flex-col h-full">
      {/* Header */}
      <div className="px-4 py-3 border-b border-gray-800 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h3 className="text-sm font-semibold text-white">{phaseLabel(execution.phase)}</h3>
          <span className={`text-xs px-2 py-0.5 rounded-full ${typeBadgeClass(execution.type)}`}>
            {execution.type}
          </span>
          <span className={`text-xs px-2 py-0.5 rounded-full ${statusBadgeClass(execution.status)}`}>
            {execution.status.replace('_', ' ')}
          </span>
        </div>
        <div className="flex items-center gap-4 text-xs text-gray-500">
          <span>{formatElapsed(elapsed)}</span>
          {(execution.tokens_in > 0 || execution.tokens_out > 0) && (
            <span>
              {execution.tokens_in.toLocaleString()} in / {execution.tokens_out.toLocaleString()} out
            </span>
          )}
        </div>
      </div>

      {/* Chat area */}
      <div className="flex-1 min-h-0">
        <ChatInterface messages={messages} status={execution.status} onSend={handleSend} />
      </div>

      {/* Action bar */}
      {(execution.status === 'waiting_input' || execution.status === 'running') && (
        <div className="px-4 py-3 border-t border-gray-800 flex items-center justify-end gap-2">
          <button
            onClick={handleCancel}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-red-400 hover:bg-red-500/10 transition-colors"
          >
            <XCircle size={14} />
            Cancel
          </button>
          {execution.status === 'waiting_input' && (
            <button
              onClick={handleApprove}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-green-400 hover:bg-green-500/10 transition-colors"
            >
              <CheckCircle2 size={14} />
              Approve
            </button>
          )}
        </div>
      )}
    </div>
  )
}

function typeBadgeClass(type: string): string {
  return type === 'issue'
    ? 'bg-purple-500/20 text-purple-400'
    : 'bg-blue-500/20 text-blue-400'
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'running':
      return 'bg-blue-500/20 text-blue-400'
    case 'waiting_input':
      return 'bg-yellow-500/20 text-yellow-400'
    case 'completed':
      return 'bg-green-500/20 text-green-400'
    case 'failed':
      return 'bg-red-500/20 text-red-400'
    case 'cancelled':
      return 'bg-gray-500/20 text-gray-400'
    default:
      return 'bg-gray-500/20 text-gray-400'
  }
}

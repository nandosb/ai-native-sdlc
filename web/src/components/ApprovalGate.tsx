import { useState } from 'react'
import { submitApproval } from '../lib/api'
import { CheckCircle2, XCircle, FileText, X } from 'lucide-react'

interface Props {
  phase: string
  artifacts: Record<string, string>
  phaseArtifactMap?: Record<string, string>
  onClose: () => void
  onApproved: () => void
}

export function ApprovalGate({ phase, artifacts, phaseArtifactMap, onClose, onApproved }: Props) {
  const [comment, setComment] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleAction = async (action: 'approve' | 'reject') => {
    setSubmitting(true)
    try {
      await submitApproval(action, comment)
      if (action === 'approve') {
        onApproved()
      } else {
        onClose()
      }
    } catch (err) {
      console.error('Approval failed:', err)
    } finally {
      setSubmitting(false)
    }
  }

  const artifactKey = phaseArtifactMap?.[phase] ?? (phase === 'design' ? 'scoping_doc' : 'pert')
  const artifactPath = artifacts[artifactKey]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-gray-900 border border-gray-800 rounded-xl shadow-2xl w-full max-w-lg mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800">
          <div>
            <h2 className="text-lg font-semibold text-white">Approval Gate</h2>
            <p className="text-sm text-gray-400 mt-0.5">
              Phase <span className="capitalize font-medium text-gray-300">{phase}</span> requires approval
            </p>
          </div>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-300">
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="px-6 py-4 space-y-4">
          {/* Artifact reference */}
          {artifactPath && (
            <div className="flex items-start gap-3 p-3 bg-gray-800/50 rounded-lg">
              <FileText size={18} className="text-blue-400 mt-0.5" />
              <div>
                <div className="text-sm font-medium text-gray-300">Generated Artifact</div>
                <div className="text-xs text-gray-500 font-mono mt-0.5">{artifactPath}</div>
                <p className="text-xs text-gray-400 mt-1">
                  Review this artifact before approving. The file has been written to disk.
                </p>
              </div>
            </div>
          )}

          {/* Phase-specific guidance */}
          <div className="text-sm text-gray-400">
            {phase === 'design' ? (
              <p>
                The solution designer has produced a scoping document. Review it for completeness,
                technical accuracy, and alignment with the PRD before continuing to task decomposition.
              </p>
            ) : (
              <p>
                The task decomposer has produced a PERT diagram with tasks and dependencies.
                Review the task breakdown, estimates, and dependency graph before creating Linear issues.
              </p>
            )}
          </div>

          {/* Comment */}
          <div>
            <label className="text-xs text-gray-500 block mb-1.5">Comments (optional)</label>
            <textarea
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="Add feedback or notes..."
              rows={3}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:ring-1 focus:ring-brand-500 focus:border-brand-500"
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-800">
          <button
            onClick={() => handleAction('reject')}
            disabled={submitting}
            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-red-400 bg-red-500/10 hover:bg-red-500/20 transition-colors disabled:opacity-50"
          >
            <XCircle size={16} />
            Request Changes
          </button>
          <button
            onClick={() => handleAction('approve')}
            disabled={submitting}
            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-green-400 bg-green-500/10 hover:bg-green-500/20 transition-colors disabled:opacity-50"
          >
            <CheckCircle2 size={16} />
            Approve
          </button>
        </div>
      </div>
    </div>
  )
}

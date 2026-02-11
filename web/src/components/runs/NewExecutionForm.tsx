import { useState, useEffect } from 'react'
import { X, Play } from 'lucide-react'
import { fetchIssues, type StatusResponse } from '../../lib/api'
import { phaseLabel } from '../../lib/phases'

interface Props {
  status: StatusResponse | null
  onClose: () => void
  onSubmit: (phase: string, params: Record<string, string>) => void
}

const PHASES = ['bootstrap', 'design', 'planning', 'tracking', 'executing']

export function NewExecutionForm({ status, onClose, onSubmit }: Props) {
  const [phase, setPhase] = useState('')
  const [params, setParams] = useState<Record<string, string>>({})
  const [issueOptions, setIssueOptions] = useState<{ id: string; title: string }[]>([])
  const [loadingIssues, setLoadingIssues] = useState(false)

  const set = (key: string, value: string) =>
    setParams((prev) => ({ ...prev, [key]: value }))

  // Reset params and prefill only when phase selection changes
  useEffect(() => {
    setParams(prefill(phase, status))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phase])

  // Load issues when executing phase is selected
  useEffect(() => {
    if (phase !== 'executing' || issueOptions.length > 0) return
    setLoadingIssues(true)
    fetchIssues()
      .then((res) => {
        const list = Object.values(res.issues || {}).map((i) => ({
          id: i.id,
          title: i.title,
        }))
        setIssueOptions(list)
      })
      .catch(() => {})
      .finally(() => setLoadingIssues(false))
  }, [phase, issueOptions.length])

  const handleSubmit = () => {
    if (!phase) return
    const cleaned: Record<string, string> = {}
    for (const [k, v] of Object.entries(params)) {
      if (v.trim()) cleaned[k] = v.trim()
    }
    onSubmit(phase, cleaned)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-gray-900 border border-gray-800 rounded-xl shadow-2xl w-full max-w-lg mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800">
          <div>
            <h2 className="text-lg font-semibold text-white">New Phase Execution</h2>
            <p className="text-sm text-gray-400 mt-0.5">Run a single SDLC phase interactively</p>
          </div>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-300">
            <X size={20} />
          </button>
        </div>

        {/* Fields */}
        <div className="px-6 py-4 space-y-4">
          {/* Phase selector */}
          <div>
            <label className="text-xs text-gray-500 block mb-1.5">Phase</label>
            <select
              value={phase}
              onChange={(e) => setPhase(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value="">Select a phase...</option>
              {PHASES.map((p) => (
                <option key={p} value={p}>
                  {phaseLabel(p)}
                </option>
              ))}
            </select>
          </div>

          {/* Phase-specific fields */}
          {phase === 'bootstrap' && (
            <Field
              label="Repo (optional)"
              hint="Leave empty to bootstrap all repos"
              value={params.repo || ''}
              onChange={(v) => set('repo', v)}
              options={status?.repos?.map((r) => r.name)}
            />
          )}

          {phase === 'design' && (
            <>
              <Field
                label="PRD URL / Path"
                value={params.prd || ''}
                onChange={(v) => set('prd', v)}
              />
              <Field
                label="Output path"
                value={params.output || ''}
                onChange={(v) => set('output', v)}
              />
            </>
          )}

          {phase === 'planning' && (
            <>
              <Field
                label="Scoping document path"
                value={params.scoping_doc || ''}
                onChange={(v) => set('scoping_doc', v)}
              />
              <Field
                label="Output path"
                value={params.output || ''}
                onChange={(v) => set('output', v)}
              />
            </>
          )}

          {phase === 'tracking' && (
            <>
              <Field
                label="PERT document path"
                value={params.pert || ''}
                onChange={(v) => set('pert', v)}
              />
              <Field
                label="Linear team"
                value={params.team || ''}
                onChange={(v) => set('team', v)}
              />
            </>
          )}

          {phase === 'executing' && (
            <Field
              label="Issue ID (optional)"
              hint="Leave empty to execute all ready issues"
              value={params.issue || ''}
              onChange={(v) => set('issue', v)}
              options={issueOptions.map((i) => `${i.id}: ${i.title}`)}
              loading={loadingIssues}
            />
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-800">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg text-sm font-medium text-gray-400 hover:text-gray-200 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={!phase}
            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            <Play size={16} />
            Run Phase
          </button>
        </div>
      </div>
    </div>
  )
}

function Field({
  label,
  hint,
  value,
  onChange,
  options,
  loading,
}: {
  label: string
  hint?: string
  value: string
  onChange: (v: string) => void
  options?: string[]
  loading?: boolean
}) {
  if (options && options.length > 0) {
    return (
      <div>
        <label className="text-xs text-gray-500 block mb-1.5">{label}</label>
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          <option value="">—</option>
          {options.map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
        {hint && <p className="text-xs text-gray-600 mt-1">{hint}</p>}
      </div>
    )
  }

  return (
    <div>
      <label className="text-xs text-gray-500 block mb-1.5">{label}</label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={hint}
        disabled={loading}
        className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
      />
      {hint && !options && <p className="text-xs text-gray-600 mt-1">{hint}</p>}
    </div>
  )
}

function prefill(phase: string, status: StatusResponse | null): Record<string, string> {
  if (!status) return {}
  switch (phase) {
    case 'design':
      return { prd: status.prd_url || '', output: 'scoping-doc.md' }
    case 'planning':
      return { scoping_doc: status.artifacts?.scoping_doc || '', output: 'pert.md' }
    case 'tracking':
      return { pert: status.artifacts?.pert || '', team: status.repos?.[0]?.team || '' }
    default:
      return {}
  }
}

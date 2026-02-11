import { useState, useEffect } from 'react'
import { fetchIssues, type IssueState, type IssueStatus } from '../lib/api'
import { ExternalLink, GitBranch, Clock, ChevronRight } from 'lucide-react'

const columns: { id: IssueStatus; label: string; color: string }[] = [
  { id: 'blocked', label: 'Blocked', color: 'border-red-500/50' },
  { id: 'ready', label: 'Ready', color: 'border-blue-500/50' },
  { id: 'implementing', label: 'Implementing', color: 'border-purple-500/50' },
  { id: 'reviewing', label: 'Reviewing', color: 'border-yellow-500/50' },
  { id: 'awaiting_human', label: 'Awaiting Human', color: 'border-orange-500/50' },
  { id: 'done', label: 'Done', color: 'border-green-500/50' },
]

export function IssueBoard() {
  const [issues, setIssues] = useState<Record<string, IssueState>>({})
  const [selected, setSelected] = useState<IssueState | null>(null)

  useEffect(() => {
    fetchIssues().then((data) => setIssues(data.issues || {})).catch(() => {})
    const interval = setInterval(() => {
      fetchIssues().then((data) => setIssues(data.issues || {})).catch(() => {})
    }, 3000)
    return () => clearInterval(interval)
  }, [])

  const issuesByStatus = (status: IssueStatus) =>
    Object.values(issues).filter((i) => i.status === status)

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold text-white">Issue Board</h2>

      {Object.keys(issues).length === 0 ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">No issues tracked yet. Run the tracking phase to create issues.</p>
        </div>
      ) : (
        <div className="flex gap-3 overflow-x-auto pb-4">
          {columns.map((col) => {
            const colIssues = issuesByStatus(col.id)
            return (
              <div key={col.id} className="flex-shrink-0 w-56">
                <div className={`border-t-2 ${col.color} rounded-t-sm`}>
                  <div className="flex items-center justify-between px-2 py-2">
                    <span className="text-xs font-medium text-gray-400 uppercase tracking-wide">
                      {col.label}
                    </span>
                    <span className="text-xs text-gray-600">{colIssues.length}</span>
                  </div>
                </div>
                <div className="space-y-2 min-h-[200px]">
                  {colIssues.map((issue) => (
                    <button
                      key={issue.id}
                      onClick={() => setSelected(issue)}
                      className="card w-full text-left hover:border-gray-700 transition-colors"
                    >
                      <div className="text-xs font-mono text-gray-500 mb-1">{issue.id}</div>
                      <div className="text-sm text-gray-200 line-clamp-2">{issue.title}</div>
                      <div className="flex items-center gap-2 mt-2">
                        {issue.repo && (
                          <span className="badge badge-gray text-[10px]">{issue.repo}</span>
                        )}
                        {issue.iterations > 0 && (
                          <span className="badge badge-purple text-[10px]">
                            iter {issue.iterations}
                          </span>
                        )}
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Issue Detail Panel */}
      {selected && (
        <IssueDetail issue={selected} onClose={() => setSelected(null)} />
      )}
    </div>
  )
}

function IssueDetail({ issue, onClose }: { issue: IssueState; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative w-96 bg-gray-900 border-l border-gray-800 p-6 overflow-y-auto">
        <button onClick={onClose} className="absolute top-4 right-4 text-gray-500 hover:text-gray-300">
          <ChevronRight size={20} />
        </button>

        <div className="space-y-4">
          <div>
            <span className="text-xs font-mono text-gray-500">{issue.id}</span>
            {issue.linear_id && (
              <span className="ml-2 text-xs text-gray-500">{issue.linear_id}</span>
            )}
          </div>

          <h3 className="text-lg font-medium text-white">{issue.title}</h3>

          <div className="space-y-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-gray-500">Status</span>
              <span className={`badge ${statusBadge(issue.status)}`}>{issue.status}</span>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-gray-500">Repo</span>
              <span className="text-gray-300">{issue.repo || '-'}</span>
            </div>

            {issue.branch && (
              <div className="flex items-center justify-between">
                <span className="text-gray-500">Branch</span>
                <span className="text-gray-300 font-mono text-xs flex items-center gap-1">
                  <GitBranch size={12} /> {issue.branch}
                </span>
              </div>
            )}

            {issue.pr_url && (
              <div className="flex items-center justify-between">
                <span className="text-gray-500">PR</span>
                <a
                  href={issue.pr_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300 flex items-center gap-1 text-xs"
                >
                  View PR <ExternalLink size={12} />
                </a>
              </div>
            )}

            <div className="flex items-center justify-between">
              <span className="text-gray-500">Iterations</span>
              <span className="text-gray-300 flex items-center gap-1">
                <Clock size={12} /> {issue.iterations}
              </span>
            </div>

            {issue.depends_on && issue.depends_on.length > 0 && (
              <div>
                <span className="text-gray-500 block mb-1">Depends on</span>
                <div className="flex flex-wrap gap-1">
                  {issue.depends_on.map((dep) => (
                    <span key={dep} className="badge badge-gray text-xs">{dep}</span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function statusBadge(status: IssueStatus) {
  switch (status) {
    case 'blocked': return 'badge-red'
    case 'ready': return 'badge-blue'
    case 'implementing': return 'badge-purple'
    case 'reviewing': return 'badge-yellow'
    case 'awaiting_human': return 'badge-yellow'
    case 'done': return 'badge-green'
    default: return 'badge-gray'
  }
}

import { useState, useEffect } from 'react'
import {
  fetchIntegrationHealth,
  fetchExecutions,
  fetchIssues,
  type StatusResponse,
  type IntegrationHealth,
} from '../lib/api'
import type { WSEvent } from '../hooks/useWebSocket'
import { useRuns } from '../hooks/useRuns'
import { MetricsSummary } from './dashboard/MetricsSummary'
import { phaseLabel } from '../lib/phases'
import {
  CheckCircle2,
  Loader2,
  AlertCircle,
  GitPullRequest,
  Cpu,
  Activity,
  Link2,
  RefreshCw,
  XCircle,
} from 'lucide-react'

interface Props {
  status: StatusResponse | null
  events: WSEvent[]
}

export function Dashboard({ status, events }: Props) {
  const { runs } = useRuns()
  const [integrations, setIntegrations] = useState<IntegrationHealth[]>([])
  const [healthLoading, setHealthLoading] = useState(false)
  const [activeExecutionCount, setActiveExecutionCount] = useState(0)
  const [prCount, setPrCount] = useState(0)

  useEffect(() => {
    loadHealth()
  }, [])

  // Fetch active execution count from API
  useEffect(() => {
    const load = () => {
      fetchExecutions(status?.run_id)
        .then((execs) => {
          setActiveExecutionCount(execs.filter((e) => e.status === 'running').length)
        })
        .catch(() => {})
    }
    load()
    const interval = setInterval(load, 5000)
    return () => clearInterval(interval)
  }, [status?.run_id])

  // Fetch PR count from issues API
  useEffect(() => {
    const load = () => {
      fetchIssues()
        .then((res) => {
          const count = Object.values(res.issues || {}).filter((i) => i.pr_url).length
          setPrCount(count)
        })
        .catch(() => {})
    }
    load()
    const interval = setInterval(load, 10000)
    return () => clearInterval(interval)
  }, [])

  const loadHealth = () => {
    setHealthLoading(true)
    fetchIntegrationHealth()
      .then(setIntegrations)
      .catch(() => {})
      .finally(() => setHealthLoading(false))
  }

  const recentEvents = events.slice(-10).reverse()

  // Count active agents from recent events
  const activeAgents = events.filter(
    (e) => e.type === 'agent.spawned' && !events.some(
      (c) => c.type === 'agent.completed' && c.data?.agent === e.data?.agent && c.timestamp > e.timestamp
    )
  ).length

  // Integrations summary for stat card
  const healthyCount = integrations.filter((i) => i.ok).length
  const totalIntegrations = integrations.length

  return (
    <div className="space-y-6">
      {/* Active Runs Summary */}
      <div className="card">
        <h2 className="text-sm font-medium text-gray-400 mb-3">Active Runs</h2>
        {runs.length === 0 ? (
          <p className="text-gray-500 text-sm">No runs yet</p>
        ) : (
          <div className="space-y-2">
            {runs.map((run) => (
              <div
                key={run.id}
                className={`flex items-center gap-3 px-3 py-2 rounded-md ${
                  run.id === status?.run_id ? 'bg-gray-800 ring-1 ring-gray-700' : 'bg-gray-800/40'
                }`}
              >
                <RunStatusIcon status={run.phase_status} />
                <span className="font-mono text-xs text-gray-300 truncate">{run.id.slice(0, 12)}</span>
                <span className="text-xs text-gray-400">{phaseLabel(run.phase)}</span>
                <span className={`text-[10px] px-1.5 py-0.5 rounded ${runStatusBadgeClass(run.phase_status)}`}>
                  {run.phase_status}
                </span>
                <span className="text-[10px] text-gray-600 ml-auto flex-shrink-0">
                  {run.issue_count} issues
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard
          icon={<Activity size={20} />}
          label="Active Executions"
          value={activeExecutionCount}
          color="blue"
        />
        <StatCard
          icon={<GitPullRequest size={20} />}
          label="PRs Created"
          value={prCount}
          color="green"
        />
        <StatCard
          icon={<Cpu size={20} />}
          label="Active Agents"
          value={activeAgents}
          color="purple"
        />
        <StatCard
          icon={<Link2 size={20} />}
          label="Integrations"
          value={totalIntegrations > 0 ? `${healthyCount}/${totalIntegrations} healthy` : '-'}
          color="yellow"
        />
      </div>

      {/* Run Details + Recent Events + Integrations (2-col grid) */}
      <div className="grid grid-cols-2 gap-4">
        {/* Run Info */}
        <div className="card">
          <h2 className="text-sm font-medium text-gray-400 mb-3">Run Details</h2>
          {status ? (
            <dl className="space-y-2 text-sm">
              <div className="flex justify-between">
                <dt className="text-gray-500">Run ID</dt>
                <dd className="font-mono text-gray-300">{status.run_id}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Phase</dt>
                <dd className="text-gray-300">{phaseLabel(status.phase)}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Status</dt>
                <dd>
                  <span className={`badge ${statusBadge(status.phase_status)}`}>
                    {status.phase_status}
                  </span>
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Repos</dt>
                <dd className="text-gray-300">{status.repos?.length ?? 0}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Updated</dt>
                <dd className="text-gray-300">{new Date(status.updated_at).toLocaleTimeString()}</dd>
              </div>
            </dl>
          ) : (
            <p className="text-gray-500 text-sm">Loading...</p>
          )}
        </div>

        {/* Recent Events */}
        <div className="card">
          <h2 className="text-sm font-medium text-gray-400 mb-3">Recent Events</h2>
          <div className="space-y-1 max-h-60 overflow-y-auto">
            {recentEvents.length === 0 ? (
              <p className="text-gray-500 text-sm">No events yet</p>
            ) : (
              recentEvents.map((evt, i) => (
                <div key={i} className="flex items-center gap-2 text-xs py-1">
                  <span className="text-gray-600 font-mono">
                    {new Date(evt.timestamp).toLocaleTimeString()}
                  </span>
                  <span className={`badge ${eventBadge(evt.type)}`}>
                    {evt.type.split('.')[0]}
                  </span>
                  <span className="text-gray-400 truncate">
                    {eventSummary(evt)}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Integrations detail */}
        <div className="card col-span-2">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-medium text-gray-400">Integrations</h2>
            <button
              onClick={loadHealth}
              disabled={healthLoading}
              className="text-gray-500 hover:text-gray-300 transition-colors"
              title="Refresh"
            >
              <RefreshCw size={14} className={healthLoading ? 'animate-spin' : ''} />
            </button>
          </div>
          <div className="grid grid-cols-2 gap-x-8 gap-y-2">
            {integrations.length === 0 && !healthLoading && (
              <p className="text-gray-500 text-sm">Click refresh to check</p>
            )}
            {integrations.map((integ) => (
              <div key={integ.name} className="flex items-center justify-between py-1.5 border-b border-gray-800 last:border-0">
                <div className="flex items-center gap-2">
                  {integ.ok ? (
                    <CheckCircle2 size={14} className="text-green-400" />
                  ) : (
                    <XCircle size={14} className="text-red-400" />
                  )}
                  <span className="text-sm text-gray-300 capitalize">{integ.name}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  {integ.mode && (
                    <span className={`text-[10px] px-1.5 py-0.5 rounded ${
                      integ.mode === 'api' ? 'bg-blue-500/20 text-blue-400' : 'bg-purple-500/20 text-purple-400'
                    }`}>
                      {integ.mode.toUpperCase()}
                    </span>
                  )}
                  <span className={`text-xs ${integ.ok ? 'text-gray-500' : 'text-red-400'}`}>
                    {integ.detail}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Collapsible Metrics */}
      <MetricsSummary />
    </div>
  )
}

function RunStatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'running':
      return <Loader2 size={14} className="text-blue-400 animate-spin flex-shrink-0" />
    case 'completed':
      return <CheckCircle2 size={14} className="text-green-400 flex-shrink-0" />
    case 'failed':
      return <AlertCircle size={14} className="text-red-400 flex-shrink-0" />
    case 'gate':
      return <AlertCircle size={14} className="text-yellow-400 flex-shrink-0" />
    default:
      return <CheckCircle2 size={14} className="text-gray-600 flex-shrink-0" />
  }
}

function runStatusBadgeClass(status: string): string {
  switch (status) {
    case 'running': return 'bg-blue-500/20 text-blue-400'
    case 'completed': return 'bg-green-500/20 text-green-400'
    case 'gate': return 'bg-yellow-500/20 text-yellow-400'
    case 'failed': return 'bg-red-500/20 text-red-400'
    default: return 'bg-gray-500/20 text-gray-400'
  }
}

function StatCard({ icon, label, value, color }: {
  icon: React.ReactNode
  label: string
  value: string | number
  color: string
}) {
  const colors: Record<string, string> = {
    blue: 'text-blue-400',
    green: 'text-green-400',
    purple: 'text-purple-400',
    yellow: 'text-yellow-400',
  }
  return (
    <div className="card flex items-center gap-3">
      <div className={colors[color] || 'text-gray-400'}>{icon}</div>
      <div>
        <div className="text-2xl font-semibold text-white">{value}</div>
        <div className="text-xs text-gray-500">{label}</div>
      </div>
    </div>
  )
}

function statusBadge(status: string) {
  switch (status) {
    case 'running': return 'badge-blue'
    case 'completed': return 'badge-green'
    case 'gate': return 'badge-yellow'
    case 'failed': return 'badge-red'
    default: return 'badge-gray'
  }
}

function eventBadge(type: string) {
  if (type.startsWith('phase')) return 'badge-blue'
  if (type.startsWith('agent')) return 'badge-purple'
  if (type.startsWith('issue')) return 'badge-green'
  if (type.startsWith('error')) return 'badge-red'
  return 'badge-gray'
}

function eventSummary(evt: WSEvent) {
  const d = evt.data
  switch (evt.type) {
    case 'phase.started': return `Phase ${phaseLabel(String(d?.phase ?? ''))} started`
    case 'phase.completed': return `Phase ${phaseLabel(String(d?.phase ?? ''))} completed`
    case 'phase.gate': return `Approval gate: ${phaseLabel(String(d?.phase ?? ''))}`
    case 'agent.spawned': return `${d?.agent} started${d?.issue ? ` (${d.issue})` : ''}`
    case 'agent.completed': return `${d?.agent} finished${d?.issue ? ` (${d.issue})` : ''}`
    case 'issue.status_changed': return `${d?.issue_id}: ${d?.status}`
    case 'error': return `${d?.error || 'Unknown error'}`
    default: return evt.type
  }
}

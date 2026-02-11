import { useState, useEffect } from 'react'
import { fetchMetrics, type MetricsResponse } from '../../lib/api'
import { Coins, Zap, Clock, BarChart3, ChevronDown, ChevronRight } from 'lucide-react'

export function MetricsSummary() {
  const [expanded, setExpanded] = useState(false)
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null)

  // Only poll when expanded
  useEffect(() => {
    if (!expanded) return
    fetchMetrics().then(setMetrics).catch(() => {})
    const interval = setInterval(() => {
      fetchMetrics().then(setMetrics).catch(() => {})
    }, 5000)
    return () => clearInterval(interval)
  }, [expanded])

  return (
    <div className="card">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between text-sm font-medium text-gray-400 hover:text-gray-200 transition-colors"
      >
        <div className="flex items-center gap-2">
          <BarChart3 size={16} />
          Metrics
        </div>
        {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      </button>

      {expanded && (
        <div className="mt-4 space-y-6">
          {!metrics ? (
            <p className="text-gray-500 text-sm">Loading metrics...</p>
          ) : (
            <>
              {/* Summary Cards */}
              <div className="grid grid-cols-4 gap-4">
                <MiniCard
                  icon={<Zap size={16} />}
                  label="Total Tokens"
                  value={formatNumber(metrics.tokens_in + metrics.tokens_out)}
                  sub={`${formatNumber(metrics.tokens_in)} in / ${formatNumber(metrics.tokens_out)} out`}
                  color="text-blue-400"
                />
                <MiniCard
                  icon={<Coins size={16} />}
                  label="Estimated Cost"
                  value={`$${metrics.total_cost.toFixed(4)}`}
                  color="text-green-400"
                />
                <MiniCard
                  icon={<BarChart3 size={16} />}
                  label="Agent Calls"
                  value={Object.values(metrics.by_agent).reduce((sum, a) => sum + a.calls, 0).toString()}
                  color="text-purple-400"
                />
                <MiniCard
                  icon={<Clock size={16} />}
                  label="Total Time"
                  value={formatDuration(Object.values(metrics.phase_timings).reduce((a, b) => a + b, 0))}
                  color="text-yellow-400"
                />
              </div>

              {/* Agent Breakdown */}
              {Object.keys(metrics.by_agent).length > 0 && (
                <div>
                  <h3 className="text-sm font-medium text-gray-400 mb-3">Usage by Agent</h3>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="text-gray-500 text-xs">
                          <th className="text-left py-2 pr-4">Agent</th>
                          <th className="text-right py-2 px-4">Calls</th>
                          <th className="text-right py-2 px-4">Tokens In</th>
                          <th className="text-right py-2 px-4">Tokens Out</th>
                          <th className="text-right py-2 px-4">Cost</th>
                          <th className="text-right py-2 pl-4">Share</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(metrics.by_agent)
                          .sort(([, a], [, b]) => b.cost - a.cost)
                          .map(([agent, usage]) => (
                            <tr key={agent} className="border-t border-gray-800">
                              <td className="py-2 pr-4 font-mono text-gray-300">{agent}</td>
                              <td className="py-2 px-4 text-right text-gray-400">{usage.calls}</td>
                              <td className="py-2 px-4 text-right text-gray-400">{formatNumber(usage.tokens_in)}</td>
                              <td className="py-2 px-4 text-right text-gray-400">{formatNumber(usage.tokens_out)}</td>
                              <td className="py-2 px-4 text-right text-gray-300">${usage.cost.toFixed(4)}</td>
                              <td className="py-2 pl-4 text-right">
                                <div className="flex items-center justify-end gap-2">
                                  <div className="w-20 h-1.5 bg-gray-800 rounded-full overflow-hidden">
                                    <div
                                      className="h-full bg-blue-500 rounded-full"
                                      style={{
                                        width: `${metrics.total_cost > 0 ? (usage.cost / metrics.total_cost) * 100 : 0}%`,
                                      }}
                                    />
                                  </div>
                                  <span className="text-xs text-gray-500 w-10 text-right">
                                    {metrics.total_cost > 0 ? ((usage.cost / metrics.total_cost) * 100).toFixed(0) : 0}%
                                  </span>
                                </div>
                              </td>
                            </tr>
                          ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {/* Phase Timings */}
              {Object.keys(metrics.phase_timings).length > 0 && (
                <div>
                  <h3 className="text-sm font-medium text-gray-400 mb-3">Phase Timings</h3>
                  <div className="space-y-3">
                    {Object.entries(metrics.phase_timings).map(([phase, ms]) => {
                      const maxMs = Math.max(...Object.values(metrics.phase_timings))
                      return (
                        <div key={phase} className="flex items-center gap-3">
                          <span className="text-sm text-gray-400 w-24 capitalize">{phase}</span>
                          <div className="flex-1 h-2 bg-gray-800 rounded-full overflow-hidden">
                            <div
                              className="h-full bg-brand-500 rounded-full transition-all"
                              style={{ width: `${maxMs > 0 ? (ms / maxMs) * 100 : 0}%` }}
                            />
                          </div>
                          <span className="text-xs text-gray-500 w-20 text-right">{formatDuration(ms)}</span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function MiniCard({ icon, label, value, sub, color }: {
  icon: React.ReactNode
  label: string
  value: string
  sub?: string
  color: string
}) {
  return (
    <div className="bg-gray-800/50 rounded-lg p-3">
      <div className={`flex items-center gap-2 ${color} mb-1`}>
        {icon}
        <span className="text-xs text-gray-500">{label}</span>
      </div>
      <div className="text-lg font-semibold text-white">{value}</div>
      {sub && <div className="text-xs text-gray-500 mt-0.5">{sub}</div>}
    </div>
  )
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toString()
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  const seconds = ms / 1000
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  const minutes = seconds / 60
  if (minutes < 60) return `${minutes.toFixed(1)}m`
  const hours = minutes / 60
  return `${hours.toFixed(1)}h`
}

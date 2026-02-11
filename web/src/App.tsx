import { useState, useEffect } from 'react'
import { Dashboard } from './components/Dashboard'
import { IssueBoard } from './components/IssueBoard'
import { AgentLogs } from './components/AgentLogs'
import { ManifestEditor } from './components/ManifestEditor'
import { RunSelector } from './components/RunSelector'
import { Runs } from './components/Runs'
import { Help } from './components/Help'
import { useWebSocket } from './hooks/useWebSocket'
import { fetchStatus, type StatusResponse } from './lib/api'
import {
  LayoutDashboard,
  KanbanSquare,
  Terminal,
  Settings,
  Play,
  Wifi,
  WifiOff,
  HelpCircle,
} from 'lucide-react'

type Tab = 'dashboard' | 'runs' | 'issues' | 'logs' | 'config' | 'help'

export default function App() {
  const [tab, setTab] = useState<Tab>(() => {
    const saved = localStorage.getItem('sdlc:tab')
    return (saved as Tab) || 'dashboard'
  })
  const [status, setStatus] = useState<StatusResponse | null>(null)
  const { events, connected } = useWebSocket()

  useEffect(() => {
    fetchStatus().then(setStatus).catch(() => {})
    const interval = setInterval(() => {
      fetchStatus().then(setStatus).catch(() => {})
    }, 5000)
    return () => clearInterval(interval)
  }, [])

  // Persist active tab to localStorage
  useEffect(() => {
    localStorage.setItem('sdlc:tab', tab)
  }, [tab])

  // Filter events by active run_id
  const activeRunId = status?.run_id ?? ''
  const runEvents = events.filter(
    (e) => !e.run_id || e.run_id === activeRunId
  )

  const tabs: { id: Tab; label: string; icon: typeof LayoutDashboard }[] = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'runs', label: 'Runs', icon: Play },
    { id: 'issues', label: 'Issues', icon: KanbanSquare },
    { id: 'logs', label: 'Logs', icon: Terminal },
    { id: 'config', label: 'Config', icon: Settings },
    { id: 'help', label: 'Help', icon: HelpCircle },
  ]

  return (
    <div className="min-h-screen bg-gray-950">
      {/* Header */}
      <header className="border-b border-gray-800 bg-gray-950/80 backdrop-blur-sm sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-lg font-semibold text-white">Agentic SDLC</span>
            {status && (
              <RunSelector
                currentRunId={status.run_id}
                onRunSelected={() => fetchStatus().then(setStatus).catch(() => {})}
              />
            )}
          </div>

          <nav className="flex items-center gap-1">
            {tabs.map(({ id, label, icon: Icon }) => (
              <button
                key={id}
                onClick={() => setTab(id)}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-sm transition-colors ${
                  tab === id
                    ? 'bg-gray-800 text-white'
                    : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
                }`}
              >
                <Icon size={16} />
                {label}
              </button>
            ))}
          </nav>

          <div className="flex items-center gap-2">
            {connected ? (
              <span className="flex items-center gap-1.5 text-xs text-green-400">
                <Wifi size={14} /> Live
              </span>
            ) : (
              <span className="flex items-center gap-1.5 text-xs text-red-400">
                <WifiOff size={14} /> Offline
              </span>
            )}
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="max-w-7xl mx-auto px-4 py-6">
        {tab === 'dashboard' && <Dashboard status={status} events={runEvents} />}
        {tab === 'runs' && <Runs status={status} events={runEvents} />}
        {tab === 'issues' && <IssueBoard />}
        {tab === 'logs' && <AgentLogs events={runEvents} />}
        {tab === 'config' && <ManifestEditor />}
        {tab === 'help' && <Help />}
      </main>
    </div>
  )
}

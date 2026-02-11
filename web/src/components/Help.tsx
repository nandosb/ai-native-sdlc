import { useState } from 'react'
import {
  BookOpen,
  Rocket,
  PenTool,
  Network,
  ListChecks,
  Code2,
  ChevronRight,
  Terminal,
  GitBranch,
  Bot,
  FileText,
  ArrowRight,
  Zap,
  Shield,
  LayoutDashboard,
  Play,
  KanbanSquare,
  ScrollText,
  Settings,
} from 'lucide-react'

type Section = 'overview' | 'phases' | 'ui-guide' | 'cli'

const sections: { id: Section; label: string; icon: typeof BookOpen }[] = [
  { id: 'overview', label: 'Overview', icon: BookOpen },
  { id: 'phases', label: 'SDLC Phases', icon: Rocket },
  { id: 'ui-guide', label: 'UI Guide', icon: LayoutDashboard },
  { id: 'cli', label: 'CLI Reference', icon: Terminal },
]

const phases = [
  {
    number: 1,
    name: 'Bootstrap',
    icon: FileText,
    color: 'blue',
    tagline: 'Lay the foundation',
    summary:
      'Generates essential documentation (CLAUDE.md and ARCHITECTURE.md) for each repository so that AI agents can understand your codebase before making changes.',
    details: [
      'Auto-detects the project language (Go, TypeScript, Python, Rust, Java)',
      'Creates CLAUDE.md with coding conventions, build instructions, and project rules',
      'Creates ARCHITECTURE.md with a structural overview of the codebase',
      'Only runs once per repo — skipped if already bootstrapped',
    ],
    input: 'Repository path',
    output: 'CLAUDE.md + ARCHITECTURE.md committed to each repo',
    agent: 'doc-generator',
    model: 'Sonnet',
  },
  {
    number: 2,
    name: 'Design',
    icon: PenTool,
    color: 'purple',
    tagline: 'Shape the solution',
    summary:
      'Takes your Product Requirements Document (PRD) and produces a detailed scoping document that defines how the feature will be implemented across your repositories.',
    details: [
      'Accepts PRDs from local files, Notion pages, or URLs',
      'Analyzes repo structure and existing code to inform the design',
      'Produces a scoping document with architecture decisions, component breakdown, and integration points',
      'Optionally writes the scoping document back to Notion',
    ],
    input: 'PRD (Product Requirements Document)',
    output: 'Scoping document (markdown)',
    agent: 'solution-designer',
    model: 'Opus',
  },
  {
    number: 3,
    name: 'Planning',
    icon: Network,
    color: 'cyan',
    tagline: 'Break it down',
    summary:
      'Transforms the scoping document into a PERT chart — a structured task list with dependency ordering, estimates, and repo assignments.',
    details: [
      'Decomposes the solution into granular, implementable tasks',
      'Maps dependencies between tasks so they can be parallelized safely',
      'Assigns each task to the correct repository',
      'Provides time estimates for each unit of work',
    ],
    input: 'Scoping document from Design phase',
    output: 'PERT document (structured tasks with dependencies)',
    agent: 'task-decomposer',
    model: 'Opus',
  },
  {
    number: 4,
    name: 'Tracking',
    icon: ListChecks,
    color: 'yellow',
    tagline: 'Organize the work',
    summary:
      'Creates Linear issues from the PERT document, establishing blocking relationships so the execution phase knows which tasks can run in parallel.',
    details: [
      'Creates one Linear issue per task with title, description, and estimates',
      'Sets up blocking/blocked-by relationships between issues',
      'Issues matched by title to prevent duplicates on re-runs',
      'Falls back to Claude MCP if the Linear API key is not configured',
    ],
    input: 'PERT document from Planning phase',
    output: 'Linear issues with dependency relationships',
    agent: 'linear-issue-creator',
    model: 'Sonnet',
  },
  {
    number: 5,
    name: 'Executing',
    icon: Code2,
    color: 'green',
    tagline: 'Ship the code',
    summary:
      'Implements each issue in an isolated git worktree, runs code through AI review cycles, and opens pull requests — all in parallel where dependencies allow.',
    details: [
      'Topologically sorts issues and processes them in dependency-safe batches',
      'Each issue gets its own git worktree — your main branch is never touched',
      'A Coder agent writes the implementation based on the issue description',
      'A Quality Reviewer agent reviews the code (up to 3 iterations)',
      'A Feedback Writer agent generates the PR description',
      'PRs are created automatically via the GitHub CLI',
    ],
    input: 'Linear issues from Tracking phase',
    output: 'Pull requests with reviewed code',
    agent: 'coder + quality-reviewer + feedback-writer',
    model: 'Opus / Sonnet',
  },
]

const colorMap: Record<string, { bg: string; text: string; border: string; ring: string; glow: string }> = {
  blue:   { bg: 'bg-blue-500/10',   text: 'text-blue-400',   border: 'border-blue-500/20',   ring: 'ring-blue-500/30',   glow: 'shadow-blue-500/5' },
  purple: { bg: 'bg-purple-500/10', text: 'text-purple-400', border: 'border-purple-500/20', ring: 'ring-purple-500/30', glow: 'shadow-purple-500/5' },
  cyan:   { bg: 'bg-cyan-500/10',   text: 'text-cyan-400',   border: 'border-cyan-500/20',   ring: 'ring-cyan-500/30',   glow: 'shadow-cyan-500/5' },
  yellow: { bg: 'bg-yellow-500/10', text: 'text-yellow-400', border: 'border-yellow-500/20', ring: 'ring-yellow-500/30', glow: 'shadow-yellow-500/5' },
  green:  { bg: 'bg-green-500/10',  text: 'text-green-400',  border: 'border-green-500/20',  ring: 'ring-green-500/30',  glow: 'shadow-green-500/5' },
}

export function Help() {
  const [section, setSection] = useState<Section>('phases')
  const [expandedPhase, setExpandedPhase] = useState<number | null>(null)

  return (
    <div className="flex gap-6 min-h-[calc(100vh-8rem)]">
      {/* Sidebar */}
      <nav className="w-48 flex-shrink-0">
        <div className="sticky top-20 space-y-1">
          {sections.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setSection(id)}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors ${
                section === id
                  ? 'bg-gray-800 text-white'
                  : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
              }`}
            >
              <Icon size={16} />
              {label}
            </button>
          ))}
        </div>
      </nav>

      {/* Content */}
      <div className="flex-1 min-w-0">
        {section === 'overview' && <OverviewSection />}
        {section === 'phases' && (
          <PhasesSection
            expandedPhase={expandedPhase}
            onToggle={(n) => setExpandedPhase(expandedPhase === n ? null : n)}
          />
        )}
        {section === 'ui-guide' && <UIGuideSection />}
        {section === 'cli' && <CLISection />}
      </div>
    </div>
  )
}

/* ─── Overview ─── */

function OverviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-white mb-2">Agentic SDLC</h1>
        <p className="text-gray-400 leading-relaxed max-w-2xl">
          An AI-powered software development lifecycle tool that takes you from a product
          requirements document to approved pull requests — automatically. It orchestrates
          AI agents across five phases, using git worktrees for isolation and Linear for
          project tracking.
        </p>
      </div>

      <div className="card">
        <h2 className="text-sm font-medium text-gray-400 mb-4">How it works</h2>
        <div className="flex items-center gap-2 flex-wrap">
          {['PRD', 'Bootstrap', 'Design', 'Planning', 'Tracking', 'Executing', 'PRs'].map(
            (step, i, arr) => (
              <div key={step} className="flex items-center gap-2">
                <span
                  className={`px-3 py-1.5 rounded-md text-xs font-medium ${
                    i === 0 || i === arr.length - 1
                      ? 'bg-gray-800 text-gray-300 border border-gray-700'
                      : 'bg-blue-500/10 text-blue-400 border border-blue-500/20'
                  }`}
                >
                  {step}
                </span>
                {i < arr.length - 1 && <ArrowRight size={14} className="text-gray-600" />}
              </div>
            )
          )}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="card">
          <div className="flex items-center gap-2 mb-2">
            <GitBranch size={16} className="text-blue-400" />
            <h3 className="text-sm font-medium text-white">Git Isolation</h3>
          </div>
          <p className="text-xs text-gray-400 leading-relaxed">
            Each issue is implemented in its own git worktree. Your main branch is never
            touched during execution.
          </p>
        </div>
        <div className="card">
          <div className="flex items-center gap-2 mb-2">
            <Bot size={16} className="text-purple-400" />
            <h3 className="text-sm font-medium text-white">AI Where It Matters</h3>
          </div>
          <p className="text-xs text-gray-400 leading-relaxed">
            LLMs are only used for reasoning tasks. Deterministic operations like API calls,
            file checks, and JSON parsing are handled in Go.
          </p>
        </div>
        <div className="card">
          <div className="flex items-center gap-2 mb-2">
            <Shield size={16} className="text-green-400" />
            <h3 className="text-sm font-medium text-white">Resilient State</h3>
          </div>
          <p className="text-xs text-gray-400 leading-relaxed">
            State is persisted atomically to disk before every action. If the process
            crashes, it resumes exactly where it left off.
          </p>
        </div>
      </div>

      <div className="card">
        <h2 className="text-sm font-medium text-gray-400 mb-3">Required integrations</h2>
        <div className="grid grid-cols-2 gap-x-8 gap-y-2 text-sm">
          {[
            { name: 'Claude CLI', desc: 'AI agent execution', env: 'claude (installed)' },
            { name: 'GitHub CLI', desc: 'PR creation & repo access', env: 'gh (installed)' },
            { name: 'Linear', desc: 'Issue tracking', env: 'LINEAR_API_KEY' },
            { name: 'Notion', desc: 'PRD source (optional)', env: 'NOTION_API_KEY' },
          ].map((item) => (
            <div key={item.name} className="flex items-center justify-between py-1.5 border-b border-gray-800 last:border-0">
              <div>
                <span className="text-gray-300">{item.name}</span>
                <span className="text-gray-600 ml-2 text-xs">{item.desc}</span>
              </div>
              <code className="text-[11px] text-gray-500 font-mono">{item.env}</code>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

/* ─── Phases ─── */

function PhasesSection({
  expandedPhase,
  onToggle,
}: {
  expandedPhase: number | null
  onToggle: (n: number) => void
}) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-white mb-2">SDLC Phases</h1>
        <p className="text-gray-400 text-sm max-w-2xl leading-relaxed">
          The Agentic SDLC pipeline runs through five sequential phases. Each phase produces
          an artifact that feeds into the next, with approval gates between them so you stay
          in control.
        </p>
      </div>

      {/* Visual pipeline */}
      <div className="card overflow-hidden">
        <div className="flex items-stretch">
          {phases.map((phase, i) => {
            const c = colorMap[phase.color]
            return (
              <div key={phase.number} className="flex items-stretch flex-1">
                <button
                  onClick={() => onToggle(phase.number)}
                  className={`flex-1 flex flex-col items-center gap-1.5 py-4 px-2 transition-all hover:bg-gray-800/50 ${
                    expandedPhase === phase.number ? 'bg-gray-800/60' : ''
                  }`}
                >
                  <div className={`w-9 h-9 rounded-full ${c.bg} ${c.border} border flex items-center justify-center`}>
                    <phase.icon size={16} className={c.text} />
                  </div>
                  <span className={`text-[11px] font-bold ${c.text}`}>{phase.number}</span>
                  <span className="text-xs text-gray-300 font-medium">{phase.name}</span>
                  <span className="text-[10px] text-gray-500">{phase.tagline}</span>
                </button>
                {i < phases.length - 1 && (
                  <div className="flex items-center px-0.5">
                    <ChevronRight size={14} className="text-gray-700" />
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {/* Phase cards */}
      <div className="space-y-3">
        {phases.map((phase) => {
          const c = colorMap[phase.color]
          const isExpanded = expandedPhase === phase.number
          return (
            <div
              key={phase.number}
              className={`border rounded-lg transition-all ${
                isExpanded
                  ? `${c.border} ring-1 ${c.ring} shadow-lg ${c.glow}`
                  : 'border-gray-800 hover:border-gray-700'
              }`}
            >
              <button
                onClick={() => onToggle(phase.number)}
                className="w-full flex items-center gap-4 px-5 py-4 text-left"
              >
                <div
                  className={`w-10 h-10 rounded-lg ${c.bg} ${c.border} border flex items-center justify-center flex-shrink-0`}
                >
                  <phase.icon size={18} className={c.text} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className={`text-xs font-bold ${c.text}`}>Phase {phase.number}</span>
                    <span className="text-sm font-medium text-white">{phase.name}</span>
                    <span className="text-xs text-gray-500 italic hidden sm:inline">
                      {phase.tagline}
                    </span>
                  </div>
                  <p className="text-xs text-gray-400 mt-0.5 line-clamp-1">{phase.summary}</p>
                </div>
                <ChevronRight
                  size={16}
                  className={`text-gray-600 transition-transform flex-shrink-0 ${
                    isExpanded ? 'rotate-90' : ''
                  }`}
                />
              </button>

              {isExpanded && (
                <div className="px-5 pb-5 pt-0 space-y-4 border-t border-gray-800/50">
                  <p className="text-sm text-gray-300 leading-relaxed pt-4">{phase.summary}</p>

                  <div>
                    <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
                      What happens
                    </h4>
                    <ul className="space-y-1.5">
                      {phase.details.map((d, i) => (
                        <li key={i} className="flex items-start gap-2 text-sm text-gray-400">
                          <Zap size={12} className={`${c.text} mt-1 flex-shrink-0`} />
                          {d}
                        </li>
                      ))}
                    </ul>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="rounded-md bg-gray-800/50 px-3 py-2.5">
                      <span className="text-[10px] font-medium text-gray-500 uppercase tracking-wider">
                        Input
                      </span>
                      <p className="text-sm text-gray-300 mt-0.5">{phase.input}</p>
                    </div>
                    <div className="rounded-md bg-gray-800/50 px-3 py-2.5">
                      <span className="text-[10px] font-medium text-gray-500 uppercase tracking-wider">
                        Output
                      </span>
                      <p className="text-sm text-gray-300 mt-0.5">{phase.output}</p>
                    </div>
                  </div>

                  <div className="flex items-center gap-4 text-xs">
                    <div className="flex items-center gap-1.5">
                      <Bot size={12} className="text-gray-500" />
                      <span className="text-gray-500">Agent:</span>
                      <span className="text-gray-300 font-mono">{phase.agent}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <Zap size={12} className="text-gray-500" />
                      <span className="text-gray-500">Model:</span>
                      <span className="text-gray-300">{phase.model}</span>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

/* ─── UI Guide ─── */

function UIGuideSection() {
  const tabs = [
    {
      icon: LayoutDashboard,
      name: 'Dashboard',
      desc: 'Overview of active runs, stats (executions, PRs, agents, integrations), recent events, and token usage metrics.',
    },
    {
      icon: Play,
      name: 'Runs',
      desc: 'Start new phase executions, view progress in real-time, and interact with agents through the chat interface when they need input.',
    },
    {
      icon: KanbanSquare,
      name: 'Issues',
      desc: 'Kanban board showing all tracked issues across six statuses: Blocked, Ready, Implementing, Reviewing, Awaiting Human, and Done.',
    },
    {
      icon: ScrollText,
      name: 'Logs',
      desc: 'Live agent activity stream. Filter by agent type or issue. Pause scrolling to inspect past output.',
    },
    {
      icon: Settings,
      name: 'Config',
      desc: 'Edit the manifest.yaml configuration — repositories, PRD URL, and pipeline settings.',
    },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-white mb-2">UI Guide</h1>
        <p className="text-gray-400 text-sm max-w-2xl leading-relaxed">
          A quick walkthrough of each section available in the navigation bar.
        </p>
      </div>

      <div className="space-y-3">
        {tabs.map(({ icon: Icon, name, desc }) => (
          <div key={name} className="card flex items-start gap-4">
            <div className="w-9 h-9 rounded-lg bg-gray-800 border border-gray-700 flex items-center justify-center flex-shrink-0">
              <Icon size={16} className="text-gray-300" />
            </div>
            <div>
              <h3 className="text-sm font-medium text-white">{name}</h3>
              <p className="text-xs text-gray-400 mt-0.5 leading-relaxed">{desc}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="card">
        <h2 className="text-sm font-medium text-gray-400 mb-3">Connection status</h2>
        <p className="text-xs text-gray-400 leading-relaxed">
          The indicator in the top-right corner shows whether the WebSocket connection is
          active. <span className="text-green-400 font-medium">Live</span> means events stream
          in real-time. <span className="text-red-400 font-medium">Offline</span> means the
          server is unreachable — the UI will reconnect automatically when the server is back.
        </p>
      </div>

      <div className="card">
        <h2 className="text-sm font-medium text-gray-400 mb-3">Running a phase</h2>
        <ol className="space-y-2 text-sm text-gray-400 list-decimal list-inside">
          <li>Go to the <span className="text-white font-medium">Runs</span> tab</li>
          <li>Click <span className="text-white font-medium">New Phase</span> in the sidebar</li>
          <li>Select the phase and fill in required parameters</li>
          <li>Monitor progress in the execution panel and chat interface</li>
          <li>When an approval gate is reached, use the chat to approve or provide feedback</li>
        </ol>
      </div>
    </div>
  )
}

/* ─── CLI Reference ─── */

function CLISection() {
  const commands = [
    { cmd: 'sdlc init', desc: 'Validate the manifest and initialize state.json' },
    { cmd: 'sdlc bootstrap', desc: 'Generate CLAUDE.md + ARCHITECTURE.md per repo' },
    { cmd: 'sdlc design', desc: 'PRD to scoping document (--prd <file>)' },
    { cmd: 'sdlc plan', desc: 'Scoping doc to PERT chart (--scoping-doc <file>)' },
    { cmd: 'sdlc track', desc: 'PERT to Linear issues' },
    { cmd: 'sdlc execute', desc: 'Issues to PRs (--parallel <n>)' },
    { cmd: 'sdlc run', desc: 'Full pipeline with approval gates' },
    { cmd: 'sdlc approve', desc: 'Approve the current gate' },
    { cmd: 'sdlc status', desc: 'Print current state' },
    { cmd: 'sdlc serve', desc: 'Start this web UI on :3000' },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-white mb-2">CLI Reference</h1>
        <p className="text-gray-400 text-sm max-w-2xl leading-relaxed">
          Every phase can be run standalone from the command line, or orchestrated through the
          full pipeline.
        </p>
      </div>

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800">
              <th className="text-left py-2 px-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                Command
              </th>
              <th className="text-left py-2 px-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                Description
              </th>
            </tr>
          </thead>
          <tbody>
            {commands.map(({ cmd, desc }) => (
              <tr key={cmd} className="border-b border-gray-800/50 last:border-0">
                <td className="py-2 px-3">
                  <code className="text-xs font-mono text-blue-400">{cmd}</code>
                </td>
                <td className="py-2 px-3 text-gray-400">{desc}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="card">
        <h2 className="text-sm font-medium text-gray-400 mb-3">Build from source</h2>
        <div className="space-y-2 text-xs font-mono">
          <div className="bg-gray-800/50 rounded-md px-3 py-2 text-gray-300">
            <span className="text-gray-500"># Build the React frontend</span>
            <br />
            cd web && npm install && npm run build
          </div>
          <div className="bg-gray-800/50 rounded-md px-3 py-2 text-gray-300">
            <span className="text-gray-500"># Build the Go binary</span>
            <br />
            go build -o sdlc ./cmd/sdlc/
          </div>
        </div>
      </div>

      <div className="card">
        <h2 className="text-sm font-medium text-gray-400 mb-3">State files</h2>
        <div className="grid grid-cols-3 gap-3 text-xs">
          <div className="bg-gray-800/50 rounded-md px-3 py-2">
            <code className="text-gray-300">state.json</code>
            <p className="text-gray-500 mt-0.5">Current run state, phase, issues, artifacts</p>
          </div>
          <div className="bg-gray-800/50 rounded-md px-3 py-2">
            <code className="text-gray-300">metrics.jsonl</code>
            <p className="text-gray-500 mt-0.5">Per-agent token usage, appended per invocation</p>
          </div>
          <div className="bg-gray-800/50 rounded-md px-3 py-2">
            <code className="text-gray-300">manifest.yaml</code>
            <p className="text-gray-500 mt-0.5">Pipeline configuration (repos, PRD, settings)</p>
          </div>
        </div>
      </div>
    </div>
  )
}

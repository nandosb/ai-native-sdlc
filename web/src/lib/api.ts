const BASE = ''

export interface StatusResponse {
  run_id: string
  phase: string
  phase_status: string
  prd_url: string
  repos: RepoConfig[]
  issue_count: number
  artifacts: Record<string, string>
  updated_at: string
}

export interface RepoConfig {
  name: string
  path: string
  team: string
  language?: string
}

export interface IssueState {
  id: string
  title: string
  repo: string
  status: IssueStatus
  linear_id?: string
  branch?: string
  worktree?: string
  pr_url?: string
  depends_on?: string[]
  iterations: number
}

export type IssueStatus =
  | 'blocked'
  | 'ready'
  | 'implementing'
  | 'reviewing'
  | 'awaiting_human'
  | 'done'

export interface IssuesResponse {
  issues: Record<string, IssueState>
  grouped: Record<IssueStatus, IssueState[]>
}

export interface MetricsResponse {
  tokens_in: number
  tokens_out: number
  total_cost: number
  by_agent: Record<string, AgentUsage>
  phase_timings: Record<string, number>
}

export interface AgentUsage {
  tokens_in: number
  tokens_out: number
  cost: number
  calls: number
}

export interface PhaseInfo {
  name: string
  order: number
  gate?: boolean
  status?: string
  current?: boolean
}

export async function fetchStatus(): Promise<StatusResponse> {
  const res = await fetch(`${BASE}/api/status`)
  return res.json()
}

export async function fetchIssues(): Promise<IssuesResponse> {
  const res = await fetch(`${BASE}/api/issues`)
  return res.json()
}

export async function fetchMetrics(): Promise<MetricsResponse> {
  const res = await fetch(`${BASE}/api/metrics`)
  return res.json()
}

export async function fetchPhases(): Promise<PhaseInfo[]> {
  const res = await fetch(`${BASE}/api/phases`)
  return res.json()
}

export interface ManifestData {
  prd: string
  repos: RepoConfig[]
}

export async function fetchManifest(): Promise<ManifestData> {
  const res = await fetch(`${BASE}/api/manifest`)
  return res.json()
}

export async function saveManifest(data: ManifestData): Promise<{ status: string }> {
  const res = await fetch(`${BASE}/api/manifest`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

export async function submitApproval(action: 'approve' | 'reject', comment?: string) {
  const res = await fetch(`${BASE}/api/approve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action, comment }),
  })
  return res.json()
}

export interface RunSummary {
  id: string
  phase: string
  phase_status: string
  prd_url: string
  issue_count: number
  created_at: string
  updated_at: string
}

export async function fetchRuns(): Promise<RunSummary[]> {
  const res = await fetch(`${BASE}/api/runs`)
  return res.json()
}

export async function selectRun(runId: string): Promise<{ status: string }> {
  const res = await fetch(`${BASE}/api/runs/select`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ run_id: runId }),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

// --- Artifact config (single source of truth from server) ---

export interface ArtifactDef {
  key: string
  default_file: string
  notion_title: string
}

export interface ArtifactConfigResponse {
  artifacts: Record<string, ArtifactDef>
  phase_artifact: Record<string, string>
}

export async function fetchArtifactConfig(): Promise<ArtifactConfigResponse> {
  const res = await fetch(`${BASE}/api/artifacts/config`)
  return res.json()
}

// --- Artifacts ---

export interface ArtifactResponse {
  key: string
  path: string
  content: string
}

export async function fetchArtifact(key: string): Promise<ArtifactResponse> {
  const res = await fetch(`${BASE}/api/artifacts/${encodeURIComponent(key)}`)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

// --- Phase execution ---

export async function runPhase(
  phase: string,
  params?: Record<string, string>
): Promise<{ status: string; phase: string }> {
  const res = await fetch(`${BASE}/api/phases/${phase}/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ params: params || {} }),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

export async function runAll(): Promise<{ status: string }> {
  const res = await fetch(`${BASE}/api/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

// --- Integration health ---

export interface IntegrationHealth {
  name: string
  ok: boolean
  detail?: string
  mode?: string // "api" | "mcp" | ""
  checked_at: string
}

export async function fetchIntegrationHealth(): Promise<IntegrationHealth[]> {
  const res = await fetch(`${BASE}/api/health/integrations`)
  return res.json()
}

// --- Executions (Runner tab) ---

export type ExecutionType = 'phase' | 'issue'
export type ExecutionStatus = 'running' | 'waiting_input' | 'completed' | 'failed' | 'cancelled'

export interface ExecutionMessage {
  role: 'system' | 'assistant' | 'user'
  content: string
  timestamp: string
}

export interface Execution {
  id: string
  run_id: string
  type: ExecutionType
  phase: string
  issue_id?: string
  status: ExecutionStatus
  session_id: string
  messages: ExecutionMessage[]
  params: Record<string, string>
  parent_id?: string
  created_at: string
  updated_at: string
  tokens_in: number
  tokens_out: number
}

export async function fetchExecutions(runId?: string): Promise<Execution[]> {
  const qs = runId ? `?run_id=${encodeURIComponent(runId)}` : ''
  const res = await fetch(`${BASE}/api/executions${qs}`)
  return res.json()
}

export async function fetchExecution(id: string): Promise<Execution> {
  const res = await fetch(`${BASE}/api/executions/${encodeURIComponent(id)}`)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

export async function createExecution(req: {
  run_id?: string
  type?: ExecutionType
  phase: string
  issue_id?: string
  params?: Record<string, string>
}): Promise<{ id: string; session_id: string; status: string }> {
  const res = await fetch(`${BASE}/api/executions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

export async function sendExecutionMessage(
  id: string,
  content: string,
): Promise<{ status: string }> {
  const res = await fetch(`${BASE}/api/executions/${encodeURIComponent(id)}/message`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

export async function approveExecution(id: string): Promise<{ status: string }> {
  const res = await fetch(`${BASE}/api/executions/${encodeURIComponent(id)}/approve`, {
    method: 'POST',
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

export async function cancelExecution(id: string): Promise<{ status: string }> {
  const res = await fetch(`${BASE}/api/executions/${encodeURIComponent(id)}/cancel`, {
    method: 'POST',
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text)
  }
  return res.json()
}

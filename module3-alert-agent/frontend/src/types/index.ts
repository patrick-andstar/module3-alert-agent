export interface EventBase {
  host_id: string
  user_id: string
  file_path: string
  file_hash: string
  sensitive: boolean
  sensitive_type: string
  risk_level: 'critical' | 'high' | 'medium' | 'low' | 'info'
  process_name: string
  process_path: string
  target: string
  operation: string
  sensitive_file_id: string
}

export interface Event extends EventBase {
  event_id: string
  timestamp: number
}

export interface EventBatch {
  host_id: string
  events: Event[]
}

export interface ScenarioStep {
  label: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE'
  path: string
  body?: Record<string, unknown>
}

export interface Scenario {
  id: string
  title: string
  summary: string
  kind?: 'primary' | 'utility' | 'secondary'
  stageFocus?: 'whitelist' | 'dedup' | 'recall_agent' | 'true_alert' | 'seed' | 'uncertain'
  outcomeHint?: string
  steps: () => ScenarioStep[]
}

export interface StepResult {
  label: string
  request: {
    method: string
    path: string
    body: Record<string, unknown> | null
  }
  status: number
  response: unknown
}

export interface AlertRecord {
  event_id: string
  host_id: string
  user_id: string
  sensitive_type: string
  risk_level: string
  process_name: string
  target: string
  operation: string
  timestamp: number
  is_merge_event?: boolean
  file_count?: number
  agent_verdict?: string
  agent_confidence?: number
  agent_explanation?: string
  recall_score?: number
  false_positive_reason?: string
  [key: string]: unknown
}

export interface AlertsQueryResponse {
  data: AlertRecord[]
  total: number
  page: number
  page_size: number
}

export interface FalsePositiveRecord {
  id: number
  scenario_key: string
  host_id: string
  user_id: string
  sensitive_type: string
  risk_level: string
  process_name: string
  process_path: string
  target: string
  operation: string
  reason: string
  hit_count: number
  last_seen_at: string
  expired_at: string
  created_at: string
  [key: string]: unknown
}

export interface WhitelistRule {
  id: number
  rule_name: string
  logic: 'AND' | 'OR'
  process_name: string
  user_id: string
  file_path_pattern: string
  time_window_start: string
  time_window_end: string
  enabled: boolean
  [key: string]: unknown
}

export type RunState = 'idle' | 'running' | 'ok' | 'error'

export interface DashboardStats {
  alertCount: number
  fpCount: number
  whitelistCount: number
}

import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Database,
  FileWarning,
  Loader2,
  Play,
  Radio,
  RefreshCw,
  ShieldCheck,
  Sparkles,
  Terminal,
  Zap,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/toaster'
import { ScenarioRunner } from '@/components/scenarios/ScenarioRunner'
import { AlertLogsTable } from '@/components/records/AlertLogsTable'
import { FalsePositivesTable } from '@/components/records/FalsePositivesTable'
import { WhitelistTable } from '@/components/records/WhitelistTable'
import { useScenarios } from '@/hooks/useScenarios'
import { useRecords } from '@/hooks/useRecords'
import { cn } from '@/lib/utils'
import type { AlertRecord, RunState, Scenario, StepResult } from '@/types'

type PipelineStage = {
  id: string
  label: string
  short: string
  description: string
  proof: string
}

const pipelineStages: PipelineStage[] = [
  {
    id: 'whitelist',
    label: '白名单过滤',
    short: 'Rule',
    description: '确定性规则先挡掉已知噪声',
    proof: 'dropped > 0',
  },
  {
    id: 'dedup',
    label: '去重合并',
    short: 'Merge',
    description: '相同行为在窗口内聚合',
    proof: 'file_count',
  },
  {
    id: 'recall',
    label: '结构化召回',
    short: 'Recall',
    description: '从误报模式库召回相似场景',
    proof: 'recall_score',
  },
  {
    id: 'react',
    label: 'ReAct 精判',
    short: 'Reason',
    description: 'Eino Agent 输出结论和解释',
    proof: 'confidence',
  },
  {
    id: 'query',
    label: '最终入库查询',
    short: 'Query',
    description: '模块四通过查询 API 获取证据',
    proof: '/api/alerts/query',
  },
]

const stageByFocus: Record<string, string[]> = {
  whitelist: ['whitelist'],
  dedup: ['dedup', 'query'],
  seed: ['recall'],
  recall_agent: ['recall', 'react', 'query'],
  true_alert: ['react', 'query'],
  uncertain: ['recall', 'react', 'query'],
}

const runStateLabel: Record<RunState, string> = {
  idle: '就绪',
  running: '实时分析中',
  ok: '演示完成',
  error: '执行异常',
}

const mainScenarioIds = new Set(['whitelist_drop', 'confirmed_false_positive', 'true_alert'])

function getLastResponse(results: StepResult[]): Record<string, unknown> | null {
  const last = results[results.length - 1]?.response
  return last && typeof last === 'object' && !Array.isArray(last) ? (last as Record<string, unknown>) : null
}

function getLatestRelevantAlert(activeScenario: Scenario | null, alerts: AlertRecord[]): AlertRecord | undefined {
  if (!activeScenario) return alerts[0]

  if (activeScenario.id === 'whitelist_drop' || activeScenario.id === 'prepare_demo_data') {
    return undefined
  }
  if (activeScenario.id === 'confirmed_false_positive') {
    return alerts.find((alert) => alert.target === 'internal-crm.company.com') || alerts[0]
  }
  if (activeScenario.id === 'true_alert') {
    return alerts.find((alert) => alert.target === 'mail.qq.com') || alerts[0]
  }
  return alerts[0]
}

function displayValue(value: unknown, fallback = '—') {
  if (value === null || value === undefined || value === '') return fallback
  if (typeof value === 'number') return Number.isInteger(value) ? String(value) : value.toFixed(2)
  return String(value)
}

function riskTone(risk?: string) {
  if (risk === 'critical') return 'text-[#ff5d6c]'
  if (risk === 'high') return 'text-[#ff9f52]'
  if (risk === 'medium') return 'text-[#ffca6a]'
  if (risk === 'low') return 'text-[#5df0bd]'
  if (risk === 'info') return 'text-[#aab8c5]'
  return 'text-[#f5f1e7]'
}

function verdictTone(verdict?: string) {
  if (verdict === 'false_positive') return 'text-[#5df0bd]'
  if (verdict === 'true_alert') return 'text-[#ff5d6c]'
  if (verdict === 'uncertain') return 'text-[#ffbd5b]'
  return 'text-[#f5f1e7]'
}

function EvidenceMetric({
  label,
  value,
  tone,
}: {
  label: string
  value: unknown
  tone?: string
}) {
  return (
    <div className="theater-metric">
      <span>{label}</span>
      <strong className={tone}>{displayValue(value)}</strong>
    </div>
  )
}

function ScenarioButton({
  scenario,
  active,
  running,
  onClick,
}: {
  scenario: Scenario
  active: boolean
  running: boolean
  onClick: (scenario: Scenario) => void
}) {
  const Icon =
    scenario.id === 'whitelist_drop'
      ? ShieldCheck
      : scenario.id === 'true_alert'
        ? FileWarning
        : Sparkles

  return (
    <button
      type="button"
      onClick={() => onClick(scenario)}
      disabled={running}
      className={cn('theater-scenario-button', active && 'is-active')}
    >
      <span className="theater-scenario-index">
        <Icon className="h-4 w-4" />
      </span>
      <span className="min-w-0">
        <b>{scenario.title}</b>
        <small>{scenario.outcomeHint || scenario.summary}</small>
      </span>
      {active && running ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
    </button>
  )
}

export default function App() {
  const [token, setToken] = useState('')
  const { scenarios, activeScenario, steps, results, runState, error, runScenario } = useScenarios()
  const { alerts, falsePositives, whitelist, stats, loading, refreshRecords } = useRecords()

  const handleRefresh = useCallback(() => {
    refreshRecords(token)
  }, [refreshRecords, token])

  const handleScenarioClick = useCallback(
    async (scenario: Scenario) => {
      await runScenario(scenario, token, () => refreshRecords(token))
    },
    [runScenario, token, refreshRecords]
  )

  useEffect(() => {
    refreshRecords(token)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const primaryScenarios = useMemo(
    () => scenarios.filter((scenario) => mainScenarioIds.has(scenario.id)),
    [scenarios]
  )
  const prepareScenario = useMemo(
    () => scenarios.find((scenario) => scenario.id === 'prepare_demo_data'),
    [scenarios]
  )
  const secondaryScenarios = useMemo(
    () => scenarios.filter((scenario) => !mainScenarioIds.has(scenario.id) && scenario.id !== 'prepare_demo_data'),
    [scenarios]
  )

  const activeStages = useMemo(() => {
    if (runState === 'running') return pipelineStages.map((stage) => stage.id)
    if (!activeScenario?.stageFocus) return ['whitelist', 'dedup', 'recall', 'react', 'query']
    return stageByFocus[activeScenario.stageFocus] || []
  }, [activeScenario?.stageFocus, runState])

  const latestAlert = getLatestRelevantAlert(activeScenario, alerts)
  const lastResponse = getLastResponse(results)
  const accepted = lastResponse?.accepted
  const dropped = lastResponse?.dropped
  const statusText = error
    ? '请求失败，保留原始响应'
    : runState === 'running'
      ? '正在调用真实 API'
      : runState === 'ok'
        ? '真实接口已返回'
        : '等待选择演示场景'

  const verdict = latestAlert?.agent_verdict
  const oldRisk = displayValue(latestAlert?.old_risk_level || latestAlert?.risk_level, 'HIGH')
  const newRisk = displayValue(latestAlert?.risk_level, activeScenario?.id === 'true_alert' ? 'CRITICAL' : 'LOW')

  return (
    <TooltipProvider>
      <div className="theater-root">
        <span className="sr-only">DLP Alert Agent Console</span>

        <main className="theater-page">
          <section className="theater-hero" aria-label="DLP Alert Agent Demo Theater">
            <aside className="theater-report-panel">
              <div className="theater-report-kicker">
                <Radio className="h-4 w-4" />
                LIVE DEMO
              </div>
              <h1>
                <span>告警日志误报分析Agent</span>
                <span>Agent</span>
              </h1>
              <p>
                用一条可解释流水线证明模块三的价值：先降噪，再保真，最后把 Agent 证据交给模块四查询。
              </p>

              <div className="theater-report-stats" aria-label="demo stats">
                <div>
                  <span>主演示场景</span>
                  <strong>3</strong>
                </div>
                <div>
                  <span>治理链路</span>
                  <strong>5</strong>
                </div>
                <div>
                  <span>真实接口</span>
                  <strong>API</strong>
                </div>
              </div>

              <label className="theater-token-label" htmlFor="admin-token">
                Admin Token
              </label>
              <Input
                id="admin-token"
                value={token}
                onChange={(event) => setToken(event.target.value)}
                placeholder="未配置可留空"
                className="theater-token-input"
                autoComplete="off"
              />

              <div className="theater-report-actions">
                <Button
                  type="button"
                  onClick={handleRefresh}
                  disabled={loading}
                  className="theater-light-button"
                >
                  <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
                  刷新证据
                </Button>
                {prepareScenario && (
                  <Button
                    type="button"
                    onClick={() => handleScenarioClick(prepareScenario)}
                    disabled={runState === 'running'}
                    className="theater-dark-button"
                  >
                    <Database className="h-4 w-4" />
                    准备演示数据
                  </Button>
                )}
              </div>
            </aside>

            <section className="theater-stage">
              <header className="theater-stage-header">
                <div>
                  <p className="theater-eyebrow">Explainable false-positive governance</p>
                  <h2>先规则过滤，再 Agent 精判</h2>
                  <span>{activeScenario?.summary || '选择一个场景，现场触发真实 API 调用。'}</span>
                </div>
                <div className={cn('theater-live-state', runState)}>
                  <span />
                  {runStateLabel[runState]}
                </div>
              </header>

              <div className="theater-pipeline" aria-label="processing pipeline">
                {pipelineStages.map((stage, index) => {
                  const active = activeStages.includes(stage.id)
                  return (
                    <div key={stage.id} className={cn('theater-pipeline-step', active && 'is-active')}>
                      <span className="theater-pipeline-number">{String(index + 1).padStart(2, '0')}</span>
                      <b>{stage.short}</b>
                      <strong>{stage.label}</strong>
                      <small>{stage.description}</small>
                      <code>{stage.proof}</code>
                    </div>
                  )
                })}
              </div>

              <div className="theater-stage-grid">
                <div className="theater-scenario-bank">
                  <div className="theater-section-title">
                    <Zap className="h-4 w-4" />
                    主演示按钮
                  </div>
                  <div className="theater-scenario-list">
                    {primaryScenarios.map((scenario) => (
                      <ScenarioButton
                        key={scenario.id}
                        scenario={scenario}
                        active={activeScenario?.id === scenario.id}
                        running={runState === 'running'}
                        onClick={handleScenarioClick}
                      />
                    ))}
                  </div>
                </div>

                <div className="theater-verdict-panel">
                  <div className="theater-section-title">
                    <Activity className="h-4 w-4" />
                    当前结论
                  </div>

                  <div className="theater-risk-shift">
                    <div>
                      <span>原始风险</span>
                      <strong className={riskTone(String(oldRisk).toLowerCase())}>{oldRisk}</strong>
                    </div>
                    <span className="theater-arrow">→</span>
                    <div>
                      <span>治理后</span>
                      <strong className={riskTone(String(newRisk).toLowerCase())}>{newRisk}</strong>
                    </div>
                  </div>

                  <div className="theater-metric-grid">
                    <EvidenceMetric label="verdict" value={verdict || (runState === 'idle' ? 'waiting' : 'pending')} tone={verdictTone(verdict)} />
                    <EvidenceMetric label="confidence" value={latestAlert?.agent_confidence} />
                    <EvidenceMetric label="recall_score" value={latestAlert?.recall_score} />
                    <EvidenceMetric label="accepted / dropped" value={`${displayValue(accepted, '—')} / ${displayValue(dropped, '—')}`} />
                  </div>

                  <div className="theater-explanation">
                    <span>{statusText}</span>
                    <p>
                      {error ||
                        latestAlert?.agent_explanation ||
                        activeScenario?.outcomeHint ||
                        '这里会展示真实接口返回后的 Agent 解释、风险变化和查询证据。'}
                    </p>
                  </div>
                </div>
              </div>
            </section>
          </section>

          <section className="theater-evidence" aria-label="evidence detail">
            <div className="theater-evidence-header">
              <div>
                <p className="theater-eyebrow">Evidence detail</p>
                <h2>真实 API 证据区</h2>
              </div>
              <div className="theater-record-counts">
                <span>alert_logs <b>{stats.alertCount}</b></span>
                <span>false_positive_library <b>{stats.fpCount}</b></span>
                <span>whitelist_rules <b>{stats.whitelistCount}</b></span>
              </div>
            </div>

            <ScenarioRunner steps={steps} results={results} runState={runState} error={error} />

            {secondaryScenarios.length > 0 && (
              <div className="theater-secondary">
                <div className="theater-section-title">
                  <Terminal className="h-4 w-4" />
                  备用场景
                </div>
                <div className="theater-secondary-list">
                  {secondaryScenarios.map((scenario) => (
                    <button
                      key={scenario.id}
                      type="button"
                      disabled={runState === 'running'}
                      onClick={() => handleScenarioClick(scenario)}
                      className={cn('theater-secondary-button', activeScenario?.id === scenario.id && 'is-active')}
                    >
                      <b>{scenario.title}</b>
                      <span>{scenario.outcomeHint || scenario.summary}</span>
                    </button>
                  ))}
                </div>
              </div>
            )}

            <div className="theater-tables">
              <AlertLogsTable data={alerts} loading={loading} />
              <FalsePositivesTable data={falsePositives} loading={loading} />
              <WhitelistTable data={whitelist} loading={loading} />
            </div>
          </section>

          {runState === 'error' && (
            <div className="theater-error-banner" role="alert">
              <AlertTriangle className="h-4 w-4" />
              <span>演示请求异常。原始响应已保留在证据区，请检查后端、Admin Token 或外部 Agent 服务。</span>
            </div>
          )}

          {runState === 'ok' && (
            <div className="theater-success-toast" role="status">
              <CheckCircle2 className="h-4 w-4" />
              <span>场景执行完成，证据记录已刷新。</span>
            </div>
          )}
        </main>
      </div>
      <Toaster />
    </TooltipProvider>
  )
}

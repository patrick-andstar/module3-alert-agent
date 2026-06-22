import { useState, useCallback } from 'react'
import type { Scenario, StepResult, RunState, ScenarioStep } from '@/types'

const nowSeconds = () => Math.floor(Date.now() / 1000)

const eventBase = {
  host_id: 'host-demo-01',
  user_id: 'alice',
  file_path: 'C:/Users/Alice/Desktop/customer.xlsx',
  file_hash: 'hash-demo-customer',
  sensitive: true,
  sensitive_type: 'customer',
  risk_level: 'high',
  process_name: 'dlp-demo-browser.exe',
  process_path: 'C:/DLPDemo/dlp-demo-browser.exe',
  target: 'internal-crm.company.com',
  operation: 'upload',
  sensitive_file_id: 'file-demo-001',
}

const scenarios: Scenario[] = [
  {
    id: 'prepare_demo_data',
    title: '准备演示数据',
    summary: '写入统一的 CRM 正常业务误报模式，后续场景仍走真实 API。',
    kind: 'utility',
    stageFocus: 'seed',
    outcomeHint: '合成但真实入库的演示数据包',
    steps: () => [
      {
        label: '写入 CRM 误报模式',
        method: 'POST',
        path: '/api/false-positives',
        body: {
          scenario_key: 'customer|upload|dlp-demo-browser.exe|internal-crm.company.com',
          user_id: 'alice',
          sensitive_type: 'customer',
          risk_level: 'low',
          process_name: 'dlp-demo-browser.exe',
          process_path: 'C:/DLPDemo/dlp-demo-browser.exe',
          target: 'internal-crm.company.com',
          operation: 'upload',
          reason: 'normal crm upload from trusted internal workflow',
          hit_count: 1,
        },
      },
    ],
  },
  {
    id: 'whitelist_drop',
    title: '白名单命中丢弃',
    summary: '先写入白名单规则，再上报 backup.exe 告警，验证不进入 alert_logs。',
    kind: 'primary',
    stageFocus: 'whitelist',
    outcomeHint: '确定性规则优先，命中后 dropped > 0',
    steps: () => [
      {
        label: '创建白名单',
        method: 'POST',
        path: '/api/whitelist',
        body: {
          rule_name: 'demo-backup-whitelist',
          logic: 'OR',
          process_name: 'backup.exe',
          enabled: true,
        },
      },
      {
        label: '上报告警',
        method: 'POST',
        path: '/api/client/events',
        body: {
          host_id: 'host-demo-01',
          events: [
            {
              ...eventBase,
              event_id: `evt-whitelist-${Date.now()}`,
              process_name: 'backup.exe',
              process_path: 'C:/Backup/backup.exe',
              target: 'D:/Backup/customer.xlsx',
              timestamp: nowSeconds(),
            },
          ],
        },
      },
    ],
  },
  {
    id: 'dedup_merge',
    title: '去重窗口合并',
    summary: '同一 host/user/process/type/operation 在窗口内重复上报，验证聚合输出。',
    kind: 'secondary',
    stageFocus: 'dedup',
    outcomeHint: '重复告警被聚合为一个 merge event',
    steps: () => {
      const stamp = nowSeconds()
      const group = `evt-dedup-${Date.now()}`
      return [
        {
          label: '批量上报重复告警',
          method: 'POST',
          path: '/api/client/events',
          body: {
            host_id: 'host-demo-01',
            events: [
              {
                ...eventBase,
                event_id: `${group}-a`,
                file_path: 'C:/Users/Alice/Desktop/customer-a.xlsx',
                file_hash: 'hash-a',
                timestamp: stamp,
              },
              {
                ...eventBase,
                event_id: `${group}-b`,
                file_path: 'C:/Users/Alice/Desktop/customer-b.xlsx',
                file_hash: 'hash-b',
                timestamp: stamp + 20,
              },
            ],
          },
        },
      ]
    },
  },
  {
    id: 'seed_false_positive',
    title: '预置误报模式',
    summary: '写入 false_positive_library，供后续结构化召回命中。',
    kind: 'secondary',
    stageFocus: 'seed',
    outcomeHint: 'scenario_key upsert，重复点击累加 hit_count',
    steps: () => [
      {
        label: '写入误报模式',
        method: 'POST',
        path: '/api/false-positives',
        body: {
          scenario_key: 'customer|upload|dlp-demo-browser.exe|internal-crm.company.com',
          user_id: 'alice',
          sensitive_type: 'customer',
          risk_level: 'low',
          process_name: 'dlp-demo-browser.exe',
          process_path: 'C:/DLPDemo/dlp-demo-browser.exe',
          target: 'internal-crm.company.com',
          operation: 'upload',
          reason: 'normal crm upload from trusted internal workflow',
          hit_count: 1,
        },
      },
    ],
  },
  {
    id: 'confirmed_false_positive',
    title: '召回命中后 Agent 精判',
    summary: '先预置误报模式，再上报完全匹配事件，验证 recall_score 与 Agent 输出。',
    kind: 'primary',
    stageFocus: 'recall_agent',
    outcomeHint: '结构化召回 + ReAct 精判输出可解释误报',
    steps: () => {
      const timestamp = nowSeconds()
      const seedScenario = scenarios.find((s) => s.id === 'seed_false_positive')!
      return [
        ...seedScenario.steps(),
        {
          label: '上报匹配误报模式的告警',
          method: 'POST',
          path: '/api/client/events',
          body: {
            host_id: 'host-demo-01',
            events: [
              {
                ...eventBase,
                event_id: `evt-fp-${Date.now()}`,
                timestamp,
              },
            ],
          },
        },
      ]
    },
  },
  {
    id: 'uncertain_candidate',
    title: '疑似误报',
    summary: '字段部分相似但 target 不同，验证低/中证据场景的展示。',
    kind: 'secondary',
    stageFocus: 'uncertain',
    outcomeHint: '证据不足时最多有限降级，不写误报库',
    steps: () => [
      {
        label: '上报部分相似告警',
        method: 'POST',
        path: '/api/client/events',
        body: {
          host_id: 'host-demo-01',
          events: [
            {
              ...eventBase,
              event_id: `evt-uncertain-${Date.now()}`,
              target: 'partner-crm.example.com',
              timestamp: nowSeconds(),
            },
          ],
        },
      },
    ],
  },
  {
    id: 'empty_recall_agent_judgement',
    title: '空召回 Agent 判断',
    summary: '使用全新业务字段，不预置误报模式，验证空召回仍调用 Agent 且不写误报库。',
    kind: 'secondary',
    stageFocus: 'uncertain',
    outcomeHint: '空召回仍调用 Agent，但不沉淀误报模式',
    steps: () => [
      {
        label: '上报无历史模式告警',
        method: 'POST',
        path: '/api/client/events',
        body: {
          host_id: 'host-demo-02',
          events: [
            {
              ...eventBase,
              event_id: `evt-empty-recall-${Date.now()}`,
              host_id: 'host-demo-02',
              user_id: 'bob',
              file_path: 'C:/Users/Bob/Desktop/legal-contract.pdf',
              file_hash: 'hash-legal-contract',
              sensitive_type: 'legal_contract',
              process_name: 'edge.exe',
              process_path: 'C:/Program Files/Microsoft/Edge/Application/msedge.exe',
              target: 'legal-review.internal',
              operation: 'upload',
              risk_level: 'critical',
              sensitive_file_id: 'file-demo-legal',
              timestamp: nowSeconds(),
            },
          ],
        },
      },
    ],
  },
  {
    id: 'true_alert',
    title: '非误报真实告警',
    summary: '敏感文件上传外部邮箱，验证真实告警进入 alert_logs 且保留风险等级。',
    kind: 'primary',
    stageFocus: 'true_alert',
    outcomeHint: '真实风险保留 critical，不自动误降级',
    steps: () => [
      {
        label: '上报外发告警',
        method: 'POST',
        path: '/api/client/events',
        body: {
          host_id: 'host-demo-01',
          events: [
            {
              ...eventBase,
              event_id: `evt-true-${Date.now()}`,
              target: 'mail.qq.com',
              operation: 'upload',
              risk_level: 'critical',
              timestamp: nowSeconds(),
            },
          ],
        },
      },
    ],
  },
]

function headers(token: string): Record<string, string> {
  const result: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token.trim()) {
    result['Authorization'] = `Bearer ${token}`
  }
  return result
}

async function request(step: ScenarioStep, token: string): Promise<StepResult> {
  const response = await fetch(step.path, {
    method: step.method,
    headers: headers(token),
    body: step.body ? JSON.stringify(step.body) : undefined,
  })
  const text = await response.text()
  let parsed: unknown = text
  try {
    parsed = JSON.parse(text)
  } catch {
    parsed = text
  }
  return {
    label: step.label,
    request: {
      method: step.method,
      path: step.path,
      body: step.body || null,
    },
    status: response.status,
    response: parsed,
  }
}

function formatRequestError(result: StepResult): string {
  const responseText =
    typeof result.response === 'string' ? result.response : JSON.stringify(result.response)
  return `${result.request.method} ${result.request.path} returned HTTP ${result.status}: ${responseText}`
}

export function useScenarios() {
  const [activeScenario, setActiveScenario] = useState<Scenario | null>(null)
  const [steps, setSteps] = useState<ScenarioStep[]>([])
  const [results, setResults] = useState<StepResult[]>([])
  const [runState, setRunState] = useState<RunState>('idle')
  const [error, setError] = useState<string | null>(null)

  const runScenario = useCallback(async (scenario: Scenario, token: string, onComplete?: () => void) => {
    setActiveScenario(scenario)
    setRunState('running')
    setError(null)
    const generatedSteps = scenario.steps()
    setSteps(generatedSteps)
    setResults([])

    try {
      const stepResults: StepResult[] = []
      for (const step of generatedSteps) {
        const result = await request(step, token)
        stepResults.push(result)
        setResults([...stepResults])
        if (result.status >= 400) {
          throw new Error(formatRequestError(result))
        }
      }
      setRunState('ok')
      onComplete?.()
    } catch (err) {
      setRunState('error')
      setError(err instanceof Error ? err.stack || err.message : String(err))
    }
  }, [])

  return {
    scenarios,
    activeScenario,
    steps,
    results,
    runState,
    error,
    runScenario,
  }
}

import { useState } from 'react'
import { AlertCircle, CheckCircle2, ChevronDown, ChevronRight, Loader2, Terminal } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import type { ScenarioStep, StepResult, RunState, AlertRecord } from '@/types'

interface ScenarioRunnerProps {
  steps: ScenarioStep[]
  results: StepResult[]
  currentAlerts: AlertRecord[]
  runState: RunState
  error: string | null
}

const methodColor: Record<string, string> = {
  GET: 'text-[#34D399] border-[#22C55E]/30 bg-[#22C55E]/8',
  POST: 'text-[#60A5FA] border-[#3B82F6]/30 bg-[#3B82F6]/8',
  PUT: 'text-[#FBBF24] border-[#F59E0B]/30 bg-[#F59E0B]/8',
  DELETE: 'text-[#F87171] border-[#EF4444]/30 bg-[#EF4444]/8',
}

function json(value: unknown) {
  return JSON.stringify(value, null, 2)
}

function eventCount(step: ScenarioStep) {
  const events = step.body?.events
  return Array.isArray(events) ? events.length : undefined
}

function resultForStep(results: StepResult[], index: number) {
  return results[index]
}

function DedupHighlight({ alerts }: { alerts: AlertRecord[] }) {
  const merged = alerts.find((alert) => alert.is_merge_event)
  if (!merged) return null
  const files = Array.isArray(merged.files) ? merged.files : []

  return (
    <div className="scenario-merge-highlight">
      <div>
        <span>merge event_id</span>
        <strong>{merged.event_id}</strong>
      </div>
      <div>
        <span>file_count</span>
        <strong>{merged.file_count || files.length}</strong>
      </div>
      <div>
        <span>time_range</span>
        <strong>{String(merged.time_range || '-')}</strong>
      </div>
      <div>
        <span>duration</span>
        <strong>{String(merged.duration || '-')}</strong>
      </div>
      {files.length > 0 && (
        <pre className="json-view json-wrap scenario-merge-files">{json(files)}</pre>
      )}
    </div>
  )
}

function StepCard({
  step,
  result,
  index,
}: {
  step: ScenarioStep
  result?: StepResult
  index: number
}) {
  const [open, setOpen] = useState(index === 0)
  const count = eventCount(step)

  return (
    <div className="scenario-step-card">
      <button type="button" className="scenario-step-summary" onClick={() => setOpen((value) => !value)}>
        {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        <span className="text-[11px] font-mono text-[#6B7280]">#{index + 1}</span>
        <Badge className={cn('text-[10px] font-mono border', methodColor[step.method] || '')}>
          {step.method}
        </Badge>
        <span className="scenario-step-title">{step.label}</span>
        {count ? <span className="scenario-step-pill">{count} events</span> : null}
        {result ? (
          <Badge variant={result.status >= 400 ? 'destructive' : 'success'} className="text-[10px] px-1.5 py-0">
            {result.status}
          </Badge>
        ) : (
          <span className="scenario-step-pending">pending</span>
        )}
      </button>

      {open && (
        <div className="scenario-step-detail">
          <div className="scenario-step-path">{step.path}</div>
          <Tabs defaultValue="request" className="w-full">
            <TabsList className="h-8 bg-[#0F1116]">
              <TabsTrigger value="request" className="h-6 px-2 text-xs">
                Request
              </TabsTrigger>
              <TabsTrigger value="response" className="h-6 px-2 text-xs">
                Response
              </TabsTrigger>
            </TabsList>
            <TabsContent value="request">
              <pre className="json-view json-wrap scenario-json-block">
                {json({
                  method: step.method,
                  path: step.path,
                  body: step.body || null,
                })}
              </pre>
            </TabsContent>
            <TabsContent value="response">
              <pre className="json-view json-wrap scenario-json-block">
                {result ? json({ status: result.status, body: result.response }) : 'Waiting for response...'}
              </pre>
            </TabsContent>
          </Tabs>
        </div>
      )}
    </div>
  )
}

export function ScenarioRunner({ steps, results, currentAlerts, runState, error }: ScenarioRunnerProps) {
  const hasSteps = steps.length > 0

  return (
    <Card className="mb-4 bg-[#1A1D23] border-[#2D3748] rounded-lg">
      <CardHeader className="py-3 px-4 flex-row items-center justify-between space-y-0">
        <div className="flex items-center gap-2">
          {runState === 'running' && <Loader2 className="w-4 h-4 text-[#F59E0B] animate-spin" />}
          {runState === 'ok' && <CheckCircle2 className="w-4 h-4 text-[#22C55E]" />}
          {runState === 'error' && <AlertCircle className="w-4 h-4 text-[#EF4444]" />}
          {runState === 'idle' && <Terminal className="w-4 h-4 text-[#6B7280]" />}
          <p className="text-sm font-semibold text-[#F9FAFB]">本次运行证据</p>
        </div>
        <span className="text-[11px] font-mono text-[#9CA3AF]">
          {results.length}/{steps.length} steps
        </span>
      </CardHeader>
      <Separator className="bg-[#2D3748]" />
      <CardContent className="p-4">
        {!hasSteps ? (
          <div className="flex flex-col items-center justify-center py-10 text-[#6B7280]">
            <Terminal className="w-8 h-8 mb-2 opacity-25" />
            <p className="text-xs">点击主演示按钮后显示完整 Request / Response</p>
          </div>
        ) : (
          <ScrollArea className="h-[420px] pr-3">
            <div className="space-y-3">
              <DedupHighlight alerts={currentAlerts} />
              {steps.map((step, index) => (
                <StepCard key={`${step.path}-${index}`} step={step} result={resultForStep(results, index)} index={index} />
              ))}
              {error && (
                <div className="p-3 rounded-md bg-[#EF4444]/8 border border-[#EF4444]/20">
                  <p className="text-xs text-[#EF4444] font-mono whitespace-pre-wrap break-all">{error}</p>
                </div>
              )}
            </div>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  )
}

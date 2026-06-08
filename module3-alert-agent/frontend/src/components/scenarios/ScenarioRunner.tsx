import { AlertCircle, CheckCircle2, Loader2, Terminal } from 'lucide-react'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import type { ScenarioStep, StepResult, RunState } from '@/types'

interface ScenarioRunnerProps {
  steps: ScenarioStep[]
  results: StepResult[]
  runState: RunState
  error: string | null
}

const methodColor: Record<string, string> = {
  GET:    'text-[#34D399] border-[#22C55E]/30 bg-[#22C55E]/8',
  POST:   'text-[#60A5FA] border-[#3B82F6]/30 bg-[#3B82F6]/8',
  PUT:    'text-[#FBBF24] border-[#F59E0B]/30 bg-[#F59E0B]/8',
  DELETE: 'text-[#F87171] border-[#EF4444]/30 bg-[#EF4444]/8',
}

export function ScenarioRunner({ steps, results, runState, error }: ScenarioRunnerProps) {
  const hasSteps = steps.length > 0

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
      {/* ---- 场景输入 ---- */}
      <Card className="bg-[#1A1D23] border-[#2D3748] rounded-lg">
        <CardHeader className="py-3 px-4 flex-row items-center justify-between space-y-0">
          <div className="flex items-center gap-2">
            <Terminal className="w-4 h-4 text-[#6B7280]" />
            <p className="text-sm font-semibold text-[#F9FAFB]">场景输入</p>
          </div>
          {runState === 'idle' && (
            <span className="text-[11px] text-[#6B7280]">等待触发</span>
          )}
          {runState === 'running' && (
            <span className="text-[11px] text-[#F59E0B] animate-pulse">执行中</span>
          )}
          {runState === 'ok' && (
            <span className="text-[11px] text-[#22C55E]">已完成</span>
          )}
          {runState === 'error' && (
            <span className="text-[11px] text-[#EF4444]">异常</span>
          )}
        </CardHeader>
        <Separator className="bg-[#2D3748]" />
        <CardContent className="p-4">
          {!hasSteps ? (
            <div className="flex flex-col items-center justify-center py-10 text-[#6B7280]">
              <Terminal className="w-8 h-8 mb-2 opacity-25" />
              <p className="text-xs">点击左侧场景按钮后显示请求步骤</p>
            </div>
          ) : (
            <ScrollArea className="h-[280px]">
              <div className="space-y-2">
                {steps.map((step, i) => (
                  <div key={i} className="flex items-start gap-3 p-3 rounded-md bg-[#16181D] border border-[#252A34]">
                    <span className="text-[11px] font-mono text-[#6B7280] mt-0.5 shrink-0">
                      #{i + 1}
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge className={cn('text-[10px] font-mono border', methodColor[step.method] || '')}>
                          {step.method}
                        </Badge>
                        <span className="text-sm font-medium text-[#D1D5DB] truncate">{step.label}</span>
                      </div>
                      <p className="text-[11px] font-mono text-[#6B7280] truncate">{step.path}</p>
                    </div>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </CardContent>
      </Card>

      {/* ---- 执行输出 ---- */}
      <Card className="bg-[#1A1D23] border-[#2D3748] rounded-lg">
        <CardHeader className="py-3 px-4 flex-row items-center justify-between space-y-0">
          <div className="flex items-center gap-2">
            {runState === 'running' && <Loader2 className="w-4 h-4 text-[#F59E0B] animate-spin" />}
            {runState === 'ok' && <CheckCircle2 className="w-4 h-4 text-[#22C55E]" />}
            {runState === 'error' && <AlertCircle className="w-4 h-4 text-[#EF4444]" />}
            {runState === 'idle' && <Terminal className="w-4 h-4 text-[#6B7280]" />}
            <p className="text-sm font-semibold text-[#F9FAFB]">执行输出</p>
          </div>
          {results.length > 0 && (
            <span className="text-[11px] font-mono text-[#9CA3AF]">{results.length} 条响应</span>
          )}
        </CardHeader>
        <Separator className="bg-[#2D3748]" />
        <CardContent className="p-4">
          {error ? (
            <div className="p-3 rounded-md bg-[#EF4444]/8 border border-[#EF4444]/20">
              <p className="text-xs text-[#EF4444] font-mono whitespace-pre-wrap break-all">{error}</p>
            </div>
          ) : results.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-10 text-[#6B7280]">
              <CheckCircle2 className="w-8 h-8 mb-2 opacity-25" />
              <p className="text-xs">点击左侧按钮后显示完整请求与响应</p>
            </div>
          ) : (
            <ScrollArea className="h-[280px]">
              <div className="space-y-2">
                {results.map((result, i) => (
                  <div key={i} className="rounded-md border border-[#252A34] overflow-hidden">
                    <div className="flex items-center gap-2 px-3 py-2 bg-[#16181D] border-b border-[#252A34]">
                      <span className="text-[11px] font-mono text-[#6B7280]">#{i + 1}</span>
                      <Badge className={cn('text-[10px] font-mono border', methodColor[result.request.method] || '')}>
                        {result.request.method}
                      </Badge>
                      <span className="text-xs font-medium text-[#D1D5DB] truncate flex-1">{result.label}</span>
                      <Badge
                        variant={result.status >= 400 ? 'destructive' : 'success'}
                        className="text-[10px] px-1.5 py-0"
                      >
                        {result.status}
                      </Badge>
                    </div>
                    <pre className="json-view p-3 text-xs max-h-[180px] overflow-auto bg-[#0F1116]">
                      {JSON.stringify(result.response, null, 2)}
                    </pre>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

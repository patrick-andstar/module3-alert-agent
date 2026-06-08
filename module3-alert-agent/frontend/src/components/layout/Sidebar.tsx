import { useMemo } from 'react'
import {
  Shield, ShieldCheck, Layers, Database,
  Search, FileQuestion, AlertTriangle, AlertCircle,
  Activity
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { Scenario, RunState } from '@/types'

/* ---- Scenario icon map ---- */
const scenarioIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  whitelist_drop: ShieldCheck,
  dedup_merge: Layers,
  seed_false_positive: Database,
  confirmed_false_positive: Search,
  uncertain_candidate: FileQuestion,
  empty_recall_agent_judgement: AlertTriangle,
  true_alert: AlertCircle,
}

interface SidebarProps {
  scenarios: Scenario[]
  activeScenarioId: string | null
  runState: RunState
  token: string
  onTokenChange: (token: string) => void
  onScenarioClick: (scenario: Scenario) => void
  className?: string
}

export function Sidebar({
  scenarios,
  activeScenarioId,
  runState,
  token,
  onTokenChange,
  onScenarioClick,
  className,
}: SidebarProps) {
  return (
    <aside className={cn(
      'flex flex-col h-full bg-[#0A0C10] border-r border-[#2D3748]',
      className
    )}>
      {/* Brand */}
      <div className="flex items-center gap-3 px-4 py-5">
        <div className="flex items-center justify-center w-9 h-9 rounded-md bg-[#3B82F6]/10">
          <Shield className="w-5 h-5 text-[#3B82F6]" />
        </div>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-[#F9FAFB] tracking-tight">DLP Alert Agent</p>
          <p className="text-[11px] text-[#9CA3AF] mt-0.5">模块三 · 场景触发台</p>
        </div>
      </div>

      <Separator className="bg-[#2D3748]" />

      {/* Token */}
      <div className="px-4 py-4">
        <label className="text-[11px] font-medium text-[#6B7280] uppercase tracking-wider mb-1.5 block">
          Admin Token
        </label>
        <Input
          value={token}
          onChange={(e) => onTokenChange(e.target.value)}
          placeholder="未配置可留空"
          className="h-8 text-xs font-mono bg-[#1A1D23] border-[#2D3748] text-[#D1D5DB] placeholder:text-[#6B7280] focus-visible:ring-[#3B82F6]/30"
          autoComplete="off"
        />
      </div>

      <Separator className="bg-[#2D3748]" />

      {/* Nav */}
      <div className="px-3 py-3 flex-1 flex flex-col min-h-0">
        <p className="px-2 mb-2 text-[11px] font-medium text-[#6B7280] uppercase tracking-wider">
          业务场景
        </p>
        <ScrollArea className="flex-1">
          <div className="space-y-0.5 pr-1">
            {scenarios.map((scenario) => {
              const isActive = activeScenarioId === scenario.id
              const isRunning = isActive && runState === 'running'
              const Icon = scenarioIcons[scenario.id] || Activity

              return (
                <button
                  key={scenario.id}
                  onClick={() => onScenarioClick(scenario)}
                  className={cn(
                    'w-full flex items-start gap-3 px-3 py-2.5 rounded-md transition-all duration-150 text-left group',
                    isActive
                      ? 'bg-[#3B82F6]/8 border border-[#3B82F6]/20'
                      : 'border border-transparent hover:bg-[#252A34]/60 hover:border-[#2D3748]'
                  )}
                >
                  <Icon className={cn(
                    'w-4 h-4 mt-0.5 shrink-0',
                    isActive ? 'text-[#3B82F6]' : 'text-[#6B7280] group-hover:text-[#9CA3AF]'
                  )} />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className={cn(
                        'text-sm font-medium truncate',
                        isActive ? 'text-[#F9FAFB]' : 'text-[#D1D5DB]'
                      )}>
                        {scenario.title}
                      </span>
                      {isRunning && (
                        <Badge variant="warning" className="shrink-0 text-[10px] px-1.5 py-0 h-4 leading-none">
                          执行中
                        </Badge>
                      )}
                      {isActive && runState === 'ok' && (
                        <Badge variant="success" className="shrink-0 text-[10px] px-1.5 py-0 h-4 leading-none">
                          完成
                        </Badge>
                      )}
                    </div>
                    <p className="text-[11px] text-[#6B7280] mt-0.5 line-clamp-2 leading-relaxed">
                      {scenario.summary}
                    </p>
                  </div>
                </button>
              )
            })}
          </div>
        </ScrollArea>
      </div>
    </aside>
  )
}

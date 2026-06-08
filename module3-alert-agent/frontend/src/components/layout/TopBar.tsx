import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { RunState } from '@/types'

interface TopBarProps {
  activeTitle: string
  runState: RunState
  scenarioBadge: string
  loading: boolean
  onRefresh: () => void
}

const stateConfig: Record<RunState, { label: string; dot: string }> = {
  idle:    { label: '就绪', dot: 'bg-[#6B7280]' },
  running: { label: '执行中', dot: 'bg-[#F59E0B] animate-pulse' },
  ok:      { label: '完成', dot: 'bg-[#22C55E]' },
  error:   { label: '异常', dot: 'bg-[#EF4444]' },
}

export function TopBar({ activeTitle, runState, scenarioBadge, loading, onRefresh }: TopBarProps) {
  const state = stateConfig[runState]

  return (
    <header className="flex items-center justify-between gap-4 mb-6">
      <div className="min-w-0">
        <p className="text-[11px] text-[#6B7280] uppercase tracking-wider font-medium mb-1">
          MySQL-backed flow
        </p>
        <div className="flex items-center gap-3">
          <h2 className="text-xl font-semibold text-[#F9FAFB] tracking-tight truncate">
            {activeTitle || '选择一个业务场景'}
          </h2>
          {scenarioBadge && scenarioBadge !== '等待触发' && (
            <Badge variant="outline" className="font-mono text-[11px] shrink-0 border-[#2D3748] text-[#9CA3AF]">
              {scenarioBadge}
            </Badge>
          )}
        </div>
      </div>
      <div className="flex items-center gap-3 shrink-0">
        <span className="inline-flex items-center gap-1.5 text-[12px] text-[#9CA3AF]">
          <span className={cn('w-1.5 h-1.5 rounded-full inline-block', state.dot)} />
          {state.label}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={loading}
          className="gap-2 h-8 text-xs bg-[#1A1D23] border-[#2D3748] text-[#D1D5DB] hover:bg-[#252A34] hover:border-[#3B4A5E]"
        >
          <RefreshCw className={cn('w-3.5 h-3.5', loading && 'animate-spin')} />
          刷新记录
        </Button>
      </div>
    </header>
  )
}

import { Activity, Database, ShieldCheck } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { DashboardStats } from '@/types'

interface StatsCardsProps {
  stats: DashboardStats
  loading: boolean
}

const cards = [
  { key: 'alertCount' as const, label: '告警查询记录', desc: 'POST /api/alerts/query', icon: Activity, color: 'blue' },
  { key: 'fpCount' as const, label: '误报库记录', desc: 'GET /api/false-positives', icon: Database, color: 'amber' },
  { key: 'whitelistCount' as const, label: '白名单记录', desc: 'GET /api/whitelist', icon: ShieldCheck, color: 'emerald' },
] as const

const colors = {
  blue:    { num: 'text-[#60A5FA]', icon: 'text-[#3B82F6]/20', glow: 'rgba(59,130,246,0.4)' },
  amber:   { num: 'text-[#FBBF24]', icon: 'text-[#F59E0B]/20', glow: 'rgba(245,158,11,0.4)' },
  emerald: { num: 'text-[#34D399]', icon: 'text-[#22C55E]/20', glow: 'rgba(34,197,94,0.4)' },
}

export function StatsCards({ stats, loading }: StatsCardsProps) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-4">
      {cards.map(({ key, label, desc, icon: Icon, color }) => {
        const c = colors[color]
        return (
          <div
            key={key}
            className="glass-card rounded-lg p-4 stat-glow transition-all duration-200 hover:border-[#3B4A5E]"
          >
            <div className="flex items-center justify-between mb-3">
              <p className="text-[11px] font-medium text-[#9CA3AF] uppercase tracking-wider">{label}</p>
              <Icon className={cn('w-4 h-4', c.icon)} />
            </div>

            {loading ? (
              <div className="space-y-2">
                <div className="h-8 w-20 rounded skeleton-pulse bg-[#252A34]" />
                <div className="h-3 w-28 rounded skeleton-pulse bg-[#252A34]" />
              </div>
            ) : (
              <>
                <p className={cn('text-[28px] font-bold font-mono tabular-nums leading-none', c.num)}>
                  {stats[key]}
                </p>
                <p className="text-[11px] text-[#6B7280] mt-1.5">{desc}</p>
              </>
            )}
          </div>
        )
      })}
    </div>
  )
}

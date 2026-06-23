import { Activity } from 'lucide-react'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import type { AlertRecord } from '@/types'

interface AlertLogsTableProps {
  data: AlertRecord[]
  loading: boolean
  onSelect?: (row: AlertRecord) => void
}

const riskStyle: Record<string, string> = {
  critical: 'text-[#F87171] border-[#EF4444]/30 bg-[#EF4444]/8',
  high:     'text-[#FB923C] border-[#F97316]/30 bg-[#F97316]/8',
  medium:   'text-[#FBBF24] border-[#F59E0B]/30 bg-[#F59E0B]/8',
  low:      'text-[#34D399] border-[#22C55E]/30 bg-[#22C55E]/8',
  info:     'text-[#9CA3AF] border-[#6B7280]/30 bg-[#6B7280]/8',
}

const verdictStyle: Record<string, string> = {
  false_positive: 'text-[#34D399] border-[#22C55E]/30 bg-[#22C55E]/8',
  true_alert:     'text-[#F87171] border-[#EF4444]/30 bg-[#EF4444]/8',
  uncertain:      'text-[#FBBF24] border-[#F59E0B]/30 bg-[#F59E0B]/8',
}

export function AlertLogsTable({ data, loading, onSelect }: AlertLogsTableProps) {
  return (
    <div className="bg-[#1A1D23] border border-[#2D3748] rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-[#2D3748]">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-[#60A5FA]" />
          <p className="text-sm font-semibold text-[#F9FAFB]">alert_logs</p>
        </div>
        <span className="text-[10px] font-mono text-[#6B7280]">POST /api/alerts/query</span>
      </div>
      <ScrollArea className="h-[320px]">
        {loading ? (
          <div className="p-6 text-center text-xs text-[#6B7280]">加载中...</div>
        ) : data.length === 0 ? (
          <div className="p-6 text-center text-xs text-[#6B7280]">暂无数据</div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="border-[#2D3748] bg-[#16181D] hover:bg-[#16181D]">
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">Event ID</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">用户</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">类型</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">风险</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">Agent 判断</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((row) => (
                <TableRow
                  key={row.event_id}
                  className="border-[#252A34] hover:bg-[#252A34]/50 transition-colors cursor-pointer"
                  onClick={() => onSelect?.(row)}
                >
                  <TableCell className="font-mono text-[11px] text-[#D1D5DB] py-2.5 max-w-[160px] truncate">
                    {row.event_id}
                  </TableCell>
                  <TableCell className="text-xs text-[#D1D5DB] py-2.5">{row.user_id}</TableCell>
                  <TableCell className="py-2.5">
                    <Badge className="text-[10px] border-[#2D3748] bg-[#252A34] text-[#9CA3AF] font-normal">
                      {row.sensitive_type}
                    </Badge>
                  </TableCell>
                  <TableCell className="py-2.5">
                    <span className={cn('inline-flex text-[10px] font-medium px-2 py-0.5 rounded border', riskStyle[row.risk_level] || '')}>
                      {row.risk_level}
                    </span>
                  </TableCell>
                  <TableCell className="py-2.5">
                    {row.agent_verdict ? (
                      <span className={cn('inline-flex text-[10px] font-medium px-2 py-0.5 rounded border', verdictStyle[row.agent_verdict] || '')}>
                        {row.agent_verdict}
                      </span>
                    ) : (
                      <span className="text-[11px] text-[#6B7280]">—</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </ScrollArea>
    </div>
  )
}

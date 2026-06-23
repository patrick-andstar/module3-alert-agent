import { ShieldCheck } from 'lucide-react'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import type { WhitelistRule } from '@/types'

interface WhitelistTableProps {
  data: WhitelistRule[]
  loading: boolean
  onSelect?: (row: WhitelistRule) => void
}

export function WhitelistTable({ data, loading, onSelect }: WhitelistTableProps) {
  return (
    <div className="bg-[#1A1D23] border border-[#2D3748] rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-[#2D3748]">
        <div className="flex items-center gap-2">
          <ShieldCheck className="w-4 h-4 text-[#34D399]" />
          <p className="text-sm font-semibold text-[#F9FAFB]">whitelist_rules</p>
        </div>
        <span className="text-[10px] font-mono text-[#6B7280]">GET /api/whitelist</span>
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
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">规则名</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">逻辑</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">进程</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">用户</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((row) => (
                <TableRow
                  key={row.id}
                  className="border-[#252A34] hover:bg-[#252A34]/50 transition-colors cursor-pointer"
                  onClick={() => onSelect?.(row)}
                >
                  <TableCell className="text-xs text-[#D1D5DB] py-2.5 font-medium">{row.rule_name}</TableCell>
                  <TableCell className="py-2.5">
                    <span className={cn(
                      'inline-flex text-[10px] font-mono font-medium px-2 py-0.5 rounded border',
                      row.logic === 'AND'
                        ? 'border-[#3B82F6]/20 text-[#60A5FA] bg-[#3B82F6]/8'
                        : 'border-[#A78BFA]/20 text-[#A78BFA] bg-[#A78BFA]/8'
                    )}>
                      {row.logic}
                    </span>
                  </TableCell>
                  <TableCell className="text-xs text-[#D1D5DB] py-2.5 font-mono max-w-[100px] truncate">
                    {row.process_name || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-[#D1D5DB] py-2.5">
                    {row.user_id || '—'}
                  </TableCell>
                  <TableCell className="py-2.5">
                    <span className={cn(
                      'inline-flex text-[10px] font-medium px-2 py-0.5 rounded border',
                      row.enabled
                        ? 'text-[#34D399] border-[#22C55E]/20 bg-[#22C55E]/8'
                        : 'text-[#9CA3AF] border-[#6B7280]/20 bg-[#6B7280]/8'
                    )}>
                      {row.enabled ? '启用' : '禁用'}
                    </span>
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

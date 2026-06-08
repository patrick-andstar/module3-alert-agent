import { Database } from 'lucide-react'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { FalsePositiveRecord } from '@/types'

interface FalsePositivesTableProps {
  data: FalsePositiveRecord[]
  loading: boolean
}

export function FalsePositivesTable({ data, loading }: FalsePositivesTableProps) {
  return (
    <div className="bg-[#1A1D23] border border-[#2D3748] rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-[#2D3748]">
        <div className="flex items-center gap-2">
          <Database className="w-4 h-4 text-[#FBBF24]" />
          <p className="text-sm font-semibold text-[#F9FAFB]">false_positive_library</p>
        </div>
        <span className="text-[10px] font-mono text-[#6B7280]">GET /api/false-positives</span>
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
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">Scenario Key</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">类型</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">进程</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">操作</TableHead>
                <TableHead className="text-[11px] text-[#9CA3AF] font-medium py-2.5">命中</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((row) => (
                <TableRow key={row.id} className="border-[#252A34] hover:bg-[#252A34]/50 transition-colors">
                  <TableCell className="font-mono text-[11px] text-[#D1D5DB] py-2.5 max-w-[180px] truncate">
                    {row.scenario_key}
                  </TableCell>
                  <TableCell className="py-2.5">
                    <Badge className="text-[10px] border-[#2D3748] bg-[#252A34] text-[#9CA3AF] font-normal">
                      {row.sensitive_type}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-[#D1D5DB] py-2.5 font-mono max-w-[100px] truncate">
                    {row.process_name}
                  </TableCell>
                  <TableCell className="text-xs text-[#D1D5DB] py-2.5">{row.operation}</TableCell>
                  <TableCell className="py-2.5">
                    <span className="inline-flex text-[10px] font-medium px-2 py-0.5 rounded border border-[#F59E0B]/20 text-[#FBBF24] bg-[#F59E0B]/8">
                      {row.hit_count}
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

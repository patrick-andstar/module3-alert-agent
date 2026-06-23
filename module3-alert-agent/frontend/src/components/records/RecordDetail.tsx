import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { AlertRecord, FalsePositiveRecord, WhitelistRule } from '@/types'

type DetailRecord =
  | { kind: 'alert'; row: AlertRecord }
  | { kind: 'false_positive'; row: FalsePositiveRecord }
  | { kind: 'whitelist'; row: WhitelistRule }

interface RecordDetailProps {
  record: DetailRecord | null
  onClose: () => void
}

function value(value: unknown) {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  return String(value)
}

function Field({ label, children }: { label: string; children: unknown }) {
  return (
    <div className="record-detail-field">
      <span>{label}</span>
      <strong>{value(children)}</strong>
    </div>
  )
}

function AlertBusiness({ row }: { row: AlertRecord }) {
  const files = Array.isArray(row.files) ? row.files : []
  return (
    <div className="record-detail-grid">
      <Field label="event_id">{row.event_id}</Field>
      <Field label="risk">{`${value(row.old_risk_level)} -> ${value(row.risk_level)}`}</Field>
      <Field label="agent_verdict">{row.agent_verdict}</Field>
      <Field label="confidence">{row.agent_confidence}</Field>
      <Field label="recall_score">{row.recall_score}</Field>
      <Field label="is_merge_event">{row.is_merge_event}</Field>
      <Field label="file_count">{row.file_count}</Field>
      <Field label="false_positive_reason">{row.false_positive_reason}</Field>
      <div className="record-detail-wide">
        <span>agent_explanation</span>
        <p>{value(row.agent_explanation)}</p>
      </div>
      {files.length > 0 && (
        <div className="record-detail-wide">
          <span>files[]</span>
          <pre className="json-view json-wrap">{JSON.stringify(files, null, 2)}</pre>
        </div>
      )}
    </div>
  )
}

function FalsePositiveBusiness({ row }: { row: FalsePositiveRecord }) {
  return (
    <div className="record-detail-grid">
      <Field label="scenario_key">{row.scenario_key}</Field>
      <Field label="sensitive_type">{row.sensitive_type}</Field>
      <Field label="operation">{row.operation}</Field>
      <Field label="process_name">{row.process_name}</Field>
      <Field label="process_path">{row.process_path}</Field>
      <Field label="target">{row.target}</Field>
      <Field label="hit_count">{row.hit_count}</Field>
      <Field label="last_seen_at">{row.last_seen_at}</Field>
      <Field label="expired_at">{row.expired_at}</Field>
      <div className="record-detail-wide">
        <span>reason</span>
        <p>{value(row.reason)}</p>
      </div>
    </div>
  )
}

function WhitelistBusiness({ row }: { row: WhitelistRule }) {
  return (
    <div className="record-detail-grid">
      <Field label="rule_name">{row.rule_name}</Field>
      <Field label="logic">{row.logic}</Field>
      <Field label="enabled">{row.enabled}</Field>
      <Field label="process_name">{row.process_name}</Field>
      <Field label="user_id">{row.user_id}</Field>
      <Field label="file_path_pattern">{row.file_path_pattern}</Field>
      <Field label="time_window_start">{row.time_window_start}</Field>
      <Field label="time_window_end">{row.time_window_end}</Field>
    </div>
  )
}

function title(record: DetailRecord | null) {
  if (!record) return 'Record detail'
  if (record.kind === 'alert') return `alert_logs: ${record.row.event_id}`
  if (record.kind === 'false_positive') return `false_positive_library: ${record.row.scenario_key}`
  return `whitelist_rules: ${record.row.rule_name}`
}

export function RecordDetail({ record, onClose }: RecordDetailProps) {
  return (
    <Sheet open={!!record} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="record-detail-sheet overflow-y-auto bg-[#11151B] border-[#2D3748] text-[#F9FAFB] sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle className="text-[#F9FAFB] break-words">{title(record)}</SheetTitle>
          <SheetDescription>业务化详情 + 原始 JSON</SheetDescription>
        </SheetHeader>

        {record && (
          <Tabs defaultValue="business" className="mt-4">
            <TabsList className="bg-[#0F1116]">
              <TabsTrigger value="business">业务详情</TabsTrigger>
              <TabsTrigger value="json">原始 JSON</TabsTrigger>
            </TabsList>
            <TabsContent value="business">
              {record.kind === 'alert' && <AlertBusiness row={record.row} />}
              {record.kind === 'false_positive' && <FalsePositiveBusiness row={record.row} />}
              {record.kind === 'whitelist' && <WhitelistBusiness row={record.row} />}
            </TabsContent>
            <TabsContent value="json">
              <pre className="json-view json-wrap record-detail-json">
                {JSON.stringify(record.row, null, 2)}
              </pre>
            </TabsContent>
          </Tabs>
        )}
      </SheetContent>
    </Sheet>
  )
}

export type { DetailRecord }

# Demo Theater Run Isolation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the approved Demo Theater run isolation and evidence improvements without changing the database schema or recall scoring.

**Architecture:** Backend deduplication will create a real merged `event_id` using the run ID encoded in raw event IDs. The frontend will generate run IDs, execute isolated scenario steps, query current-run records, and render expandable evidence plus row details.

**Tech Stack:** Go 1.21, CloudWeGo Hertz, React 18, TypeScript, Vite, Tailwind CSS.

---

### File Map

- Modify `module3-alert-agent/internal/pipeline/dedup.go`: generate merged event IDs and preserve merged file evidence.
- Modify `module3-alert-agent/internal/pipeline/dedup_test.go`: verify merged event ID behavior.
- Modify `module3-alert-agent/frontend/src/types/index.ts`: add run-aware scenario/result/detail types.
- Modify `module3-alert-agent/frontend/src/hooks/useScenarios.ts`: generate `run_id`, promote dedup scenario, add query steps, and return current-run alerts.
- Modify `module3-alert-agent/frontend/src/components/scenarios/ScenarioRunner.tsx`: render expandable request/response evidence with wrapping JSON and dedup highlights.
- Modify `module3-alert-agent/frontend/src/components/records/*.tsx`: make table rows clickable.
- Add `module3-alert-agent/frontend/src/components/records/RecordDetail.tsx`: shared business-detail plus raw JSON modal.
- Modify `module3-alert-agent/frontend/src/App.tsx`: remove historical fallback for current conclusion, use current-run records, promote dedup button, and wire row details.
- Modify `module3-alert-agent/frontend/src/index.css`: add wrapping JSON and detail/evidence styles.

### Task 1: Backend Merged Event ID

- [ ] Add a failing Go test in `internal/pipeline/dedup_test.go` asserting merged events use `merge-<run_id>-001`.
- [ ] Update `internal/pipeline/dedup.go` so `mergeGroup` derives `run_id` from raw event IDs shaped as `evt-<run_id>-NN`.
- [ ] Run `go test ./internal/pipeline`.

### Task 2: Run-Aware Scenario Hook

- [ ] Extend frontend types with `RunContext`, current run alerts, and optional query step metadata.
- [ ] Update `useScenarios.ts` so each scenario creates a fresh run ID and all generated event IDs include it.
- [ ] Promote dedup into the primary scenario set by keeping its scenario ID available to `App`.
- [ ] Add a final `/api/alerts/query` step for scenarios that should display alert evidence.

### Task 3: Evidence UI

- [ ] Replace the flat Step output cards with expandable Step cards.
- [ ] Show request and response panes for every step.
- [ ] Add JSON wrapping classes and highlight dedup merged alert fields.

### Task 4: Current Conclusion Isolation

- [ ] Remove `alerts[0]` fallback in `App.tsx`.
- [ ] Render empty state before the first run.
- [ ] Render current conclusion from current-run alerts or accepted/dropped response for whitelist.

### Task 5: Row Detail Views

- [ ] Add `RecordDetail.tsx`.
- [ ] Make `AlertLogsTable`, `FalsePositivesTable`, and `WhitelistTable` accept `onSelect`.
- [ ] Wire selected row state in `App.tsx`.

### Task 6: Verification

- [ ] Run `go test ./...`.
- [ ] Run `npm run typecheck`.
- [ ] Run `npm run build`.
- [ ] Fix any regressions.

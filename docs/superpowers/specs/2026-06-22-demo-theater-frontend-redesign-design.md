# Demo Theater Frontend Redesign Design

Date: 2026-06-22
Project: Module 3 Alert Agent
Scope: frontend presentation layer only

## Goal

Redesign the Module 3 frontend into a 4-6 minute live-demo cockpit for the report tomorrow. The main message is:

> Module 3 is not a normal log page. It is an explainable false-positive governance pipeline: deterministic filtering first, deduplication second, Eino ReAct Agent analysis third, then queryable and explainable alert output for Module 4.

The redesign must improve visual quality and demo clarity while preserving the existing backend, API contracts, MySQL behavior, Eino Agent flow, and scenario execution logic.

## Non-Goals

- Do not change backend API behavior.
- Do not change MySQL schemas.
- Do not change Eino/Ark Agent runtime behavior.
- Do not introduce new external runtime dependencies.
- Do not use fake frontend-only results for the three main demo scenarios.
- Do not store real API keys, database passwords, or tokens in files.

## Approved Implementation Approach

Use approach 2: rebuild the frontend page composition and add Demo Theater components while reusing existing hooks and real APIs.

Keep or adapt:

- React + Vite frontend stack.
- `useScenarios` and `useRecords` as the real API integration layer.
- Existing API endpoints under `/api/*`.
- Existing backend static serving path.
- Existing detailed tables where useful.

Replace the primary visual composition in `App.tsx` with a Demo Theater layout.

Preserve E2E smoke anchors unless the smoke script is intentionally updated in the same implementation:

- The served frontend HTML or bundle must still contain `DLP Alert Agent Console`.
- The bundle must still contain scenario ids checked by `tools/e2e-smoke.ps1`: `whitelist_drop`, `dedup_merge`, `seed_false_positive`, `confirmed_false_positive`, `uncertain_candidate`, `empty_recall_agent_judgement`, and `true_alert`.

## Visual Direction

Use **Demo Theater / 战情指挥舱**.

Design characteristics:

- A strong presentation cockpit suitable for projector/live demo.
- Dark main stage with a contrasting light-gold battle-report panel.
- Primary colors: amber/gold for focus, mint/green for successful false-positive governance, red for real risk, cyan for API/technical evidence.
- Large display headline and KPI-style evidence fields.
- Monospace/tabular styling for `confidence`, `recall_score`, `risk_level`, `event_id`, and API labels.
- Motion must communicate state: running pulse, pipeline stage highlight, result transition. Avoid decorative-only motion.
- Prefer Windows-stable fonts: `Bahnschrift`, `Microsoft YaHei UI`, `Microsoft YaHei`, and monospace fallbacks.

## Page Architecture

The page has two vertical zones.

### 1. First-Screen Demo Stage

The first viewport should make the demo understandable without scrolling.

Required areas:

- Left identity/story panel:
  - Product title: intelligent false-positive governance cockpit.
  - One-sentence value proposition.
  - Live/API/Agent status.
  - Admin Token input in a compact, non-dominant position.

- Main stage:
  - Three primary scenario buttons.
  - Five-step pipeline: whitelist filtering, dedup merge, structured recall, ReAct Agent judgment, final storage/query.
  - Active stage highlighting based on selected/running scenario.
  - Current conclusion area showing risk transition and Agent evidence.

- Evidence summary:
  - `agent_verdict`
  - `agent_confidence`
  - `recall_score`
  - `risk_level` and `old_risk_level`
  - short `agent_explanation` or scenario-specific reason
  - latest request/response status

### 2. Evidence Detail Zone

Below the first screen, preserve proof that the frontend is using real data.

Required areas:

- Recent `alert_logs` records.
- Recent `false_positive_library` records.
- Recent `whitelist_rules` records.
- Expandable or visible raw response area for the last scenario run.

This area can reuse existing table components or use a redesigned compact evidence table, but it must not dominate the first screen.

## Main Demo Scenarios

Only three scenarios are primary in the stage controls.

### Scenario 1: Whitelist Drop

Purpose: prove deterministic rule priority.

Flow:

1. Create a whitelist rule.
2. Submit a matching `backup.exe` alert.
3. Show that the response has `dropped > 0`.
4. Explain that whitelist hits do not enter `alert_logs`.

Stage behavior:

- Highlight whitelist filtering.
- Show final result as discarded/no alert record.

### Scenario 2: Recall Hit + Agent Judgment

Purpose: main demo proving explainable false-positive governance.

Flow:

1. Seed a false-positive pattern.
2. Submit a matching customer-data upload alert.
3. Run structured recall and Eino ReAct Agent judgment through the real backend.
4. Show verdict, confidence, recall score, risk transition, and explanation.
5. Refresh evidence records.

Stage behavior:

- Highlight structured recall and ReAct Agent judgment.
- Show `false_positive`, confidence, recall score, and risk reduction.

### Scenario 3: True Alert

Purpose: prove the system does not incorrectly suppress real risk.

Flow:

1. Submit a high-risk/critical external upload alert, such as customer or contract data to `mail.qq.com`.
2. Show `true_alert` or non-false-positive handling.
3. Show risk level remains high/critical rather than being downgraded incorrectly.

Stage behavior:

- Highlight ReAct Agent judgment and final storage/query.
- Show true risk preservation.

## Demo Data Strategy

Use centralized synthetic demo data. The data must be synthetic but realistic, and every scenario must still call real APIs.

Shared story:

- Normal business user: `alice`.
- Normal host: `host-demo-01`.
- Normal process: `dlp-demo-browser.exe`.
- Normal target: `internal-crm.company.com`.
- Sensitive type: `customer` or a clearly named customer-data type.
- Deterministic noise process: `backup.exe`.
- True-risk user: `bob` or `eve`.
- True-risk target: `mail.qq.com`.

Add a small **Prepare Demo Data** action if needed. It may seed false-positive patterns and refresh records, but it must use real `/api/false-positives` and related APIs.

The page may label this as synthetic demo data so it is clear no real enterprise data is being shown.

## Error Handling

The demo must degrade into explainable states instead of raw stack traces.

Required states:

- Backend unavailable: show service disconnected and guide the user to start the server.
- `401 unauthorized`: highlight Admin Token and explain that the token is required.
- Agent slow: keep the scenario in running state and show the current stage.
- Agent failure: show that the external Agent boundary failed, preserve the raw response, and distinguish API ingestion from Agent analysis.
- Empty query result: explain whether the event may have been dropped by whitelist or still not visible in records.
- General request failure: show method, path, status, and recovery hint.

Raw responses must remain available for proof/debugging.

## UX Rules

- Primary demo buttons must be at least 44px tall and visually distinct.
- Do not rely on color alone; include text labels and icons where meaningful.
- Avoid nested cards inside cards.
- Do not let first-screen text overlap on common projector sizes such as 1366x768.
- Keep mobile responsive enough to avoid horizontal scroll, but desktop/projector is the priority.
- Use semantic color tokens or centralized CSS variables rather than scattered raw colors where practical.
- Respect reduced motion for major animations where practical.

## Verification Plan

Before claiming completion:

1. Run frontend build:

```powershell
cd module3-alert-agent\frontend
npm run build
```

2. Rebuild or ensure embedded frontend assets are updated according to the project’s existing process.

3. Run backend tests:

```powershell
cd module3-alert-agent
go test ./...
```

4. Run E2E smoke when MySQL and Ark environment variables are available:

```powershell
powershell -ExecutionPolicy Bypass -File .\tools\e2e-smoke.ps1
```

5. Manually verify in browser at `http://localhost:9090/`:

- First screen fits at 1366x768 without incoherent overlap.
- Three primary scenarios call real APIs.
- Records refresh after scenario completion.
- Evidence fields are visible: `agent_verdict`, `agent_confidence`, `recall_score`, `risk_level`, `old_risk_level`, `agent_explanation`.
- Raw response remains available.

## Open Items

None. The implementation should proceed within the approved scope above.

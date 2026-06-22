# Module 3 Acceptance Notes

## Phase Evidence

- Phase 1: Go/Hertz skeleton, config loading, route registration, MySQL-backed service startup, MySQL DDL, and old-table upgrade SQL are present in `cmd/server`, `internal/config`, `internal/router`, `internal/store`, `sql/schema.sql`, and `sql/upgrade.sql`.
- Phase 2: Whitelist CRUD, hot cache refresh, whitelist filtering, dedup merge windows, and `POST /api/client/events` are covered by pipeline/router tests.
- Phase 3: Ark ChatModel initialization, Eino ReAct runtime, four tools, structured false-positive recall, prompt loading, confidence threshold handling, false-positive TTL, `scenario_key` upsert, and bounded analyzer concurrency are implemented in `internal/agent` and `internal/store`.
- Phase 4: `POST /api/alerts/query` supports filters, pagination, sorting, page-size cap, and returns Agent analysis fields.
- Phase 5: Unit tests, API README, schema checks, and this acceptance note are included.
- Frontend: `GET /` serves the Demo Theater UI. The first screen is a live-demo command view with pipeline stages, primary scenario buttons, current Agent verdict metrics, and an evidence area that shows fixed JSON test inputs plus real HTTP responses for whitelist, dedup, true alert, suspected false positive, seeded false-positive pattern, and recall-hit Agent analysis scenarios.

## Verification Commands

```powershell
go test ./...
go test -race ./internal/pipeline -run TestDeduperCanAcceptConcurrentEvents -count=1
powershell -ExecutionPolicy Bypass -File .\tools\e2e-smoke.ps1
```

## External Verification Boundary

Unit tests do not call live MySQL or Ark services. Before a live demo:

1. Create the three tables from `sql/schema.sql`.
2. Set real MySQL and Ark ChatModel environment variables.
3. Run `go run ./cmd/server` from `module3-alert-agent`.
4. Open `http://localhost:9090/`.
5. Click each scenario button and verify that alert query records, false-positive-library records, and whitelist records refresh in the UI.

If the frontend source changes, rebuild it from `module3-alert-agent/frontend`:

```powershell
npm run typecheck
npm run build
```

The build writes static assets to `internal/router/web/`; current entry assets are `app.js` and `assets/app.css`.

`tools/e2e-smoke.ps1` automates the same live boundary once these environment variables are set:

- `MYSQL_HOST`
- `MYSQL_PORT`
- `MYSQL_USER`
- `MYSQL_PASSWORD`
- `MYSQL_DATABASE`
- `ARK_CHAT_MODEL`
- `ARK_API_KEY`

The script applies `sql/schema.sql` and `sql/upgrade.sql`, starts the service unless `-UseRunningServer` is passed, checks the frontend assets, and verifies whitelist, false-positive-library, and alert-query records through the real HTTP APIs.

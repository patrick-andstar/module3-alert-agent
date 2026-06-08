package store

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
	"module3-alert-agent/internal/router"
)

type MySQLService struct {
	db               *sql.DB
	whitelist        *pipeline.WhitelistCache
	deduper          *pipeline.Deduper
	stopDeduper      func()
	analyzer         router.EventAnalyzer
	analysisTimeout  time.Duration
	maxRecallRecords int
}

func NewMySQLService(ctx context.Context, db *sql.DB, dedupWindows map[string]int) (*MySQLService, error) {
	deduper := pipeline.NewDeduper(dedupWindows)
	service := &MySQLService{
		db:               db,
		whitelist:        pipeline.NewWhitelistCache(nil),
		deduper:          deduper,
		stopDeduper:      deduper.StartCleaner(30*time.Second, nil),
		analysisTimeout:  30 * time.Second,
		maxRecallRecords: 500,
	}
	if err := service.RefreshWhitelist(ctx); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *MySQLService) Close() {
	stop := s.stopDeduper
	s.stopDeduper = nil
	if stop != nil {
		stop()
	}
}

func (s *MySQLService) ProcessEvents(ctx context.Context, events []model.Event) pipeline.Result {
	pipe := pipeline.New(s.whitelist, s.deduper)
	result := pipe.Process(events)
	result.Events = s.analyzeEvents(ctx, result.Events)
	for _, event := range result.Events {
		if err := s.saveAlert(ctx, event); err != nil {
			slog.Error("save alert", "event_id", event.EventID, "err", err)
		}
	}
	return result
}

func (s *MySQLService) SetAnalyzer(analyzer router.EventAnalyzer) {
	s.analyzer = analyzer
}

func (s *MySQLService) SetAnalysisTimeout(timeout time.Duration) {
	if timeout > 0 {
		s.analysisTimeout = timeout
	}
}

func (s *MySQLService) SetMaxRecallRecords(limit int) {
	if limit > 0 {
		s.maxRecallRecords = limit
	}
}

func (s *MySQLService) WhitelistCache() *pipeline.WhitelistCache {
	return s.whitelist
}

func (s *MySQLService) QueryAlerts(query router.AlertQuery) router.AlertQueryResult {
	countSQL, countArgs := BuildAlertCountSQL(query)
	var total int
	if err := s.db.QueryRowContext(context.Background(), countSQL, countArgs...).Scan(&total); err != nil {
		slog.Error("count alerts", "err", err)
		return router.AlertQueryResult{Total: 0, Page: 1, PageSize: 20, Data: []model.Event{}}
	}

	selectSQL, selectArgs := BuildAlertQuerySQL(query)
	rows, err := s.db.QueryContext(context.Background(), selectSQL, selectArgs...)
	if err != nil {
		slog.Error("query alerts", "err", err)
		return router.AlertQueryResult{Total: total, Page: 1, PageSize: 20, Data: []model.Event{}}
	}
	defer rows.Close()

	data := []model.Event{}
	for rows.Next() {
		event, err := scanAlertEvent(rows)
		if err != nil {
			slog.Error("scan alert", "err", err)
			continue
		}
		data = append(data, event)
	}
	if err := rows.Err(); err != nil {
		slog.Error("iterate alerts", "err", err)
	}

	normalized := model.NormalizeAlertQuery(query)
	return router.AlertQueryResult{
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
		Data:     data,
	}
}

func (s *MySQLService) ListWhitelistRules() []model.WhitelistRule {
	rules, err := s.loadWhitelistRules(context.Background(), false)
	if err != nil {
		slog.Error("list whitelist rules", "err", err)
		return []model.WhitelistRule{}
	}
	return rules
}

func (s *MySQLService) CreateWhitelistRule(rule model.WhitelistRule) model.WhitelistRule {
	res, err := s.db.ExecContext(context.Background(), BuildWhitelistInsertSQL(), WhitelistInsertArgs(rule)...)
	if err != nil {
		slog.Error("create whitelist rule", "err", err)
		return model.WhitelistRule{}
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("read whitelist insert id", "err", err)
	}
	rule = normalizeWhitelistRule(rule)
	rule.ID = id
	_ = s.RefreshWhitelist(context.Background())
	return rule
}

func (s *MySQLService) UpdateWhitelistRule(id int64, rule model.WhitelistRule) (model.WhitelistRule, bool) {
	res, err := s.db.ExecContext(context.Background(), BuildWhitelistUpdateSQL(), WhitelistUpdateArgs(id, rule)...)
	if err != nil {
		slog.Error("update whitelist rule", "err", err)
		return model.WhitelistRule{}, false
	}
	affected, err := res.RowsAffected()
	if err != nil {
		slog.Error("read whitelist affected rows", "err", err)
		return model.WhitelistRule{}, false
	}
	if affected == 0 {
		return model.WhitelistRule{}, false
	}
	rule = normalizeWhitelistRule(rule)
	rule.ID = id
	_ = s.RefreshWhitelist(context.Background())
	return rule, true
}

func (s *MySQLService) DeleteWhitelistRule(id int64) bool {
	res, err := s.db.ExecContext(context.Background(), "DELETE FROM whitelist_rules WHERE id = ?", id)
	if err != nil {
		slog.Error("delete whitelist rule", "err", err)
		return false
	}
	affected, err := res.RowsAffected()
	if err != nil || affected == 0 {
		return false
	}
	_ = s.RefreshWhitelist(context.Background())
	return true
}

func (s *MySQLService) ListFalsePositiveRecords() []model.FalsePositiveRecord {
	records, err := s.listFalsePositiveRecords(context.Background())
	if err != nil {
		slog.Error("list false positives", "err", err)
		return []model.FalsePositiveRecord{}
	}
	return records
}

func (s *MySQLService) ListFalsePositiveRecordsForAnalysis(ctx context.Context) ([]model.FalsePositiveRecord, error) {
	return s.listFalsePositiveRecords(ctx)
}

func (s *MySQLService) Save(record model.FalsePositiveRecord) error {
	args, err := FalsePositiveInsertArgs(record)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), BuildFalsePositiveInsertSQL(), args...)
	return err
}

func (s *MySQLService) listFalsePositiveRecords(ctx context.Context) ([]model.FalsePositiveRecord, error) {
	limit := s.maxRecallRecords
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(scenario_key, ''), COALESCE(host_id, ''), COALESCE(user_id, ''), COALESCE(sensitive_type, ''), COALESCE(risk_level, ''), COALESCE(process_name, ''), COALESCE(process_path, ''), COALESCE(target, ''), COALESCE(operation, ''), COALESCE(reason, ''), COALESCE(embedding_json, '[]'), COALESCE(hit_count, 1), COALESCE(last_seen_at, created_at), expired_at, created_at
FROM false_positive_library
WHERE expired_at > ?
ORDER BY created_at DESC
LIMIT ?`, time.Now(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []model.FalsePositiveRecord{}
	for rows.Next() {
		record, err := scanFalsePositiveRecord(rows)
		if err != nil {
			slog.Error("scan false positive", "err", err)
			continue
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *MySQLService) DeleteFalsePositiveRecord(id int64) bool {
	res, err := s.db.ExecContext(context.Background(), "DELETE FROM false_positive_library WHERE id = ?", id)
	if err != nil {
		slog.Error("delete false positive", "err", err)
		return false
	}
	affected, err := res.RowsAffected()
	return err == nil && affected > 0
}

func (s *MySQLService) RefreshWhitelist(ctx context.Context) error {
	rules, err := s.loadWhitelistRules(ctx, true)
	if err != nil {
		return err
	}
	s.whitelist.Refresh(rules)
	return nil
}

func (s *MySQLService) saveAlert(ctx context.Context, event model.Event) error {
	_, err := s.db.ExecContext(ctx, BuildAlertUpsertSQL(), AlertUpsertArgs(event)...)
	return err
}

func (s *MySQLService) analyzeEvents(ctx context.Context, events []model.Event) []model.Event {
	return router.AnalyzeEvents(ctx, s.analyzer, s.analysisTimeout, events)
}

func (s *MySQLService) loadWhitelistRules(ctx context.Context, enabledOnly bool) ([]model.WhitelistRule, error) {
	rows, err := s.db.QueryContext(ctx, BuildWhitelistSelectSQL(enabledOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := []model.WhitelistRule{}
	for rows.Next() {
		var rule model.WhitelistRule
		if err := rows.Scan(&rule.ID, &rule.RuleName, &rule.Logic, &rule.ProcessName, &rule.UserID, &rule.FilePathPattern, &rule.TimeWindowStart, &rule.TimeWindowEnd, &rule.Enabled); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

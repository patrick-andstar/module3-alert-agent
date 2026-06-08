package router

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
)

type Service interface {
	ProcessEvents(context.Context, []model.Event) pipeline.Result
	QueryAlerts(AlertQuery) AlertQueryResult
	ListWhitelistRules() []model.WhitelistRule
	CreateWhitelistRule(model.WhitelistRule) model.WhitelistRule
	UpdateWhitelistRule(int64, model.WhitelistRule) (model.WhitelistRule, bool)
	DeleteWhitelistRule(int64) bool
	ListFalsePositiveRecords() []model.FalsePositiveRecord
	Save(model.FalsePositiveRecord) error
	DeleteFalsePositiveRecord(int64) bool
}

type EventAnalyzer interface {
	Analyze(context.Context, model.Event) (model.Event, error)
}

type MemoryService struct {
	mu              sync.RWMutex
	nextID          int64
	nextFPID        int64
	rules           []model.WhitelistRule
	fpRecords       []model.FalsePositiveRecord
	alerts          []model.Event
	whitelist       *pipeline.WhitelistCache
	deduper         *pipeline.Deduper
	stopDeduper     func()
	analyzer        EventAnalyzer
	now             func() time.Time
	analysisTimeout time.Duration
}

func NewMemoryService(dedupWindows map[string]int) *MemoryService {
	deduper := pipeline.NewDeduper(dedupWindows)
	return &MemoryService{
		nextID:          1,
		nextFPID:        1,
		whitelist:       pipeline.NewWhitelistCache(nil),
		deduper:         deduper,
		stopDeduper:     deduper.StartCleaner(30*time.Second, nil),
		now:             time.Now,
		analysisTimeout: 30 * time.Second,
	}
}

func (s *MemoryService) Close() {
	s.mu.Lock()
	stop := s.stopDeduper
	s.stopDeduper = nil
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func (s *MemoryService) SetNow(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.now = now
}

func (s *MemoryService) SetAnalyzer(analyzer EventAnalyzer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.analyzer = analyzer
}

func (s *MemoryService) SetAnalysisTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if timeout > 0 {
		s.analysisTimeout = timeout
	}
}

func (s *MemoryService) ProcessEvents(ctx context.Context, events []model.Event) pipeline.Result {
	pipe := pipeline.New(s.whitelist, s.deduper)
	result := pipe.Process(events)
	result.Events = s.analyzeEvents(ctx, result.Events)
	s.mu.Lock()
	s.alerts = append(s.alerts, result.Events...)
	s.mu.Unlock()
	return result
}

func (s *MemoryService) AddAlert(event model.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.alerts = append(s.alerts, event)
}

func (s *MemoryService) QueryAlerts(query AlertQuery) AlertQueryResult {
	query = normalizeAlertQuery(query)

	s.mu.RLock()
	filtered := make([]model.Event, 0, len(s.alerts))
	for _, event := range s.alerts {
		if matchesAlertQuery(event, query) {
			filtered = append(filtered, event)
		}
	}
	s.mu.RUnlock()

	sort.Slice(filtered, func(i, j int) bool {
		left := filtered[i]
		right := filtered[j]
		desc := query.Order == "desc"
		switch query.OrderBy {
		case "timestamp":
			if desc {
				return left.Timestamp > right.Timestamp
			}
			return left.Timestamp < right.Timestamp
		case "created_at":
			if desc {
				return left.Timestamp > right.Timestamp
			}
			return left.Timestamp < right.Timestamp
		default:
			if desc {
				return left.EventID > right.EventID
			}
			return left.EventID < right.EventID
		}
	})

	total := len(filtered)
	start := (query.Page - 1) * query.PageSize
	if start > total {
		start = total
	}
	end := start + query.PageSize
	if end > total {
		end = total
	}

	return AlertQueryResult{
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
		Data:     filtered[start:end],
	}
}

func (s *MemoryService) ListWhitelistRules() []model.WhitelistRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]model.WhitelistRule(nil), s.rules...)
}

func (s *MemoryService) CreateWhitelistRule(rule model.WhitelistRule) model.WhitelistRule {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.Logic == "" {
		rule.Logic = "OR"
	}
	rule.ID = s.nextID
	s.nextID++
	s.rules = append(s.rules, rule)
	s.refreshLocked()
	return rule
}

func (s *MemoryService) UpdateWhitelistRule(id int64, rule model.WhitelistRule) (model.WhitelistRule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.rules {
		if s.rules[i].ID == id {
			rule.ID = id
			if rule.Logic == "" {
				rule.Logic = "OR"
			}
			s.rules[i] = rule
			s.refreshLocked()
			return rule, true
		}
	}
	return model.WhitelistRule{}, false
}

func (s *MemoryService) DeleteWhitelistRule(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.rules {
		if s.rules[i].ID == id {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			s.refreshLocked()
			return true
		}
	}
	return false
}

func (s *MemoryService) ListFalsePositiveRecords() []model.FalsePositiveRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := s.now()
	records := make([]model.FalsePositiveRecord, 0, len(s.fpRecords))
	for _, record := range s.fpRecords {
		if record.ExpiredAt.IsZero() || record.ExpiredAt.After(now) {
			records = append(records, record)
		}
	}
	return records
}

func (s *MemoryService) ListFalsePositiveRecordsForAnalysis(context.Context) ([]model.FalsePositiveRecord, error) {
	return s.ListFalsePositiveRecords(), nil
}

func (s *MemoryService) Save(record model.FalsePositiveRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ScenarioKey != "" {
		for i := range s.fpRecords {
			if s.fpRecords[i].ScenarioKey == record.ScenarioKey {
				increment := record.HitCount
				if increment <= 0 {
					increment = 1
				}
				s.fpRecords[i].HitCount += increment
				if !record.LastSeenAt.IsZero() {
					s.fpRecords[i].LastSeenAt = record.LastSeenAt
				}
				if !record.ExpiredAt.IsZero() {
					s.fpRecords[i].ExpiredAt = record.ExpiredAt
				}
				if record.Reason != "" {
					s.fpRecords[i].Reason = record.Reason
				}
				return nil
			}
		}
	}

	record.ID = s.nextFPID
	s.nextFPID++
	if record.HitCount <= 0 {
		record.HitCount = 1
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = s.now()
	}
	s.fpRecords = append(s.fpRecords, record)
	return nil
}

func (s *MemoryService) DeleteFalsePositiveRecord(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.fpRecords {
		if s.fpRecords[i].ID == id {
			s.fpRecords = append(s.fpRecords[:i], s.fpRecords[i+1:]...)
			return true
		}
	}
	return false
}

func (s *MemoryService) refreshLocked() {
	s.whitelist.Refresh(s.rules)
}

func (s *MemoryService) analyzeEvents(ctx context.Context, events []model.Event) []model.Event {
	s.mu.RLock()
	analyzer := s.analyzer
	timeout := s.analysisTimeout
	s.mu.RUnlock()
	return AnalyzeEvents(ctx, analyzer, timeout, events)
}

func parseID(value string) (int64, error) {
	var id int64
	if _, err := fmt.Sscan(value, &id); err != nil {
		return 0, err
	}
	return id, nil
}

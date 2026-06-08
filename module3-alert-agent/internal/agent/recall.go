package agent

import (
	"net/url"
	"sort"
	"strings"
	"time"

	"module3-alert-agent/internal/model"
)

type RecallResult struct {
	Record     model.FalsePositiveRecord
	Similarity float64
	Score      float64
}

type StructuredRecallConfig struct {
	TopK      int
	Threshold float64
	Now       func() time.Time
}

func ScenarioKey(event model.Event) string {
	return strings.Join([]string{
		normalizeToken(event.SensitiveType),
		normalizeToken(event.Operation),
		normalizeToken(event.ProcessName),
		normalizeTarget(event.Target),
	}, "|")
}

func StructuredRecall(event model.Event, records []model.FalsePositiveRecord, cfg StructuredRecallConfig) []RecallResult {
	topK := cfg.TopK
	if topK <= 0 {
		topK = 5
	}
	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = 0.60
	}
	now := time.Now()
	if cfg.Now != nil {
		now = cfg.Now()
	}

	results := make([]RecallResult, 0, len(records))
	for _, record := range records {
		if !record.ExpiredAt.IsZero() && !record.ExpiredAt.After(now) {
			continue
		}
		score := structuredScore(event, record)
		if score >= threshold {
			results = append(results, RecallResult{Record: record, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		return results[:topK]
	}
	return results
}

func structuredScore(event model.Event, record model.FalsePositiveRecord) float64 {
	var score float64
	if normalizeToken(event.SensitiveType) == normalizeToken(record.SensitiveType) {
		score += 0.25
	}
	if normalizeToken(event.Operation) == normalizeToken(record.Operation) {
		score += 0.20
	}
	if normalizeToken(event.ProcessName) == normalizeToken(record.ProcessName) {
		score += 0.20
	}
	if normalizeTarget(event.Target) == normalizeTarget(record.Target) {
		score += 0.20
	}
	if normalizeToken(event.UserID) == normalizeToken(record.UserID) {
		score += 0.10
	}
	if normalizeToken(event.ProcessPath) == normalizeToken(record.ProcessPath) {
		score += 0.05
	}
	return score
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeTarget(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Host)
	}
	return strings.ToLower(strings.Trim(value, "/"))
}

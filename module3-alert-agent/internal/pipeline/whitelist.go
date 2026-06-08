package pipeline

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"module3-alert-agent/internal/model"
)

type WhitelistCache struct {
	mu    sync.RWMutex
	rules []model.WhitelistRule
}

func NewWhitelistCache(rules []model.WhitelistRule) *WhitelistCache {
	cache := &WhitelistCache{}
	cache.Refresh(rules)
	return cache
}

func (c *WhitelistCache) Refresh(rules []model.WhitelistRule) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rules = append([]model.WhitelistRule(nil), rules...)
}

func (c *WhitelistCache) Match(event model.Event) bool {
	matched, _ := c.MatchRule(event)
	return matched
}

func (c *WhitelistCache) MatchRule(event model.Event) (bool, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, rule := range c.rules {
		if !rule.Enabled {
			continue
		}
		if matchRule(rule, event) {
			return true, rule.RuleName
		}
	}
	return false, ""
}

func matchRule(rule model.WhitelistRule, event model.Event) bool {
	checks := []bool{}
	if rule.ProcessName != "" {
		checks = append(checks, strings.EqualFold(rule.ProcessName, event.ProcessName))
	}
	if rule.UserID != "" {
		checks = append(checks, rule.UserID == event.UserID)
	}
	if rule.FilePathPattern != "" {
		checks = append(checks, matchPath(rule.FilePathPattern, event.FilePath))
	}
	if rule.TimeWindowStart != "" || rule.TimeWindowEnd != "" {
		checks = append(checks, matchTimeWindow(rule.TimeWindowStart, rule.TimeWindowEnd, event.Timestamp))
	}
	if len(checks) == 0 {
		return false
	}

	if strings.EqualFold(rule.Logic, "AND") {
		for _, ok := range checks {
			if !ok {
				return false
			}
		}
		return true
	}

	for _, ok := range checks {
		if ok {
			return true
		}
	}
	return false
}

func matchPath(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)
	ok, err := filepath.Match(pattern, path)
	return err == nil && ok
}

func matchTimeWindow(start, end string, timestamp int64) bool {
	if start == "" && end == "" {
		return true
	}
	if timestamp <= 0 {
		return false
	}

	eventTime := time.Unix(timestamp, 0).In(time.Local)
	current := eventTime.Hour()*3600 + eventTime.Minute()*60 + eventTime.Second()

	startSeconds, ok := parseClock(start)
	if !ok {
		return false
	}
	endSeconds, ok := parseClock(end)
	if !ok {
		return false
	}

	if endSeconds < startSeconds {
		return current >= startSeconds || current <= endSeconds
	}
	return current >= startSeconds && current <= endSeconds
}

func parseClock(value string) (int, bool) {
	parsed, err := time.Parse("15:04:05", value)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*3600 + parsed.Minute()*60 + parsed.Second(), true
}

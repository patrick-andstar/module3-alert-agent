package pipeline_test

import (
	"testing"
	"time"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
)

func TestWhitelistORRuleMatchesAnyConfiguredCondition(t *testing.T) {
	cache := pipeline.NewWhitelistCache([]model.WhitelistRule{
		{
			RuleName:        "backup exemption",
			Logic:           "OR",
			ProcessName:     "backup.exe",
			FilePathPattern: "C:/Backup/*",
			Enabled:         true,
		},
	})

	event := model.Event{
		ProcessName: "chrome.exe",
		FilePath:    "C:/Backup/customer.xlsx",
	}

	if !cache.Match(event) {
		t.Fatal("OR whitelist rule did not match file path pattern")
	}
}

func TestWhitelistANDRuleRequiresAllConfiguredConditions(t *testing.T) {
	cache := pipeline.NewWhitelistCache([]model.WhitelistRule{
		{
			RuleName:    "admin crm",
			Logic:       "AND",
			UserID:      "admin",
			ProcessName: "chrome.exe",
			Enabled:     true,
		},
	})

	matching := model.Event{UserID: "admin", ProcessName: "chrome.exe"}
	if !cache.Match(matching) {
		t.Fatal("AND whitelist rule did not match when all conditions matched")
	}

	partial := model.Event{UserID: "admin", ProcessName: "outlook.exe"}
	if cache.Match(partial) {
		t.Fatal("AND whitelist rule matched with only one condition")
	}
}

func TestWhitelistRefreshReplacesCachedRules(t *testing.T) {
	cache := pipeline.NewWhitelistCache([]model.WhitelistRule{
		{RuleName: "old", Logic: "OR", ProcessName: "old.exe", Enabled: true},
	})

	cache.Refresh([]model.WhitelistRule{
		{RuleName: "new", Logic: "OR", ProcessName: "new.exe", Enabled: true},
	})

	if cache.Match(model.Event{ProcessName: "old.exe"}) {
		t.Fatal("old rule still matched after refresh")
	}
	if !cache.Match(model.Event{ProcessName: "new.exe"}) {
		t.Fatal("new rule did not match after refresh")
	}
}

func TestWhitelistTimeWindowMatchesInsideWindowOnly(t *testing.T) {
	cache := pipeline.NewWhitelistCache([]model.WhitelistRule{
		{
			RuleName:        "office-hours upload",
			Logic:           "AND",
			ProcessName:     "chrome.exe",
			TimeWindowStart: "09:00:00",
			TimeWindowEnd:   "18:00:00",
			Enabled:         true,
		},
	})

	matching := model.Event{
		ProcessName: "chrome.exe",
		Timestamp:   time.Date(2026, 6, 5, 10, 0, 0, 0, time.Local).Unix(),
	}
	if !cache.Match(matching) {
		t.Fatal("rule should match inside configured time window")
	}

	outside := matching
	outside.Timestamp = time.Date(2026, 6, 5, 20, 0, 0, 0, time.Local).Unix()
	if cache.Match(outside) {
		t.Fatal("rule should not match outside configured time window")
	}
}

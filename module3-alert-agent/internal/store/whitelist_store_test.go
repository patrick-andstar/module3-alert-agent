package store_test

import (
	"strings"
	"testing"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/store"
)

func TestWhitelistInsertArgsDefaultsLogicToOR(t *testing.T) {
	args := store.WhitelistInsertArgs(model.WhitelistRule{
		RuleName:        "office backup",
		ProcessName:     "backup.exe",
		TimeWindowStart: "09:00:00",
		TimeWindowEnd:   "18:00:00",
		Enabled:         true,
	})

	if got, want := len(args), 8; got != want {
		t.Fatalf("len(args) = %d, want %d", got, want)
	}
	if args[1] != "OR" {
		t.Fatalf("logic arg = %#v, want OR", args[1])
	}
	if args[5] != "09:00:00" || args[6] != "18:00:00" {
		t.Fatalf("time window args = %#v, want persisted start/end", args[5:7])
	}
}

func TestBuildWhitelistUpdateSQLScopesByID(t *testing.T) {
	sql := store.BuildWhitelistUpdateSQL()
	required := []string{
		"UPDATE whitelist_rules",
		"rule_name = ?",
		"time_window_start = ?",
		"time_window_end = ?",
		"enabled = ?",
		"WHERE id = ?",
	}
	for _, needle := range required {
		if !strings.Contains(sql, needle) {
			t.Fatalf("SQL %q missing %q", sql, needle)
		}
	}
}

func TestBuildWhitelistSelectSQLSeparatesListAndCacheQueries(t *testing.T) {
	listSQL := store.BuildWhitelistSelectSQL(false)
	if strings.Contains(listSQL, "WHERE enabled = TRUE") {
		t.Fatalf("list SQL %q should include disabled rules for management API", listSQL)
	}

	cacheSQL := store.BuildWhitelistSelectSQL(true)
	if !strings.Contains(cacheSQL, "WHERE enabled = TRUE") {
		t.Fatalf("cache SQL %q should only load enabled rules", cacheSQL)
	}
	if !strings.Contains(cacheSQL, "time_window_start") || !strings.Contains(cacheSQL, "time_window_end") {
		t.Fatalf("cache SQL %q should select time window columns", cacheSQL)
	}
}

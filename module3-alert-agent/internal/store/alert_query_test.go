package store_test

import (
	"strings"
	"testing"

	"module3-alert-agent/internal/router"
	"module3-alert-agent/internal/store"
)

func TestBuildAlertQueryUsesFiltersPaginationAndSort(t *testing.T) {
	sql, args := store.BuildAlertQuerySQL(router.AlertQuery{
		RiskLevel: "high",
		UserID:    "user-1",
		Page:      2,
		PageSize:  20,
		OrderBy:   "timestamp",
		Order:     "desc",
	})

	required := []string{
		"FROM alert_logs",
		"COALESCE(host_id, '')",
		"risk_level = ?",
		"user_id = ?",
		"ORDER BY timestamp DESC",
		"LIMIT ? OFFSET ?",
	}
	for _, needle := range required {
		if !strings.Contains(sql, needle) {
			t.Fatalf("SQL %q missing %q", sql, needle)
		}
	}
	if len(args) != 4 {
		t.Fatalf("len(args) = %d, want 4", len(args))
	}
	if args[2] != 20 || args[3] != 20 {
		t.Fatalf("pagination args = %#v, want limit=20 offset=20", args)
	}
}

func TestBuildAlertCountQueryUsesSameFiltersWithoutPagination(t *testing.T) {
	sql, args := store.BuildAlertCountSQL(router.AlertQuery{
		StartTime: 100,
		EndTime:   200,
		UserID:    "user-1",
		Page:      3,
		PageSize:  50,
	})

	required := []string{
		"SELECT COUNT(*) FROM alert_logs",
		"timestamp >= ?",
		"timestamp <= ?",
		"user_id = ?",
	}
	for _, needle := range required {
		if !strings.Contains(sql, needle) {
			t.Fatalf("SQL %q missing %q", sql, needle)
		}
	}
	if strings.Contains(sql, "LIMIT") || strings.Contains(sql, "ORDER BY") {
		t.Fatalf("count SQL %q should not include ordering or pagination", sql)
	}
	if len(args) != 3 {
		t.Fatalf("len(args) = %d, want 3", len(args))
	}
}

func TestBuildAlertQueryNormalizesInvalidPagingAndOrdering(t *testing.T) {
	_, args := store.BuildAlertQuerySQL(router.AlertQuery{
		Page:     0,
		PageSize: 1000,
		Order:    "sideways",
	})

	if len(args) != 2 {
		t.Fatalf("len(args) = %d, want 2 for normalized pagination", len(args))
	}
	if args[0] != 100 {
		t.Fatalf("limit arg = %#v, want capped page_size 100", args[0])
	}
	if args[1] != 0 {
		t.Fatalf("offset arg = %#v, want normalized page 1 offset 0", args[1])
	}
}

func TestBuildAlertQueryAllowsCreatedAtOrdering(t *testing.T) {
	sql, _ := store.BuildAlertQuerySQL(router.AlertQuery{OrderBy: "created_at", Order: "asc"})
	if !strings.Contains(sql, "ORDER BY created_at ASC") {
		t.Fatalf("SQL %q should order by created_at ASC", sql)
	}
}

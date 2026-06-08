package store_test

import (
	"strings"
	"testing"
	"time"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/store"
)

func TestFalsePositiveInsertArgsSerializesEmbeddingJSON(t *testing.T) {
	lastSeen := time.Date(2026, 6, 5, 11, 0, 0, 0, time.UTC)
	record := model.FalsePositiveRecord{
		ScenarioKey:   "customer|upload|chrome.exe|crm.internal",
		HostID:        "host-1",
		UserID:        "user-1",
		SensitiveType: "customer",
		RiskLevel:     "low",
		ProcessName:   "chrome.exe",
		ProcessPath:   "C:/Chrome/chrome.exe",
		Target:        "crm.internal",
		Operation:     "upload",
		Reason:        "normal crm upload",
		Embedding:     []float64{0.1, 0.2, 0.3},
		HitCount:      2,
		LastSeenAt:    lastSeen,
		ExpiredAt:     time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC),
	}

	args, err := store.FalsePositiveInsertArgs(record)
	if err != nil {
		t.Fatalf("FalsePositiveInsertArgs returned error: %v", err)
	}
	if got, want := len(args), 14; got != want {
		t.Fatalf("len(args) = %d, want %d", got, want)
	}
	if args[0] != "customer|upload|chrome.exe|crm.internal" {
		t.Fatalf("scenario_key arg = %#v, want dedupe key", args[0])
	}
	if args[10] != "[0.1,0.2,0.3]" {
		t.Fatalf("embedding_json arg = %#v, want compact JSON", args[10])
	}
	if args[11] != 2 {
		t.Fatalf("hit_count arg = %#v, want persisted count", args[11])
	}
	if args[12] != lastSeen {
		t.Fatalf("last_seen_at arg = %#v, want %v", args[12], lastSeen)
	}
}

func TestDecodeEmbeddingJSONRejectsInvalidJSON(t *testing.T) {
	if _, err := store.DecodeEmbeddingJSON("not-json"); err == nil {
		t.Fatal("DecodeEmbeddingJSON returned nil error for invalid JSON")
	}
}

func TestBuildFalsePositiveInsertSQLTargetsLibraryTable(t *testing.T) {
	sql := store.BuildFalsePositiveInsertSQL()
	required := []string{
		"INSERT INTO false_positive_library",
		"scenario_key",
		"embedding_json",
		"hit_count",
		"last_seen_at",
		"expired_at",
		"ON DUPLICATE KEY UPDATE",
	}
	for _, needle := range required {
		if !strings.Contains(sql, needle) {
			t.Fatalf("SQL %q missing %q", sql, needle)
		}
	}
}

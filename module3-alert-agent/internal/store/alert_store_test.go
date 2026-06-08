package store_test

import (
	"strings"
	"testing"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/store"
)

func TestBuildAlertUpsertSQLStoresProcessedAlert(t *testing.T) {
	sql := store.BuildAlertUpsertSQL()

	required := []string{
		"INSERT INTO alert_logs",
		"event_id",
		"risk_level",
		"old_risk_level",
		"is_merge_event",
		"file_count",
		"files_json",
		"false_positive_reason",
		"agent_verdict",
		"agent_confidence",
		"agent_explanation",
		"recall_score",
		"ON DUPLICATE KEY UPDATE",
	}
	for _, needle := range required {
		if !strings.Contains(sql, needle) {
			t.Fatalf("SQL %q missing %q", sql, needle)
		}
	}
}

func TestAlertUpsertArgsUsesDefaultFileCountForSingleEvent(t *testing.T) {
	args := store.AlertUpsertArgs(model.Event{
		EventID:             "evt-1",
		HostID:              "host-1",
		UserID:              "user-1",
		RiskLevel:           "high",
		OldRiskLevel:        "critical",
		SensitiveType:       "customer",
		ProcessName:         "chrome.exe",
		Operation:           "upload",
		FalsePositiveReason: "normal crm upload",
		AgentVerdict:        "false_positive",
		AgentConfidence:     0.91,
		AgentExplanation:    "matched structured recall",
		RecallScore:         0.86,
		Files: []model.FileInfo{
			{FilePath: "C:/a.xlsx", FileHash: "hash-a"},
		},
		Timestamp: 123,
	})

	if got, want := len(args), 23; got != want {
		t.Fatalf("len(args) = %d, want %d", got, want)
	}
	if args[15] != 1 {
		t.Fatalf("file_count arg = %#v, want 1", args[15])
	}
	if args[16] != `[{"file_path":"C:/a.xlsx","file_hash":"hash-a"}]` {
		t.Fatalf("files_json arg = %#v, want serialized files", args[16])
	}
	if args[17] != "normal crm upload" {
		t.Fatalf("false_positive_reason arg = %#v, want persisted reason", args[17])
	}
	if args[18] != "false_positive" || args[19] != 0.91 || args[20] != "matched structured recall" || args[21] != 0.86 {
		t.Fatalf("agent args = %#v, want persisted verdict/confidence/explanation/recall score", args[18:22])
	}
}

func TestAlertUpsertArgsUsesNullForEmptyOptionalEnum(t *testing.T) {
	args := store.AlertUpsertArgs(model.Event{
		EventID:   "evt-1",
		RiskLevel: "high",
		Timestamp: 123,
	})

	if args[8] != nil {
		t.Fatalf("old_risk_level arg = %#v, want nil for nullable enum", args[8])
	}
}

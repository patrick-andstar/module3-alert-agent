package store_test

import (
	"os"
	"strings"
	"testing"
)

func TestSchemaDefinesRequiredTablesAndIndexes(t *testing.T) {
	schema, err := os.ReadFile("../../sql/schema.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	text := strings.ToLower(string(schema))

	required := []string{
		"create table if not exists alert_logs",
		"create table if not exists whitelist_rules",
		"create table if not exists false_positive_library",
		"event_id varchar(64) not null unique",
		"files_json",
		"agent_verdict enum('false_positive','true_alert','uncertain')",
		"agent_confidence decimal(4,3)",
		"agent_explanation text",
		"recall_score decimal(4,3)",
		"logic enum('and','or')",
		"scenario_key varchar(512) not null unique",
		"embedding_json longtext",
		"hit_count int default 1",
		"last_seen_at datetime",
		"index idx_timestamp",
		"index idx_agent_verdict",
		"index idx_expired",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("schema missing %q", needle)
		}
	}
}

func TestUpgradeSchemaBackfillsExistingTables(t *testing.T) {
	schema, err := os.ReadFile("../../sql/upgrade.sql")
	if err != nil {
		t.Fatalf("read upgrade schema: %v", err)
	}
	text := strings.ToLower(string(schema))

	required := []string{
		"dlp_add_column_if_missing",
		"agent_verdict",
		"agent_confidence",
		"agent_explanation",
		"recall_score",
		"scenario_key",
		"hit_count",
		"last_seen_at",
		"row_number() over",
		"add unique index scenario_key",
		"idx_last_seen",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("upgrade schema missing %q", needle)
		}
	}
}

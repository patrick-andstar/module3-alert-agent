package pipeline_test

import (
	"testing"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
)

func TestPipelineDropsWhitelistedEventsAndProcessesRemainingEvents(t *testing.T) {
	pipe := pipeline.New(
		pipeline.NewWhitelistCache([]model.WhitelistRule{
			{RuleName: "backup", Logic: "OR", ProcessName: "backup.exe", Enabled: true},
		}),
		pipeline.NewDeduper(map[string]int{"high": 60}),
	)

	result := pipe.Process([]model.Event{
		{EventID: "drop", ProcessName: "backup.exe", RiskLevel: "high"},
		{
			EventID:       "keep-1",
			HostID:        "host-1",
			UserID:        "user-1",
			ProcessName:   "chrome.exe",
			SensitiveType: "客户资料",
			Operation:     "upload",
			RiskLevel:     "high",
			FilePath:      "C:/a.xlsx",
			Timestamp:     1000,
		},
		{
			EventID:       "keep-2",
			HostID:        "host-1",
			UserID:        "user-1",
			ProcessName:   "chrome.exe",
			SensitiveType: "客户资料",
			Operation:     "upload",
			RiskLevel:     "high",
			FilePath:      "C:/b.xlsx",
			Timestamp:     1010,
		},
	})

	if result.Accepted != 2 {
		t.Fatalf("Accepted = %d, want 2", result.Accepted)
	}
	if result.Dropped != 1 {
		t.Fatalf("Dropped = %d, want 1", result.Dropped)
	}
	if len(result.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1 merged alert", len(result.Events))
	}
	if !result.Events[0].IsMergeEvent {
		t.Fatal("kept events should collapse into one merged alert")
	}
}

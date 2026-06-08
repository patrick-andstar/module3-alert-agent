package pipeline_test

import (
	"sync"
	"testing"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
)

func TestDeduperMergesEventsWithExactKeyInsideRiskWindow(t *testing.T) {
	deduper := pipeline.NewDeduper(map[string]int{
		"high": 60,
	})

	first := model.Event{
		EventID:       "evt-1",
		HostID:        "host-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "客户资料",
		Operation:     "upload",
		RiskLevel:     "high",
		FilePath:      "C:/a.xlsx",
		FileHash:      "hash-a",
		Timestamp:     1000,
	}
	second := first
	second.EventID = "evt-2"
	second.FilePath = "C:/b.xlsx"
	second.FileHash = "hash-b"
	second.Timestamp = 1030

	merged := deduper.Add(first)
	if merged.IsMergeEvent {
		t.Fatal("single event should not be marked as merge")
	}

	merged = deduper.Add(second)
	if !merged.IsMergeEvent {
		t.Fatal("second event inside window should produce merge event")
	}
	if merged.FileCount != 2 {
		t.Fatalf("FileCount = %d, want 2", merged.FileCount)
	}
	if len(merged.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(merged.Files))
	}
	if merged.TimeRange != "1000-1030" {
		t.Fatalf("TimeRange = %q, want 1000-1030", merged.TimeRange)
	}
}

func TestDeduperDoesNotMergeDifferentKeyOrOutsideWindow(t *testing.T) {
	deduper := pipeline.NewDeduper(map[string]int{
		"high": 60,
	})

	first := model.Event{
		EventID:       "evt-1",
		HostID:        "host-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "客户资料",
		Operation:     "upload",
		RiskLevel:     "high",
		Timestamp:     1000,
	}
	deduper.Add(first)

	differentKey := first
	differentKey.EventID = "evt-2"
	differentKey.ProcessName = "outlook.exe"
	differentKey.Timestamp = 1010
	if merged := deduper.Add(differentKey); merged.IsMergeEvent {
		t.Fatal("event with different process_name should not merge")
	}

	outsideWindow := first
	outsideWindow.EventID = "evt-3"
	outsideWindow.Timestamp = 1100
	if merged := deduper.Add(outsideWindow); merged.IsMergeEvent {
		t.Fatal("event outside risk window should not merge")
	}
}

func TestDeduperCanAcceptConcurrentEvents(t *testing.T) {
	deduper := pipeline.NewDeduper(map[string]int{"high": 60})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deduper.Add(model.Event{
				EventID:       "evt",
				HostID:        "host-1",
				UserID:        "user-1",
				ProcessName:   "chrome.exe",
				SensitiveType: "客户资料",
				Operation:     "upload",
				RiskLevel:     "high",
				Timestamp:     int64(1000 + i),
			})
		}(i)
	}
	wg.Wait()
}

func TestDeduperStartsNewGroupAfterWindowExpires(t *testing.T) {
	deduper := pipeline.NewDeduper(map[string]int{"high": 60})

	first := model.Event{
		EventID:       "evt-1",
		HostID:        "host-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "customer",
		Operation:     "upload",
		RiskLevel:     "high",
		FilePath:      "C:/a.xlsx",
		FileHash:      "hash-a",
		Timestamp:     1000,
	}
	second := first
	second.EventID = "evt-2"
	second.Timestamp = 1100

	_ = deduper.Add(first)
	merged := deduper.Add(second)
	if merged.IsMergeEvent {
		t.Fatal("event outside the window should start a new group")
	}
	if merged.FileCount != 0 {
		t.Fatalf("FileCount = %d, want 0 before downstream normalization", merged.FileCount)
	}
}

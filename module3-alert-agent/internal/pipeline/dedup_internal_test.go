package pipeline

import (
	"testing"
	"time"

	"module3-alert-agent/internal/model"
)

func TestDeduperPurgesExpiredGroupsForDifferentKeys(t *testing.T) {
	deduper := NewDeduper(map[string]int{"high": 60})

	expired := model.Event{
		EventID:       "evt-1",
		HostID:        "host-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "customer",
		Operation:     "upload",
		RiskLevel:     "high",
		Timestamp:     1000,
	}
	fresh := expired
	fresh.EventID = "evt-2"
	fresh.ProcessName = "outlook.exe"
	fresh.Timestamp = 2000

	_ = deduper.Add(expired)
	_ = deduper.Add(fresh)

	if got := len(deduper.groups); got != 1 {
		t.Fatalf("len(groups) = %d, want 1 after purging expired groups", got)
	}
	if _, ok := deduper.groups[dedupKey(expired)]; ok {
		t.Fatal("expired group still present after later event advanced time")
	}
}

func TestDeduperStartCleanerPurgesExpiredGroupsWithoutNewEvents(t *testing.T) {
	deduper := NewDeduper(map[string]int{"high": 1})
	_ = deduper.Add(model.Event{
		EventID:       "evt-1",
		HostID:        "host-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "customer",
		Operation:     "upload",
		RiskLevel:     "high",
		Timestamp:     1,
	})

	stop := deduper.StartCleaner(5*time.Millisecond, func() int64 { return 10 })
	defer stop()

	deadline := time.After(500 * time.Millisecond)
	for {
		deduper.mu.Lock()
		got := len(deduper.groups)
		deduper.mu.Unlock()
		if got == 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("deduper cleaner did not purge expired group")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

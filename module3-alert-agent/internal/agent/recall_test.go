package agent_test

import (
	"testing"
	"time"

	"module3-alert-agent/internal/agent"
	"module3-alert-agent/internal/model"
)

func TestScenarioKeyOmitsUserAndNormalizesTarget(t *testing.T) {
	event := model.Event{
		UserID:        "alice",
		SensitiveType: "Customer",
		Operation:     "UPLOAD",
		ProcessName:   "Chrome.EXE",
		Target:        "HTTPS://Internal-CRM.Company.Com/upload?id=42",
	}

	key := agent.ScenarioKey(event)

	if key != "customer|upload|chrome.exe|internal-crm.company.com" {
		t.Fatalf("ScenarioKey = %q, want normalized scenario key without user_id", key)
	}
}

func TestStructuredRecallScoresAndSortsActiveFalsePositivePatterns(t *testing.T) {
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	event := model.Event{
		UserID:        "alice",
		SensitiveType: "customer",
		Operation:     "upload",
		ProcessName:   "chrome.exe",
		ProcessPath:   "C:/Chrome/chrome.exe",
		Target:        "internal-crm.company.com",
	}
	records := []model.FalsePositiveRecord{
		{
			ID:            1,
			UserID:        "alice",
			SensitiveType: "customer",
			Operation:     "upload",
			ProcessName:   "chrome.exe",
			ProcessPath:   "C:/Chrome/chrome.exe",
			Target:        "internal-crm.company.com",
			Reason:        "normal crm upload",
			ExpiredAt:     now.Add(24 * time.Hour),
		},
		{
			ID:            2,
			SensitiveType: "finance",
			Operation:     "send",
			ProcessName:   "outlook.exe",
			Target:        "mail.example.com",
			ExpiredAt:     now.Add(24 * time.Hour),
		},
		{
			ID:            3,
			UserID:        "alice",
			SensitiveType: "customer",
			Operation:     "upload",
			ProcessName:   "chrome.exe",
			Target:        "internal-crm.company.com",
			ExpiredAt:     now.Add(-time.Hour),
		},
	}

	recalls := agent.StructuredRecall(event, records, agent.StructuredRecallConfig{
		TopK:      5,
		Threshold: 0.60,
		Now:       func() time.Time { return now },
	})

	if len(recalls) != 1 {
		t.Fatalf("len(recalls) = %d, want only the active matching pattern", len(recalls))
	}
	if recalls[0].Record.ID != 1 {
		t.Fatalf("top recall ID = %d, want 1", recalls[0].Record.ID)
	}
	if recalls[0].Score != 1 {
		t.Fatalf("Score = %v, want exact weighted match score 1", recalls[0].Score)
	}
}

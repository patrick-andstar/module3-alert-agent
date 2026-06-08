package agent_test

import (
	"testing"
	"time"

	"module3-alert-agent/internal/agent"
	"module3-alert-agent/internal/model"
)

func TestApplyDecisionStoresScenarioKeyWhenPatternIsConfirmed(t *testing.T) {
	store := &recordingFalsePositiveStore{}
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	event := model.Event{
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "customer",
		Operation:     "upload",
		Target:        "crm.internal",
		RiskLevel:     "high",
	}

	_, err := agent.ApplyDecision(event, agent.Decision{
		IsFalsePositive:     true,
		NewRiskLevel:        "info",
		FalsePositiveReason: "正常业务",
		Confidence:          0.9,
	}, store, agent.DecisionConfig{
		ConfidenceThreshold:   0.8,
		StrongRecallThreshold: 0.75,
		FalsePositiveTTLDays:  30,
		Now:                   func() time.Time { return now },
		HasRecall:             true,
		RecallScore:           0.8,
	})
	if err != nil {
		t.Fatalf("ApplyDecision returned error: %v", err)
	}

	if len(store.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(store.writes))
	}
	if store.writes[0].ScenarioKey != "customer|upload|chrome.exe|crm.internal" {
		t.Fatalf("ScenarioKey = %q, want dedupe key", store.writes[0].ScenarioKey)
	}
}

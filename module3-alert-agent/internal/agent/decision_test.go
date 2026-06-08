package agent_test

import (
	"testing"
	"time"

	"module3-alert-agent/internal/agent"
	"module3-alert-agent/internal/model"
)

type recordingFalsePositiveStore struct {
	writes []model.FalsePositiveRecord
}

func (s *recordingFalsePositiveStore) Save(record model.FalsePositiveRecord) error {
	s.writes = append(s.writes, record)
	return nil
}

func TestApplyDecisionWritesFalsePositiveOnlyWithStrongRecallAndConfidence(t *testing.T) {
	store := &recordingFalsePositiveStore{}
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	event := model.Event{
		HostID:        "host-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "customer",
		Operation:     "upload",
		RiskLevel:     "high",
		Target:        "crm.internal",
	}
	decision := agent.Decision{
		IsFalsePositive:     true,
		NewRiskLevel:        "info",
		FalsePositiveReason: "normal crm upload",
		Confidence:          0.8,
		AgentVerdict:        "false_positive",
		Explanation:         "matched crm pattern",
	}

	output, err := agent.ApplyDecision(event, decision, store, agent.DecisionConfig{
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

	if output.RiskLevel != "info" {
		t.Fatalf("RiskLevel = %q, want info", output.RiskLevel)
	}
	if output.OldRiskLevel != "high" {
		t.Fatalf("OldRiskLevel = %q, want high", output.OldRiskLevel)
	}
	if output.FalsePositiveReason != decision.FalsePositiveReason {
		t.Fatalf("FalsePositiveReason = %q, want propagated reason", output.FalsePositiveReason)
	}
	if output.AgentVerdict != "false_positive" || output.AgentConfidence != 0.8 || output.RecallScore != 0.8 {
		t.Fatalf("agent fields = (%q,%v,%v), want false_positive/0.8/0.8", output.AgentVerdict, output.AgentConfidence, output.RecallScore)
	}
	if len(store.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(store.writes))
	}
	if store.writes[0].ScenarioKey == "" {
		t.Fatal("ScenarioKey is empty, want dedupe key on stored false-positive pattern")
	}
	if store.writes[0].ExpiredAt != now.Add(30*24*time.Hour) {
		t.Fatalf("ExpiredAt = %v, want %v", store.writes[0].ExpiredAt, now.Add(30*24*time.Hour))
	}
}

func TestApplyDecisionBelowConfidenceBecomesUncertainAndDowngradesAtMostOneLevel(t *testing.T) {
	store := &recordingFalsePositiveStore{}
	event := model.Event{RiskLevel: "high"}
	decision := agent.Decision{
		IsFalsePositive:     true,
		NewRiskLevel:        "low",
		FalsePositiveReason: "suspected normal flow",
		Confidence:          0.79,
	}

	output, err := agent.ApplyDecision(event, decision, store, agent.DecisionConfig{
		ConfidenceThreshold:   0.8,
		StrongRecallThreshold: 0.75,
		FalsePositiveTTLDays:  30,
		HasRecall:             true,
		RecallScore:           0.7,
	})
	if err != nil {
		t.Fatalf("ApplyDecision returned error: %v", err)
	}

	if output.RiskLevel != "medium" {
		t.Fatalf("RiskLevel = %q, want medium after one-level downgrade cap", output.RiskLevel)
	}
	if output.OldRiskLevel != "high" {
		t.Fatalf("OldRiskLevel = %q, want original value preserved", output.OldRiskLevel)
	}
	if output.AgentVerdict != "uncertain" {
		t.Fatalf("AgentVerdict = %q, want uncertain", output.AgentVerdict)
	}
	if len(store.writes) != 0 {
		t.Fatalf("writes = %d, want 0 below threshold", len(store.writes))
	}
}

func TestApplyDecisionWithoutRecallDowngradesAtMostTwoLevelsAndDoesNotWritePattern(t *testing.T) {
	store := &recordingFalsePositiveStore{}
	event := model.Event{RiskLevel: "critical"}
	decision := agent.Decision{
		IsFalsePositive:     true,
		NewRiskLevel:        "info",
		FalsePositiveReason: "standalone judgement",
		Confidence:          0.95,
		AgentVerdict:        "false_positive",
	}

	output, err := agent.ApplyDecision(event, decision, store, agent.DecisionConfig{
		ConfidenceThreshold:   0.8,
		StrongRecallThreshold: 0.75,
		FalsePositiveTTLDays:  30,
		HasRecall:             false,
		RecallScore:           0,
	})
	if err != nil {
		t.Fatalf("ApplyDecision returned error: %v", err)
	}

	if output.RiskLevel != "medium" {
		t.Fatalf("RiskLevel = %q, want medium after two-level no-recall cap", output.RiskLevel)
	}
	if output.AgentVerdict != "uncertain" {
		t.Fatalf("AgentVerdict = %q, want uncertain without recall evidence", output.AgentVerdict)
	}
	if len(store.writes) != 0 {
		t.Fatalf("writes = %d, want 0 without recall", len(store.writes))
	}
}

func TestApplyDecisionTrueAlertPreservesOriginalRiskAndDoesNotWritePattern(t *testing.T) {
	store := &recordingFalsePositiveStore{}
	event := model.Event{RiskLevel: "high"}
	decision := agent.Decision{
		IsFalsePositive: false,
		NewRiskLevel:    "low",
		Confidence:      0.93,
		AgentVerdict:    "true_alert",
		Explanation:     "external target remains risky",
	}

	output, err := agent.ApplyDecision(event, decision, store, agent.DecisionConfig{
		ConfidenceThreshold:   0.8,
		StrongRecallThreshold: 0.75,
		FalsePositiveTTLDays:  30,
		HasRecall:             true,
		RecallScore:           0.8,
	})
	if err != nil {
		t.Fatalf("ApplyDecision returned error: %v", err)
	}

	if output.RiskLevel != "high" {
		t.Fatalf("RiskLevel = %q, want original high", output.RiskLevel)
	}
	if output.OldRiskLevel != "" {
		t.Fatalf("OldRiskLevel = %q, want empty when preserving original risk", output.OldRiskLevel)
	}
	if output.AgentVerdict != "true_alert" || output.AgentExplanation == "" {
		t.Fatalf("agent fields = (%q,%q), want true_alert with explanation", output.AgentVerdict, output.AgentExplanation)
	}
	if len(store.writes) != 0 {
		t.Fatalf("writes = %d, want 0 for true alert", len(store.writes))
	}
}

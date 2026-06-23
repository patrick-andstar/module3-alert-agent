package agent

import (
	"time"

	"module3-alert-agent/internal/model"
)

type Decision struct {
	EventID             string  `json:"event_id"`
	IsFalsePositive     bool    `json:"is_false_positive"`
	NewRiskLevel        string  `json:"new_risk_level"`
	FalsePositiveReason string  `json:"false_positive_reason"`
	Confidence          float64 `json:"confidence"`
	AgentVerdict        string  `json:"agent_verdict"`
	Explanation         string  `json:"explanation"`
}

type DecisionConfig struct {
	ConfidenceThreshold   float64
	StrongRecallThreshold float64
	FalsePositiveTTLDays  int
	Now                   func() time.Time
	HasRecall             bool
	RecallScore           float64
}

type FalsePositiveStore interface {
	Save(model.FalsePositiveRecord) error
}

func ApplyDecision(event model.Event, decision Decision, store FalsePositiveStore, cfg DecisionConfig) (model.Event, error) {
	output := event
	output.FileCount = maxFileCount(output.FileCount)
	output.AgentConfidence = decision.Confidence
	output.AgentExplanation = decision.Explanation
	output.RecallScore = cfg.RecallScore

	verdict := normalizedVerdict(decision)
	if verdict == "true_alert" {
		output.AgentVerdict = "true_alert"
		return output, nil
	}

	if decision.FalsePositiveReason != "" {
		output.FalsePositiveReason = decision.FalsePositiveReason
	}

	now := time.Now()
	if cfg.Now != nil {
		now = cfg.Now()
	}

	strongRecallThreshold := cfg.StrongRecallThreshold
	if strongRecallThreshold <= 0 {
		strongRecallThreshold = 0.75
	}
	confidenceThreshold := cfg.ConfidenceThreshold
	if confidenceThreshold <= 0 {
		confidenceThreshold = 0.8
	}

	canWritePattern := cfg.HasRecall &&
		cfg.RecallScore >= strongRecallThreshold &&
		decision.Confidence >= confidenceThreshold &&
		decision.FalsePositiveReason != "" &&
		store != nil

	if canWritePattern {
		output.AgentVerdict = "false_positive"
		targetRisk := decision.NewRiskLevel
		if !model.ValidRiskLevel(targetRisk) {
			targetRisk = string(model.RiskInfo)
		}
		applyRiskLevel(&output, targetRisk, 4, true)
	} else {
		output.AgentVerdict = "uncertain"
		maxDowngrade := 1
		if !cfg.HasRecall && decision.Confidence >= confidenceThreshold {
			maxDowngrade = 2
		}
		applyRiskLevel(&output, decision.NewRiskLevel, maxDowngrade, false)
		return output, nil
	}

	scenarioKey := ScenarioKey(event)
	if scenarioKey == "" {
		return output, nil
	}

	record := model.FalsePositiveRecord{
		ScenarioKey:   scenarioKey,
		HostID:        event.HostID,
		UserID:        event.UserID,
		SensitiveType: event.SensitiveType,
		RiskLevel:     model.NormalizeRiskLevel(event.RiskLevel),
		ProcessName:   event.ProcessName,
		ProcessPath:   event.ProcessPath,
		Target:        event.Target,
		Operation:     event.Operation,
		Reason:        decision.FalsePositiveReason,
		HitCount:      1,
		LastSeenAt:    now,
		ExpiredAt:     now.Add(time.Duration(cfg.FalsePositiveTTLDays) * 24 * time.Hour),
		CreatedAt:     now,
	}
	return output, store.Save(record)
}

func normalizedVerdict(decision Decision) string {
	switch decision.AgentVerdict {
	case "false_positive", "true_alert", "uncertain":
		return decision.AgentVerdict
	}
	if decision.IsFalsePositive {
		return "false_positive"
	}
	return "true_alert"
}

func applyRiskLevel(event *model.Event, requested string, maxDowngrade int, allowInfo bool) {
	target := model.NormalizeRiskLevel(requested)
	if target == "" {
		return
	}
	currentRank, ok := riskRank(event.RiskLevel)
	if !ok {
		return
	}
	targetRank, ok := riskRank(target)
	if !ok || targetRank <= currentRank {
		return
	}
	if !allowInfo && target == string(model.RiskInfo) {
		targetRank = riskRankMust(string(model.RiskLow))
		target = string(model.RiskLow)
	}
	maxRank := currentRank + maxDowngrade
	lowRank := riskRankMust(string(model.RiskLow))
	if !allowInfo && maxRank > lowRank {
		maxRank = lowRank
	}
	if targetRank > maxRank {
		targetRank = maxRank
		target = riskLevelAt(targetRank)
	}
	if target != "" && target != event.RiskLevel {
		event.OldRiskLevel = event.RiskLevel
		event.RiskLevel = target
	}
}

func riskRank(level string) (int, bool) {
	switch level {
	case string(model.RiskCritical):
		return 0, true
	case string(model.RiskHigh):
		return 1, true
	case string(model.RiskMedium):
		return 2, true
	case string(model.RiskLow):
		return 3, true
	case string(model.RiskInfo):
		return 4, true
	default:
		return 0, false
	}
}

func riskRankMust(level string) int {
	rank, _ := riskRank(level)
	return rank
}

func riskLevelAt(rank int) string {
	switch rank {
	case 0:
		return string(model.RiskCritical)
	case 1:
		return string(model.RiskHigh)
	case 2:
		return string(model.RiskMedium)
	case 3:
		return string(model.RiskLow)
	case 4:
		return string(model.RiskInfo)
	default:
		return ""
	}
}

func maxFileCount(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

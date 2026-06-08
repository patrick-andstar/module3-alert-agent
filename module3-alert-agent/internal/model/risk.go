package model

type RiskLevel string

const (
	RiskCritical RiskLevel = "critical"
	RiskHigh     RiskLevel = "high"
	RiskMedium   RiskLevel = "medium"
	RiskLow      RiskLevel = "low"
	RiskInfo     RiskLevel = "info"
)

func ValidRiskLevel(level string) bool {
	switch RiskLevel(level) {
	case RiskCritical, RiskHigh, RiskMedium, RiskLow, RiskInfo:
		return true
	default:
		return false
	}
}

func NormalizeRiskLevel(level string) string {
	if ValidRiskLevel(level) {
		return level
	}
	return ""
}

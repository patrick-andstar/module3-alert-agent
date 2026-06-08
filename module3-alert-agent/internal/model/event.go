package model

type Event struct {
	EventID             string     `json:"event_id"`
	HostID              string     `json:"host_id"`
	UserID              string     `json:"user_id"`
	FilePath            string     `json:"file_path"`
	FileHash            string     `json:"file_hash"`
	Sensitive           bool       `json:"sensitive"`
	SensitiveType       string     `json:"sensitive_type"`
	RiskLevel           string     `json:"risk_level"`
	OldRiskLevel        string     `json:"old_risk_level,omitempty"`
	ProcessName         string     `json:"process_name"`
	ProcessPath         string     `json:"process_path"`
	Target              string     `json:"target"`
	Operation           string     `json:"operation"`
	Timestamp           int64      `json:"timestamp"`
	SensitiveFileID     string     `json:"sensitive_file_id"`
	IsMergeEvent        bool       `json:"is_merge_event"`
	FileCount           int        `json:"file_count"`
	Files               []FileInfo `json:"files,omitempty"`
	FalsePositiveReason string     `json:"false_positive_reason,omitempty"`
	AgentVerdict        string     `json:"agent_verdict,omitempty"`
	AgentConfidence     float64    `json:"agent_confidence,omitempty"`
	AgentExplanation    string     `json:"agent_explanation,omitempty"`
	RecallScore         float64    `json:"recall_score,omitempty"`
	TimeRange           string     `json:"time_range,omitempty"`
	Duration            string     `json:"duration,omitempty"`
}

type FileInfo struct {
	FilePath string `json:"file_path"`
	FileHash string `json:"file_hash"`
}

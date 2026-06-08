package model

import "time"

type FalsePositiveRecord struct {
	ID            int64     `json:"id"`
	ScenarioKey   string    `json:"scenario_key"`
	HostID        string    `json:"host_id"`
	UserID        string    `json:"user_id"`
	SensitiveType string    `json:"sensitive_type"`
	RiskLevel     string    `json:"risk_level"`
	ProcessName   string    `json:"process_name"`
	ProcessPath   string    `json:"process_path"`
	Target        string    `json:"target"`
	Operation     string    `json:"operation"`
	Reason        string    `json:"reason"`
	Embedding     []float64 `json:"embedding"`
	HitCount      int       `json:"hit_count"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	ExpiredAt     time.Time `json:"expired_at"`
	CreatedAt     time.Time `json:"created_at"`
}

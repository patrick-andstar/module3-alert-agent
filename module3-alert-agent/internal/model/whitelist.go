package model

type WhitelistRule struct {
	ID              int64  `json:"id"`
	RuleName        string `json:"rule_name"`
	Logic           string `json:"logic"`
	ProcessName     string `json:"process_name"`
	UserID          string `json:"user_id"`
	FilePathPattern string `json:"file_path_pattern"`
	TimeWindowStart string `json:"time_window_start,omitempty"`
	TimeWindowEnd   string `json:"time_window_end,omitempty"`
	Enabled         bool   `json:"enabled"`
}

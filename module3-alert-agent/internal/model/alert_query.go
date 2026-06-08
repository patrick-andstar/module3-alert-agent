package model

const MaxAlertQueryPageSize = 100

type AlertQuery struct {
	EventID       string `json:"event_id"`
	StartTime     int64  `json:"start_timestamp"`
	EndTime       int64  `json:"end_timestamp"`
	RiskLevel     string `json:"risk_level"`
	UserID        string `json:"user_id"`
	SensitiveType string `json:"sensitive_type"`
	ProcessName   string `json:"process_name"`
	Operation     string `json:"operation"`
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
	OrderBy       string `json:"order_by"`
	Order         string `json:"order"`
}

func NormalizeAlertQuery(query AlertQuery) AlertQuery {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > MaxAlertQueryPageSize {
		query.PageSize = MaxAlertQueryPageSize
	}
	if query.OrderBy == "" {
		query.OrderBy = "timestamp"
	}
	if query.Order != "asc" {
		query.Order = "desc"
	}
	return query
}

func MatchesAlertQuery(event Event, query AlertQuery) bool {
	if query.EventID != "" && event.EventID != query.EventID {
		return false
	}
	if query.StartTime > 0 && event.Timestamp < query.StartTime {
		return false
	}
	if query.EndTime > 0 && event.Timestamp > query.EndTime {
		return false
	}
	if query.RiskLevel != "" && event.RiskLevel != query.RiskLevel {
		return false
	}
	if query.UserID != "" && event.UserID != query.UserID {
		return false
	}
	if query.SensitiveType != "" && event.SensitiveType != query.SensitiveType {
		return false
	}
	if query.ProcessName != "" && event.ProcessName != query.ProcessName {
		return false
	}
	if query.Operation != "" && event.Operation != query.Operation {
		return false
	}
	return true
}

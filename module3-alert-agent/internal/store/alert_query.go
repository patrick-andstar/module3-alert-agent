package store

import (
	"strings"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/router"
)

func BuildAlertQuerySQL(query router.AlertQuery) (string, []any) {
	query = model.NormalizeAlertQuery(query)
	where, args := buildAlertWhere(query)

	orderBy := "timestamp"
	if allowedAlertOrderBy(query.OrderBy) {
		orderBy = query.OrderBy
	}
	order := "DESC"
	if query.Order == "asc" {
		order = "ASC"
	}

	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := "SELECT event_id, COALESCE(host_id, ''), COALESCE(user_id, ''), COALESCE(file_path, ''), COALESCE(file_hash, ''), `sensitive`, COALESCE(sensitive_type, ''), COALESCE(risk_level, ''), old_risk_level, COALESCE(process_name, ''), COALESCE(process_path, ''), COALESCE(target, ''), COALESCE(operation, ''), COALESCE(sensitive_file_id, ''), is_merge_event, file_count, COALESCE(files_json, '[]'), COALESCE(false_positive_reason, ''), agent_verdict, agent_confidence, agent_explanation, recall_score, timestamp FROM alert_logs WHERE " +
		where +
		" ORDER BY " + orderBy + " " + order +
		" LIMIT ? OFFSET ?"
	return sql, args
}

func allowedAlertOrderBy(value string) bool {
	switch value {
	case "timestamp", "event_id", "created_at":
		return true
	default:
		return false
	}
}

func BuildAlertCountSQL(query router.AlertQuery) (string, []any) {
	query = model.NormalizeAlertQuery(query)
	where, args := buildAlertWhere(query)
	return "SELECT COUNT(*) FROM alert_logs WHERE " + where, args
}

func buildAlertWhere(query router.AlertQuery) (string, []any) {
	clauses := []string{"1=1"}
	args := []any{}
	if query.EventID != "" {
		clauses = append(clauses, "event_id = ?")
		args = append(args, query.EventID)
	}
	if query.StartTime > 0 {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, query.StartTime)
	}
	if query.EndTime > 0 {
		clauses = append(clauses, "timestamp <= ?")
		args = append(args, query.EndTime)
	}
	if query.RiskLevel != "" {
		clauses = append(clauses, "risk_level = ?")
		args = append(args, query.RiskLevel)
	}
	if query.UserID != "" {
		clauses = append(clauses, "user_id = ?")
		args = append(args, query.UserID)
	}
	if query.SensitiveType != "" {
		clauses = append(clauses, "sensitive_type = ?")
		args = append(args, query.SensitiveType)
	}
	if query.ProcessName != "" {
		clauses = append(clauses, "process_name = ?")
		args = append(args, query.ProcessName)
	}
	if query.Operation != "" {
		clauses = append(clauses, "operation = ?")
		args = append(args, query.Operation)
	}
	return strings.Join(clauses, " AND "), args
}

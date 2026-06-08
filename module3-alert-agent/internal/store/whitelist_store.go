package store

import "module3-alert-agent/internal/model"

func BuildWhitelistInsertSQL() string {
	return `INSERT INTO whitelist_rules (rule_name, logic, process_name, user_id, file_path_pattern, time_window_start, time_window_end, enabled)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
}

func BuildWhitelistUpdateSQL() string {
	return `UPDATE whitelist_rules
SET rule_name = ?, logic = ?, process_name = ?, user_id = ?, file_path_pattern = ?, time_window_start = ?, time_window_end = ?, enabled = ?
WHERE id = ?`
}

func BuildWhitelistSelectSQL(enabledOnly bool) string {
	sql := `SELECT id, rule_name, logic, COALESCE(process_name, ''), COALESCE(user_id, ''), COALESCE(file_path_pattern, ''), COALESCE(TIME_FORMAT(time_window_start, '%H:%i:%s'), ''), COALESCE(TIME_FORMAT(time_window_end, '%H:%i:%s'), ''), enabled
FROM whitelist_rules`
	if enabledOnly {
		sql += "\nWHERE enabled = TRUE"
	}
	return sql + "\nORDER BY id ASC"
}

func WhitelistInsertArgs(rule model.WhitelistRule) []any {
	rule = normalizeWhitelistRule(rule)
	return []any{
		rule.RuleName,
		rule.Logic,
		rule.ProcessName,
		rule.UserID,
		rule.FilePathPattern,
		rule.TimeWindowStart,
		rule.TimeWindowEnd,
		rule.Enabled,
	}
}

func WhitelistUpdateArgs(id int64, rule model.WhitelistRule) []any {
	args := WhitelistInsertArgs(rule)
	return append(args, id)
}

func normalizeWhitelistRule(rule model.WhitelistRule) model.WhitelistRule {
	if rule.Logic == "" {
		rule.Logic = "OR"
	}
	return rule
}

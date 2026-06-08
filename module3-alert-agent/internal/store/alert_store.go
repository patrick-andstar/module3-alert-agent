package store

import (
	"database/sql"
	"encoding/json"

	"module3-alert-agent/internal/model"
)

func BuildAlertUpsertSQL() string {
	return "INSERT INTO alert_logs (\n" +
		"event_id, host_id, user_id, file_path, file_hash, `sensitive`, sensitive_type, risk_level,\n" +
		"old_risk_level, process_name, process_path, target, operation, sensitive_file_id, is_merge_event,\n" +
		"file_count, files_json, false_positive_reason, agent_verdict, agent_confidence, agent_explanation, recall_score, timestamp\n" +
		") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)\n" +
		"ON DUPLICATE KEY UPDATE\n" +
		"host_id = VALUES(host_id),\n" +
		"user_id = VALUES(user_id),\n" +
		"file_path = VALUES(file_path),\n" +
		"file_hash = VALUES(file_hash),\n" +
		"`sensitive` = VALUES(`sensitive`),\n" +
		"sensitive_type = VALUES(sensitive_type),\n" +
		"risk_level = VALUES(risk_level),\n" +
		"old_risk_level = VALUES(old_risk_level),\n" +
		"process_name = VALUES(process_name),\n" +
		"process_path = VALUES(process_path),\n" +
		"target = VALUES(target),\n" +
		"operation = VALUES(operation),\n" +
		"sensitive_file_id = VALUES(sensitive_file_id),\n" +
		"is_merge_event = VALUES(is_merge_event),\n" +
		"file_count = VALUES(file_count),\n" +
		"files_json = VALUES(files_json),\n" +
		"false_positive_reason = VALUES(false_positive_reason),\n" +
		"agent_verdict = VALUES(agent_verdict),\n" +
		"agent_confidence = VALUES(agent_confidence),\n" +
		"agent_explanation = VALUES(agent_explanation),\n" +
		"recall_score = VALUES(recall_score),\n" +
		"timestamp = VALUES(timestamp)"
}

func AlertUpsertArgs(event model.Event) []any {
	fileCount := event.FileCount
	if fileCount <= 0 {
		fileCount = 1
	}
	filesJSON := "[]"
	if len(event.Files) > 0 {
		if data, err := json.Marshal(event.Files); err == nil {
			filesJSON = string(data)
		}
	}
	return []any{
		event.EventID,
		event.HostID,
		event.UserID,
		event.FilePath,
		event.FileHash,
		event.Sensitive,
		event.SensitiveType,
		event.RiskLevel,
		nullIfEmpty(event.OldRiskLevel),
		event.ProcessName,
		event.ProcessPath,
		event.Target,
		event.Operation,
		event.SensitiveFileID,
		event.IsMergeEvent,
		fileCount,
		filesJSON,
		event.FalsePositiveReason,
		event.AgentVerdict,
		event.AgentConfidence,
		event.AgentExplanation,
		event.RecallScore,
		event.Timestamp,
	}
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func decodeFilesJSON(value string) ([]model.FileInfo, error) {
	if value == "" {
		return nil, nil
	}
	var files []model.FileInfo
	if err := json.Unmarshal([]byte(value), &files); err != nil {
		return nil, err
	}
	return files, nil
}

func scanAlertEvent(scanner interface {
	Scan(dest ...any) error
}) (model.Event, error) {
	var event model.Event
	var oldRiskLevel sql.NullString
	var filesJSON sql.NullString
	var falsePositiveReason sql.NullString
	var agentVerdict sql.NullString
	var agentConfidence sql.NullFloat64
	var agentExplanation sql.NullString
	var recallScore sql.NullFloat64
	err := scanner.Scan(
		&event.EventID,
		&event.HostID,
		&event.UserID,
		&event.FilePath,
		&event.FileHash,
		&event.Sensitive,
		&event.SensitiveType,
		&event.RiskLevel,
		&oldRiskLevel,
		&event.ProcessName,
		&event.ProcessPath,
		&event.Target,
		&event.Operation,
		&event.SensitiveFileID,
		&event.IsMergeEvent,
		&event.FileCount,
		&filesJSON,
		&falsePositiveReason,
		&agentVerdict,
		&agentConfidence,
		&agentExplanation,
		&recallScore,
		&event.Timestamp,
	)
	if err != nil {
		return event, err
	}
	if oldRiskLevel.Valid {
		event.OldRiskLevel = oldRiskLevel.String
	}
	if falsePositiveReason.Valid {
		event.FalsePositiveReason = falsePositiveReason.String
	}
	if agentVerdict.Valid {
		event.AgentVerdict = agentVerdict.String
	}
	if agentConfidence.Valid {
		event.AgentConfidence = agentConfidence.Float64
	}
	if agentExplanation.Valid {
		event.AgentExplanation = agentExplanation.String
	}
	if recallScore.Valid {
		event.RecallScore = recallScore.Float64
	}
	if filesJSON.Valid {
		event.Files, err = decodeFilesJSON(filesJSON.String)
		if err != nil {
			return event, err
		}
	}
	return event, nil
}

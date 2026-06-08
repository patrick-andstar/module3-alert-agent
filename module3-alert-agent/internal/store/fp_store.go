package store

import (
	"encoding/json"
	"time"

	"module3-alert-agent/internal/model"
)

func BuildFalsePositiveInsertSQL() string {
	return `INSERT INTO false_positive_library (
scenario_key, host_id, user_id, sensitive_type, risk_level, process_name, process_path, target, operation, reason, embedding_json, hit_count, last_seen_at, expired_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
host_id = VALUES(host_id),
user_id = VALUES(user_id),
sensitive_type = VALUES(sensitive_type),
risk_level = VALUES(risk_level),
process_name = VALUES(process_name),
process_path = VALUES(process_path),
target = VALUES(target),
operation = VALUES(operation),
reason = VALUES(reason),
embedding_json = VALUES(embedding_json),
hit_count = hit_count + VALUES(hit_count),
last_seen_at = VALUES(last_seen_at),
expired_at = VALUES(expired_at)`
}

func FalsePositiveInsertArgs(record model.FalsePositiveRecord) ([]any, error) {
	embedding, err := json.Marshal(record.Embedding)
	if err != nil {
		return nil, err
	}
	hitCount := record.HitCount
	if hitCount <= 0 {
		hitCount = 1
	}
	return []any{
		record.ScenarioKey,
		record.HostID,
		record.UserID,
		record.SensitiveType,
		model.NormalizeRiskLevel(record.RiskLevel),
		record.ProcessName,
		record.ProcessPath,
		record.Target,
		record.Operation,
		record.Reason,
		string(embedding),
		hitCount,
		record.LastSeenAt,
		record.ExpiredAt,
	}, nil
}

func DecodeEmbeddingJSON(value string) ([]float64, error) {
	var embedding []float64
	if err := json.Unmarshal([]byte(value), &embedding); err != nil {
		return nil, err
	}
	return embedding, nil
}

func scanFalsePositiveRecord(scanner interface {
	Scan(dest ...any) error
}) (model.FalsePositiveRecord, error) {
	var record model.FalsePositiveRecord
	var embeddingJSON string
	var createdAt time.Time
	err := scanner.Scan(
		&record.ID,
		&record.ScenarioKey,
		&record.HostID,
		&record.UserID,
		&record.SensitiveType,
		&record.RiskLevel,
		&record.ProcessName,
		&record.ProcessPath,
		&record.Target,
		&record.Operation,
		&record.Reason,
		&embeddingJSON,
		&record.HitCount,
		&record.LastSeenAt,
		&record.ExpiredAt,
		&createdAt,
	)
	if err != nil {
		return record, err
	}
	record.CreatedAt = createdAt
	record.Embedding, err = DecodeEmbeddingJSON(embeddingJSON)
	if err != nil {
		return record, err
	}
	return record, nil
}

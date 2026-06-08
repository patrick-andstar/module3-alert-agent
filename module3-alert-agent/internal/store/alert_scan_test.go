package store

import (
	"testing"
)

type scanStub struct {
	values []any
}

func (s scanStub) Scan(dest ...any) error {
	for i := range dest {
		switch target := dest[i].(type) {
		case *string:
			*target = s.values[i].(string)
		case *bool:
			*target = s.values[i].(bool)
		case *int:
			*target = s.values[i].(int)
		case *int64:
			*target = s.values[i].(int64)
		case *float64:
			*target = s.values[i].(float64)
		default:
			switch typed := target.(type) {
			case interface{ Scan(any) error }:
				if err := typed.Scan(s.values[i]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func TestScanAlertEventHydratesPersistedFields(t *testing.T) {
	event, err := scanAlertEvent(scanStub{values: []any{
		"evt-1",
		"host-1",
		"user-1",
		"C:/a.xlsx",
		"hash-a",
		true,
		"customer",
		"low",
		"high",
		"chrome.exe",
		"C:/chrome.exe",
		"crm.internal",
		"upload",
		"sf-1",
		true,
		2,
		`[{"file_path":"C:/a.xlsx","file_hash":"hash-a"},{"file_path":"C:/b.xlsx","file_hash":"hash-b"}]`,
		"normal crm upload",
		"false_positive",
		float64(0.91),
		"matched structured recall",
		float64(0.86),
		int64(123),
	}})
	if err != nil {
		t.Fatalf("scanAlertEvent returned error: %v", err)
	}
	if event.OldRiskLevel != "high" {
		t.Fatalf("OldRiskLevel = %q, want high", event.OldRiskLevel)
	}
	if event.FalsePositiveReason != "normal crm upload" {
		t.Fatalf("FalsePositiveReason = %q, want persisted reason", event.FalsePositiveReason)
	}
	if event.AgentVerdict != "false_positive" || event.AgentConfidence != 0.91 || event.AgentExplanation != "matched structured recall" || event.RecallScore != 0.86 {
		t.Fatalf("agent fields = (%q,%v,%q,%v), want persisted analysis fields", event.AgentVerdict, event.AgentConfidence, event.AgentExplanation, event.RecallScore)
	}
	if len(event.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(event.Files))
	}
}

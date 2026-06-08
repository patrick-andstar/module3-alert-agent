package router_test

import (
	"encoding/json"
	"strings"
	"testing"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/router"
)

func TestCreateFalsePositiveRecordForScenarioSetup(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithService(service)

	body := `{
		"scenario_key":"customer|upload|chrome.exe|internal-crm.company.com",
		"user_id":"alice",
		"sensitive_type":"customer",
		"process_name":"chrome.exe",
		"target":"internal-crm.company.com",
		"operation":"upload",
		"reason":"normal crm upload"
	}`
	resp := performRequest(h.Engine, "POST", "/api/false-positives", body)
	if resp.Code != 201 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	second := performRequest(h.Engine, "POST", "/api/false-positives", body)
	if second.Code != 201 {
		t.Fatalf("second status = %d, body = %s", second.Code, second.Body.String())
	}

	var created model.FalsePositiveRecord
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created record: %v", err)
	}
	if created.ScenarioKey == "" || created.HitCount != 1 || created.ExpiredAt.IsZero() {
		t.Fatalf("created record = %+v, want scenario key, hit count, and TTL", created)
	}

	list := performRequest(h.Engine, "GET", "/api/false-positives", "")
	if !strings.Contains(list.Body.String(), "normal crm upload") {
		t.Fatalf("list body = %s, want seeded false-positive pattern", list.Body.String())
	}
	var records []model.FalsePositiveRecord
	if err := json.Unmarshal(list.Body.Bytes(), &records); err != nil {
		t.Fatalf("decode records: %v", err)
	}
	if len(records) != 1 || records[0].HitCount != 2 {
		t.Fatalf("records = %+v, want one upserted pattern with hit_count 2", records)
	}
}

func TestCreateFalsePositiveRecordNormalizesURLTargetForScenarioKey(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithService(service)

	body := `{
		"sensitive_type":"Customer",
		"process_name":"Chrome.EXE",
		"target":"https://Internal-CRM.Company.Com/upload?id=42",
		"operation":"UPLOAD",
		"reason":"normal crm upload"
	}`
	resp := performRequest(h.Engine, "POST", "/api/false-positives", body)
	if resp.Code != 201 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var created model.FalsePositiveRecord
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created record: %v", err)
	}
	if created.ScenarioKey != "customer|upload|chrome.exe|internal-crm.company.com" {
		t.Fatalf("ScenarioKey = %q, want normalized host-only key", created.ScenarioKey)
	}
}

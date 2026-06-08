package router_test

import (
	"encoding/json"
	"testing"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/router"
)

func TestAlertsQueryFiltersSortsAndPaginates(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	service.AddAlert(model.Event{
		EventID:       "evt-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "客户资料",
		Operation:     "upload",
		RiskLevel:     "high",
		Timestamp:     1000,
	})
	service.AddAlert(model.Event{
		EventID:          "evt-2",
		UserID:           "user-1",
		ProcessName:      "chrome.exe",
		SensitiveType:    "客户资料",
		Operation:        "upload",
		RiskLevel:        "high",
		AgentVerdict:     "uncertain",
		AgentConfidence:  0.73,
		AgentExplanation: "structured recall was similar but not strong enough",
		RecallScore:      0.68,
		Timestamp:        2000,
	})
	service.AddAlert(model.Event{
		EventID:       "evt-3",
		UserID:        "user-2",
		ProcessName:   "outlook.exe",
		SensitiveType: "财务数据",
		Operation:     "send",
		RiskLevel:     "medium",
		Timestamp:     3000,
	})
	h := router.BuildWithService(service)

	body := `{
		"risk_level":"high",
		"user_id":"user-1",
		"page":1,
		"page_size":1,
		"order_by":"timestamp",
		"order":"desc"
	}`
	resp := performRequest(h.Engine, "POST", "/api/alerts/query", body)
	if resp.Code != 200 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var result router.AlertQueryResult
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("Total = %d, want 2", result.Total)
	}
	if len(result.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(result.Data))
	}
	if result.Data[0].EventID != "evt-2" {
		t.Fatalf("first event = %s, want evt-2 by desc timestamp", result.Data[0].EventID)
	}
	if result.Data[0].AgentVerdict != "uncertain" || result.Data[0].AgentConfidence != 0.73 || result.Data[0].RecallScore != 0.68 {
		t.Fatalf("agent fields = (%q,%v,%v), want query to return analysis fields", result.Data[0].AgentVerdict, result.Data[0].AgentConfidence, result.Data[0].RecallScore)
	}
}

func TestAlertsQueryCapsPageSize(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithService(service)

	resp := performRequest(h.Engine, "POST", "/api/alerts/query", `{"page_size":1000}`)
	if resp.Code != 200 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var result router.AlertQueryResult
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.PageSize > 100 {
		t.Fatalf("PageSize = %d, want capped at 100", result.PageSize)
	}
}

func TestAlertsQueryNormalizesInvalidPagingAndOrdering(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithService(service)

	resp := performRequest(h.Engine, "POST", "/api/alerts/query", `{"page":0,"page_size":1000,"order":"sideways"}`)
	if resp.Code != 200 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var result router.AlertQueryResult
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Page != 1 {
		t.Fatalf("Page = %d, want normalized page 1", result.Page)
	}
	if result.PageSize != 100 {
		t.Fatalf("PageSize = %d, want normalized cap 100", result.PageSize)
	}
}

func TestEventsPostedToClientEndpointCanBeQueried(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithService(service)

	postBody := `{
		"host_id":"host-1",
		"events":[
			{"event_id":"evt-query","user_id":"user-1","process_name":"chrome.exe","sensitive_type":"客户资料","operation":"upload","risk_level":"high","timestamp":2000}
		]
	}`
	postResp := performRequest(h.Engine, "POST", "/api/client/events", postBody)
	if postResp.Code != 200 {
		t.Fatalf("post status = %d, body = %s", postResp.Code, postResp.Body.String())
	}

	queryResp := performRequest(h.Engine, "POST", "/api/alerts/query", `{"user_id":"user-1","risk_level":"high"}`)
	if queryResp.Code != 200 {
		t.Fatalf("query status = %d, body = %s", queryResp.Code, queryResp.Body.String())
	}

	var result router.AlertQueryResult
	if err := json.Unmarshal(queryResp.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1", result.Total)
	}
	if result.Data[0].EventID != "evt-query" {
		t.Fatalf("EventID = %s, want evt-query", result.Data[0].EventID)
	}
}

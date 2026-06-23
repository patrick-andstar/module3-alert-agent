package router_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/route"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
	"module3-alert-agent/internal/router"
)

type fixedAnalyzer struct{}

func (fixedAnalyzer) Analyze(_ context.Context, event model.Event) (model.Event, error) {
	event.RiskLevel = "info"
	return event, nil
}

func TestBuildRegistersPhaseOneRoutes(t *testing.T) {
	engine := router.Build()
	routes := collectRoutes(engine.Routes())

	expected := []string{
		"GET /healthz",
		"POST /api/client/events",
		"POST /api/alerts/query",
		"GET /api/whitelist",
		"POST /api/whitelist",
		"PUT /api/whitelist/:id",
		"DELETE /api/whitelist/:id",
		"GET /api/false-positives",
		"DELETE /api/false-positives/:id",
	}
	for _, route := range expected {
		if !routes[route] {
			t.Fatalf("route %s was not registered; got %#v", route, routes)
		}
	}
}

func TestHealthzEndpointReturnsOK(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithService(service)

	resp := performRequest(h.Engine, "GET", "/healthz", "")
	if resp.Code != 200 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "ok") {
		t.Fatalf("body = %s, want ok status", resp.Body.String())
	}
}

func TestEventEndpointRunsWhitelistAndDedupPipeline(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	service.CreateWhitelistRule(model.WhitelistRule{
		RuleName:    "backup",
		Logic:       "OR",
		ProcessName: "backup.exe",
		Enabled:     true,
	})
	h := router.BuildWithService(service)

	body := `{
		"host_id":"host-1",
		"events":[
			{"event_id":"drop","host_id":"host-1","user_id":"u1","process_name":"backup.exe","sensitive_type":"customer","operation":"backup","risk_level":"high","timestamp":1000},
			{"event_id":"keep-1","host_id":"host-1","user_id":"u1","process_name":"chrome.exe","sensitive_type":"客户资料","operation":"upload","risk_level":"high","timestamp":1000,"file_path":"C:/a.xlsx"},
			{"event_id":"keep-2","host_id":"host-1","user_id":"u1","process_name":"chrome.exe","sensitive_type":"客户资料","operation":"upload","risk_level":"high","timestamp":1010,"file_path":"C:/b.xlsx"}
		]
	}`
	resp := performRequest(h.Engine, "POST", "/api/client/events", body)
	if resp.Code != 200 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var payload pipeline.Result
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Accepted != 2 || payload.Dropped != 1 {
		t.Fatalf("result = %+v, want accepted=2 dropped=1", payload)
	}
	if len(payload.Events) != 1 || !payload.Events[0].IsMergeEvent {
		t.Fatalf("events = %+v, want one merged alert", payload.Events)
	}
}

func TestEventEndpointRunsAnalyzerBeforePersistingAlerts(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	service.SetAnalyzer(fixedAnalyzer{})
	h := router.BuildWithService(service)

	body := `{
		"host_id":"host-1",
		"events":[
			{"event_id":"agent-1","user_id":"u1","process_name":"chrome.exe","sensitive_type":"客户资料","operation":"upload","risk_level":"high","timestamp":1000}
		]
	}`
	resp := performRequest(h.Engine, "POST", "/api/client/events", body)
	if resp.Code != 200 {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	queryResp := performRequest(h.Engine, "POST", "/api/alerts/query", `{"event_id":"agent-1","risk_level":"info"}`)
	if queryResp.Code != 200 {
		t.Fatalf("query status = %d, body = %s", queryResp.Code, queryResp.Body.String())
	}

	var result router.AlertQueryResult
	if err := json.Unmarshal(queryResp.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want analyzer-updated alert to be persisted", result.Total)
	}
	if result.Data[0].RiskLevel != "info" {
		t.Fatalf("RiskLevel = %q, want info", result.Data[0].RiskLevel)
	}
}

func TestWhitelistCRUDRefreshesServiceRules(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithService(service)

	create := `{"rule_name":"backup","logic":"OR","process_name":"backup.exe","enabled":true}`
	createdResp := performRequest(h.Engine, "POST", "/api/whitelist", create)
	if createdResp.Code != 201 {
		t.Fatalf("create status = %d, body = %s", createdResp.Code, createdResp.Body.String())
	}

	listResp := performRequest(h.Engine, "GET", "/api/whitelist", "")
	if listResp.Code != 200 {
		t.Fatalf("list status = %d, body = %s", listResp.Code, listResp.Body.String())
	}
	if !strings.Contains(listResp.Body.String(), "backup.exe") {
		t.Fatalf("list body = %s, want created rule", listResp.Body.String())
	}

	events := `{"host_id":"host-1","events":[{"event_id":"drop","user_id":"u1","process_name":"backup.exe","sensitive_type":"customer","operation":"backup","risk_level":"high","timestamp":1000}]}`
	eventResp := performRequest(h.Engine, "POST", "/api/client/events", events)
	var result pipeline.Result
	if err := json.Unmarshal(eventResp.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode event response: %v", err)
	}
	if result.Dropped != 1 {
		t.Fatalf("Dropped = %d, want 1 after whitelist create", result.Dropped)
	}
}

func TestProtectedRoutesRequireBearerTokenWhenConfigured(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithOptions("", service, router.Options{AdminToken: "secret"})

	resp := performRequest(h.Engine, "GET", "/api/whitelist", "")
	if resp.Code != 401 {
		t.Fatalf("status = %d, want 401 for missing token; body = %s", resp.Code, resp.Body.String())
	}

	ctx := h.Engine.NewContext()
	req := protocol.NewRequest("GET", "/api/whitelist", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.CopyTo(&ctx.Request)
	h.Engine.ServeHTTP(context.Background(), ctx)
	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d, want 200 with valid token; body = %s", ctx.Response.StatusCode(), string(ctx.Response.Body()))
	}
	ctx.Reset()
}

func TestClientEventsRouteDoesNotRequireAdminToken(t *testing.T) {
	service := router.NewMemoryService(map[string]int{"high": 60})
	h := router.BuildWithOptions("", service, router.Options{AdminToken: "secret"})

	body := `{"host_id":"host-1","events":[{"event_id":"evt-1","user_id":"user-1","process_name":"chrome.exe","sensitive_type":"customer","operation":"upload","risk_level":"high","timestamp":1000}]}`
	resp := performRequest(h.Engine, "POST", "/api/client/events", body)
	if resp.Code != 200 {
		t.Fatalf("status = %d, want 200 for client event route; body = %s", resp.Code, resp.Body.String())
	}
}

func collectRoutes(infos route.RoutesInfo) map[string]bool {
	routes := make(map[string]bool)
	for _, info := range infos {
		routes[info.Method+" "+info.Path] = true
	}
	return routes
}

func performRequest(engine *route.Engine, method, path, body string) *httptest.ResponseRecorder {
	ctx := engine.NewContext()
	req := protocol.NewRequest(method, path, nil)
	req.SetBodyString(body)
	req.Header.SetContentTypeBytes([]byte("application/json"))
	req.CopyTo(&ctx.Request)

	engine.ServeHTTP(context.Background(), ctx)

	w := httptest.NewRecorder()
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		w.Header().Add(string(key), string(value))
	})
	w.WriteHeader(ctx.Response.StatusCode())
	_, _ = w.Write(ctx.Response.Body())
	ctx.Reset()
	return w
}

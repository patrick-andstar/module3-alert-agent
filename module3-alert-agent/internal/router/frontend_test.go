package router_test

import (
	"strings"
	"testing"

	"module3-alert-agent/internal/router"
)

func TestFrontendRoutesServeScenarioConsole(t *testing.T) {
	h := router.Build()

	index := performRequest(h.Engine, "GET", "/", "")
	if index.Code != 200 {
		t.Fatalf("index status = %d, body = %s", index.Code, index.Body.String())
	}
	if !strings.Contains(index.Body.String(), "DLP Alert Agent Console") {
		t.Fatalf("index body missing console title: %s", index.Body.String())
	}

	app := performRequest(h.Engine, "GET", "/app.js", "")
	if app.Code != 200 {
		t.Fatalf("app.js status = %d, body = %s", app.Code, app.Body.String())
	}
	for _, scenario := range []string{"whitelist_drop", "dedup_merge", "confirmed_false_positive", "empty_recall_agent_judgement", "uncertain_candidate", "true_alert"} {
		if !strings.Contains(app.Body.String(), scenario) {
			t.Fatalf("app.js missing scenario %q", scenario)
		}
	}
	for _, marker := range []string{"run_id", "merge-", "业务化详情 + 原始 JSON"} {
		if !strings.Contains(app.Body.String(), marker) {
			t.Fatalf("app.js missing demo evidence marker %q", marker)
		}
	}
}

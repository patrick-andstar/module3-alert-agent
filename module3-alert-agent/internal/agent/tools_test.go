package agent_test

import (
	"context"
	"strings"
	"testing"

	"module3-alert-agent/internal/agent"
	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
)

func TestBuildToolsExposesRequiredToolNames(t *testing.T) {
	tools, err := agent.BuildTools(agent.ToolContext{})
	if err != nil {
		t.Fatalf("BuildTools returned error: %v", err)
	}

	names := map[string]bool{}
	for _, tool := range tools {
		info, err := tool.Info(context.Background())
		if err != nil {
			t.Fatalf("tool info: %v", err)
		}
		names[info.Name] = true
	}

	for _, name := range []string{
		"SearchFalsePositiveHistory",
		"GetEventDetail",
		"QueryWhitelist",
		"MarkAsFalsePositive",
	} {
		if !names[name] {
			t.Fatalf("missing tool %s in %#v", name, names)
		}
	}
}

func TestGetEventDetailToolReturnsEventJSON(t *testing.T) {
	ctx := agent.ToolContext{
		Events: map[string]model.Event{
			"evt-1": {EventID: "evt-1", ProcessName: "chrome.exe"},
		},
	}

	output, err := agent.GetEventDetail(context.Background(), ctx, agent.EventIDInput{EventID: "evt-1"})
	if err != nil {
		t.Fatalf("GetEventDetail returned error: %v", err)
	}
	if !strings.Contains(output.Event.EventID, "evt-1") {
		t.Fatalf("output = %+v, want evt-1", output)
	}
}

func TestQueryWhitelistToolUsesWhitelistCache(t *testing.T) {
	ctx := agent.ToolContext{
		Whitelist: pipeline.NewWhitelistCache([]model.WhitelistRule{
			{RuleName: "backup", Logic: "OR", ProcessName: "backup.exe", Enabled: true},
		}),
	}

	output, err := agent.QueryWhitelist(context.Background(), ctx, agent.EventInput{
		Event: model.Event{ProcessName: "backup.exe"},
	})
	if err != nil {
		t.Fatalf("QueryWhitelist returned error: %v", err)
	}
	if !output.Matched {
		t.Fatal("Matched = false, want true")
	}
}

func TestMarkAsFalsePositiveToolReturnsRecommendationWithoutWritingRecord(t *testing.T) {
	ctx := agent.ToolContext{}
	output, err := agent.MarkAsFalsePositive(context.Background(), ctx, agent.MarkFalsePositiveInput{
		Event: model.Event{
			UserID:      "user-1",
			ProcessName: "chrome.exe",
		},
		Reason:  "正常业务",
		TTLDays: 30,
	})
	if err != nil {
		t.Fatalf("MarkAsFalsePositive returned error: %v", err)
	}
	if !output.Recommended {
		t.Fatal("Recommended = false, want true")
	}
}

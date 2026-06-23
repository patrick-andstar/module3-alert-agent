package agent_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"module3-alert-agent/internal/agent"
	"module3-alert-agent/internal/model"
)

func TestParseDecisionExtractsJSONFromAgentResponse(t *testing.T) {
	decision, err := agent.ParseDecision(`analysis...
{
  "event_id": "evt-1",
  "is_false_positive": true,
  "new_risk_level": "info",
  "false_positive_reason": "normal crm upload",
  "confidence": 0.91,
  "explanation": "matched history"
}
done`)
	if err != nil {
		t.Fatalf("ParseDecision returned error: %v", err)
	}
	if !decision.IsFalsePositive || decision.NewRiskLevel != "info" || decision.Confidence != 0.91 {
		t.Fatalf("decision = %+v, want parsed false-positive decision", decision)
	}
}

func TestBuildDecisionPromptIncludesEventAndRecallContext(t *testing.T) {
	prompt := agent.BuildDecisionPrompt(model.Event{
		EventID:       "evt-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "customer",
		Operation:     "upload",
		Target:        "crm.internal",
		RiskLevel:     "high",
	}, []agent.RecallResult{
		{
			Record: model.FalsePositiveRecord{ID: 42, Reason: "normal crm upload"},
		},
	})

	for _, needle := range []string{"evt-1", "chrome.exe", "normal crm upload", "JSON"} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("prompt %q missing %q", prompt, needle)
		}
	}
}

func TestBuildDecisionPromptDoesNotAskAgentToWriteFalsePositiveLibrary(t *testing.T) {
	prompt := agent.BuildDecisionPrompt(model.Event{EventID: "evt-1"}, nil)

	for _, needle := range []string{"只表示推荐", "服务端决定"} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("prompt %q missing %q", prompt, needle)
		}
	}
	if strings.Contains(prompt, "you may call MarkAsFalsePositive") {
		t.Fatalf("prompt still tells the agent to call MarkAsFalsePositive for writes: %s", prompt)
	}
}

func TestDecisionPromptsAllowDirectJSONWhenContextIsEnough(t *testing.T) {
	if strings.Contains(agent.DefaultDecisionSystemPrompt, "必须先调用") {
		t.Fatalf("system prompt still forces tool calls: %s", agent.DefaultDecisionSystemPrompt)
	}
	if strings.Contains(agent.DefaultDecisionSystemPrompt, "must call") {
		t.Fatalf("system prompt still forces tool calls: %s", agent.DefaultDecisionSystemPrompt)
	}
	for _, needle := range []string{"上下文足够", "直接输出 JSON", "必要时"} {
		if !strings.Contains(agent.DefaultDecisionSystemPrompt, needle) {
			t.Fatalf("system prompt %q missing %q", agent.DefaultDecisionSystemPrompt, needle)
		}
	}

	prompt := agent.BuildDecisionPrompt(model.Event{EventID: "evt-1"}, nil)
	for _, needle := range []string{"当前事件", "误报召回历史", "上下文足够"} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("decision prompt %q missing %q", prompt, needle)
		}
	}
}

func TestBuildDecisionPromptRequiresChineseNaturalLanguageOutput(t *testing.T) {
	prompt := agent.BuildDecisionPrompt(model.Event{EventID: "evt-1"}, nil)

	for _, needle := range []string{
		"自然语言字段必须使用中文",
		"false_positive_reason",
		"explanation",
		"不要翻译文件名、进程名、域名、URL、路径、event_id",
	} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("decision prompt %q missing %q", prompt, needle)
		}
	}
}

func TestBuildDecisionPromptOmitsRecallEmbeddings(t *testing.T) {
	prompt := agent.BuildDecisionPrompt(model.Event{
		EventID: "evt-1",
	}, []agent.RecallResult{
		{
			Record: model.FalsePositiveRecord{
				ID:        42,
				Reason:    "normal crm upload",
				Embedding: []float64{0.123456789, 0.987654321},
			},
			Similarity: 0.91,
		},
	})

	if strings.Contains(prompt, "Embedding") || strings.Contains(prompt, "embedding") {
		t.Fatalf("prompt should not include embedding field: %s", prompt)
	}
	if strings.Contains(prompt, "0.123456789") || strings.Contains(prompt, "0.987654321") {
		t.Fatalf("prompt leaked embedding vector values: %s", prompt)
	}
	if !strings.Contains(prompt, "0.91") {
		t.Fatalf("prompt should keep recall similarity: %s", prompt)
	}
}

func TestLimitedAnalyzerCapsConcurrentCalls(t *testing.T) {
	inner := &countingAnalyzer{release: make(chan struct{})}
	limited := agent.NewLimitedAnalyzer(inner, 2)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = limited.Analyze(context.Background(), model.Event{})
		}()
	}

	deadline := time.After(2 * time.Second)
	for atomic.LoadInt64(&inner.maxActive) < 2 {
		select {
		case <-deadline:
			t.Fatalf("maxActive = %d, want 2", atomic.LoadInt64(&inner.maxActive))
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if got := atomic.LoadInt64(&inner.active); got != 2 {
		t.Fatalf("active = %d, want capped at 2 before release", got)
	}

	for i := 0; i < 5; i++ {
		inner.release <- struct{}{}
	}
	wg.Wait()
	if got := atomic.LoadInt64(&inner.maxActive); got != 2 {
		t.Fatalf("maxActive = %d, want 2", got)
	}
}

type countingAnalyzer struct {
	active    int64
	maxActive int64
	release   chan struct{}
}

func (a *countingAnalyzer) Analyze(_ context.Context, event model.Event) (model.Event, error) {
	active := atomic.AddInt64(&a.active, 1)
	for {
		maxActive := atomic.LoadInt64(&a.maxActive)
		if active <= maxActive || atomic.CompareAndSwapInt64(&a.maxActive, maxActive, active) {
			break
		}
	}
	<-a.release
	atomic.AddInt64(&a.active, -1)
	return event, nil
}

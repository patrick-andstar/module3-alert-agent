package router

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"module3-alert-agent/internal/model"
)

type memoryBlockingAnalyzer struct {
	active    int64
	maxActive int64
	release   chan struct{}
}

func (a *memoryBlockingAnalyzer) Analyze(ctx context.Context, event model.Event) (model.Event, error) {
	active := atomic.AddInt64(&a.active, 1)
	for {
		current := atomic.LoadInt64(&a.maxActive)
		if active <= current || atomic.CompareAndSwapInt64(&a.maxActive, current, active) {
			break
		}
	}
	select {
	case <-a.release:
	case <-ctx.Done():
		atomic.AddInt64(&a.active, -1)
		return event, ctx.Err()
	}
	atomic.AddInt64(&a.active, -1)
	event.RiskLevel = "info"
	return event, nil
}

type contextCapturingAnalyzer struct {
	ctxs chan context.Context
}

func (a contextCapturingAnalyzer) Analyze(ctx context.Context, event model.Event) (model.Event, error) {
	a.ctxs <- ctx
	<-ctx.Done()
	return event, ctx.Err()
}

type failingAnalyzer struct{}

func (failingAnalyzer) Analyze(_ context.Context, event model.Event) (model.Event, error) {
	return event, errors.New("agent unavailable")
}

func TestMemoryServiceAnalyzeEventsRunsBatchConcurrently(t *testing.T) {
	service := NewMemoryService(map[string]int{"high": 60})
	analyzer := &memoryBlockingAnalyzer{release: make(chan struct{})}
	service.SetAnalyzer(analyzer)

	events := []model.Event{
		{EventID: "evt-1", RiskLevel: "high"},
		{EventID: "evt-2", RiskLevel: "high"},
		{EventID: "evt-3", RiskLevel: "high"},
	}

	done := make(chan []model.Event, 1)
	go func() {
		done <- service.analyzeEvents(context.Background(), events)
	}()

	deadline := time.After(2 * time.Second)
	for atomic.LoadInt64(&analyzer.maxActive) < 2 {
		select {
		case <-deadline:
			t.Fatalf("maxActive = %d, want at least 2 concurrent analyses", atomic.LoadInt64(&analyzer.maxActive))
		default:
			time.Sleep(time.Millisecond)
		}
	}

	for range events {
		analyzer.release <- struct{}{}
	}

	output := <-done
	if len(output) != len(events) {
		t.Fatalf("len(output) = %d, want %d", len(output), len(events))
	}
	for i, event := range output {
		if event.EventID != events[i].EventID {
			t.Fatalf("output[%d].EventID = %q, want %q", i, event.EventID, events[i].EventID)
		}
		if event.RiskLevel != "info" {
			t.Fatalf("output[%d].RiskLevel = %q, want info", i, event.RiskLevel)
		}
	}
}

func TestMemoryServiceAnalyzeEventsUsesTimeoutContext(t *testing.T) {
	service := NewMemoryService(map[string]int{"high": 60})
	service.SetAnalysisTimeout(5 * time.Millisecond)
	analyzer := contextCapturingAnalyzer{ctxs: make(chan context.Context, 1)}
	service.SetAnalyzer(analyzer)

	events := []model.Event{{EventID: "evt-1", RiskLevel: "high"}}
	output := service.analyzeEvents(context.Background(), events)

	select {
	case ctx := <-analyzer.ctxs:
		if err := ctx.Err(); err == nil {
			t.Fatal("analyzer context was not canceled after timeout")
		}
	default:
		t.Fatal("analyzer was not called")
	}
	if output[0].EventID != "evt-1" || output[0].RiskLevel != "high" {
		t.Fatalf("output = %+v, want original identity and risk after timeout", output[0])
	}
	if output[0].AgentVerdict != "uncertain" || output[0].AgentConfidence != 0 {
		t.Fatalf("agent fields = (%q,%v), want visible uncertain failure", output[0].AgentVerdict, output[0].AgentConfidence)
	}
	if !strings.Contains(output[0].AgentExplanation, "agent analysis failed") {
		t.Fatalf("AgentExplanation = %q, want failure detail", output[0].AgentExplanation)
	}
}

func TestAnalyzeEventsMarksAnalyzerErrorAsUncertain(t *testing.T) {
	events := []model.Event{{EventID: "evt-1", RiskLevel: "critical"}}

	output := AnalyzeEvents(context.Background(), failingAnalyzer{}, time.Second, events)

	if output[0].RiskLevel != "critical" {
		t.Fatalf("RiskLevel = %q, want original risk preserved", output[0].RiskLevel)
	}
	if output[0].AgentVerdict != "uncertain" || output[0].AgentConfidence != 0 {
		t.Fatalf("agent fields = (%q,%v), want uncertain with zero confidence", output[0].AgentVerdict, output[0].AgentConfidence)
	}
	if !strings.Contains(output[0].AgentExplanation, "agent unavailable") {
		t.Fatalf("AgentExplanation = %q, want analyzer error detail", output[0].AgentExplanation)
	}
}

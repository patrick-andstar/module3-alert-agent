package store

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"module3-alert-agent/internal/model"
)

type blockingAnalyzer struct {
	active    int64
	maxActive int64
	release   chan struct{}
}

func (a *blockingAnalyzer) Analyze(ctx context.Context, event model.Event) (model.Event, error) {
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

type storeContextCapturingAnalyzer struct {
	ctxs chan context.Context
}

func (a storeContextCapturingAnalyzer) Analyze(ctx context.Context, event model.Event) (model.Event, error) {
	a.ctxs <- ctx
	<-ctx.Done()
	return event, ctx.Err()
}

func TestMySQLServiceAnalyzeEventsRunsBatchConcurrently(t *testing.T) {
	analyzer := &blockingAnalyzer{release: make(chan struct{})}
	service := &MySQLService{analyzer: analyzer}
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
			t.Fatalf("output[%d].RiskLevel = %q, want analyzer-updated info", i, event.RiskLevel)
		}
	}
}

func TestMySQLServiceAnalyzeEventsUsesTimeoutContext(t *testing.T) {
	analyzer := storeContextCapturingAnalyzer{ctxs: make(chan context.Context, 1)}
	service := &MySQLService{analyzer: analyzer, analysisTimeout: 5 * time.Millisecond}

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
	if output[0].EventID != "evt-1" {
		t.Fatalf("output = %+v, want original event after timeout", output[0])
	}
}

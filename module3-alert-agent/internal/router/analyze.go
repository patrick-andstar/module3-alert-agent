package router

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"module3-alert-agent/internal/model"
)

func AnalyzeEvents(ctx context.Context, analyzer EventAnalyzer, timeout time.Duration, events []model.Event) []model.Event {
	if analyzer == nil {
		return events
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	output := make([]model.Event, len(events))
	var wg sync.WaitGroup
	for i, event := range events {
		wg.Add(1)
		go func(index int, current model.Event) {
			defer wg.Done()
			analyzeCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			analyzed, err := analyzer.Analyze(analyzeCtx, current)
			if err != nil {
				slog.Error("analyze event", "event_id", current.EventID, "err", err)
				output[index] = markAnalysisFailed(current, err)
				return
			}
			output[index] = analyzed
		}(i, event)
	}
	wg.Wait()
	return output
}

func markAnalysisFailed(event model.Event, err error) model.Event {
	event.AgentVerdict = "uncertain"
	event.AgentConfidence = 0
	event.AgentExplanation = "agent analysis failed: " + err.Error()
	return event
}

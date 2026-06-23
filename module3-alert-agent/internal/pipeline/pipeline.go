package pipeline

import (
	"log/slog"

	"module3-alert-agent/internal/model"
)

type Pipeline struct {
	whitelist *WhitelistCache
	deduper   *Deduper
}

type Result struct {
	Accepted int
	Dropped  int
	Events   []model.Event
}

func New(whitelist *WhitelistCache, deduper *Deduper) *Pipeline {
	return &Pipeline{
		whitelist: whitelist,
		deduper:   deduper,
	}
}

func (p *Pipeline) Process(events []model.Event) Result {
	result := Result{}
	for _, event := range events {
		if p.whitelist != nil {
			if matched, ruleName := p.whitelist.MatchRule(event); matched {
				slog.Info("drop whitelisted event", "event_id", event.EventID, "rule_name", ruleName)
				result.Dropped++
				continue
			}
		}

		result.Accepted++
		if p.deduper != nil {
			event = p.deduper.Add(event)
		}
		if event.IsMergeEvent {
			result.Events = removeEventsWithDedupKey(result.Events, dedupKey(event))
		}
		result.Events = append(result.Events, event)
	}
	return result
}

func removeEventsWithDedupKey(events []model.Event, key string) []model.Event {
	filtered := events[:0]
	for _, event := range events {
		if dedupKey(event) != key {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

package pipeline

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"module3-alert-agent/internal/model"
)

type Deduper struct {
	mu      sync.Mutex
	windows map[string]int
	groups  map[string][]model.Event
}

func NewDeduper(windows map[string]int) *Deduper {
	copied := make(map[string]int, len(windows))
	for level, window := range windows {
		copied[level] = window
	}
	return &Deduper{
		windows: copied,
		groups:  make(map[string][]model.Event),
	}
}

func (d *Deduper) Add(event model.Event) model.Event {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.purgeExpired(event.Timestamp)

	key := dedupKey(event)
	window := int64(d.windows[event.RiskLevel])
	if window <= 0 {
		window = 60
	}

	group := d.groups[key]
	if len(group) == 0 || event.Timestamp-group[0].Timestamp > window || differentDemoRun(group[0], event) {
		d.groups[key] = []model.Event{event}
		return event
	}

	group = append(group, event)
	d.groups[key] = group
	return mergeGroup(group)
}

func (d *Deduper) StartCleaner(interval time.Duration, now func() int64) func() {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if now == nil {
		now = func() int64 { return time.Now().Unix() }
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.mu.Lock()
				d.purgeExpired(now())
				d.mu.Unlock()
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

func (d *Deduper) purgeExpired(now int64) {
	for key, group := range d.groups {
		if len(group) == 0 {
			delete(d.groups, key)
			continue
		}
		window := int64(d.windows[group[0].RiskLevel])
		if window <= 0 {
			window = 60
		}
		if now-group[0].Timestamp > window {
			delete(d.groups, key)
		}
	}
}

func dedupKey(event model.Event) string {
	return event.HostID + "\x00" + event.UserID + "\x00" + event.ProcessName + "\x00" + event.SensitiveType + "\x00" + event.Operation
}

func mergeGroup(group []model.Event) model.Event {
	merged := group[0]
	if eventID := mergedEventID(group); eventID != "" {
		merged.EventID = eventID
	}
	merged.IsMergeEvent = len(group) > 1
	merged.FileCount = len(group)
	merged.Files = make([]model.FileInfo, 0, len(group))
	for _, event := range group {
		merged.Files = append(merged.Files, model.FileInfo{
			FilePath: event.FilePath,
			FileHash: event.FileHash,
		})
	}

	start := group[0].Timestamp
	end := group[len(group)-1].Timestamp
	merged.TimeRange = fmt.Sprintf("%d-%d", start, end)
	merged.Duration = (time.Duration(end-start) * time.Second).String()
	return merged
}

var demoRunEventIDPattern = regexp.MustCompile(`^evt-(.+)-\d+$`)

func mergedEventID(group []model.Event) string {
	if len(group) <= 1 {
		return ""
	}
	matches := demoRunEventIDPattern.FindStringSubmatch(group[0].EventID)
	if len(matches) != 2 || matches[1] == "" {
		return ""
	}
	return fmt.Sprintf("merge-%s-001", matches[1])
}

func differentDemoRun(left, right model.Event) bool {
	leftRun := demoRunID(left.EventID)
	rightRun := demoRunID(right.EventID)
	return leftRun != "" && rightRun != "" && leftRun != rightRun
}

func demoRunID(eventID string) string {
	matches := demoRunEventIDPattern.FindStringSubmatch(eventID)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

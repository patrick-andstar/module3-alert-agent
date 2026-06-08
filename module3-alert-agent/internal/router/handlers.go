package router

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"module3-alert-agent/internal/model"
)

type eventRequest struct {
	HostID string        `json:"host_id"`
	Events []model.Event `json:"events"`
}

func handleEvents(service Service) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		var req eventRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		for i := range req.Events {
			if req.Events[i].HostID == "" {
				req.Events[i].HostID = req.HostID
			}
			if err := ValidateEvent(req.Events[i]); err != nil {
				c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_event", "detail": err.Error()})
				return
			}
		}
		c.JSON(consts.StatusOK, service.ProcessEvents(ctx, req.Events))
	}
}

func ValidateEvent(event model.Event) error {
	required := map[string]string{
		"event_id":       event.EventID,
		"host_id":        event.HostID,
		"user_id":        event.UserID,
		"process_name":   event.ProcessName,
		"sensitive_type": event.SensitiveType,
		"operation":      event.Operation,
		"risk_level":     event.RiskLevel,
	}
	for field, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	limits := map[string]struct {
		value string
		max   int
	}{
		"event_id":          {event.EventID, 64},
		"host_id":           {event.HostID, 64},
		"user_id":           {event.UserID, 64},
		"file_path":         {event.FilePath, 512},
		"file_hash":         {event.FileHash, 128},
		"sensitive_type":    {event.SensitiveType, 128},
		"risk_level":        {event.RiskLevel, 32},
		"process_name":      {event.ProcessName, 256},
		"process_path":      {event.ProcessPath, 512},
		"target":            {event.Target, 512},
		"operation":         {event.Operation, 64},
		"sensitive_file_id": {event.SensitiveFileID, 64},
	}
	for field, limit := range limits {
		if len(limit.value) > limit.max {
			return fmt.Errorf("%s exceeds %d characters", field, limit.max)
		}
	}
	if !model.ValidRiskLevel(event.RiskLevel) {
		return fmt.Errorf("risk_level must be one of critical, high, medium, low, info")
	}
	now := time.Now().Unix()
	if event.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be greater than 0")
	}
	if event.Timestamp > now+24*60*60 {
		return fmt.Errorf("timestamp is too far in the future")
	}
	return nil
}

func healthz() app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, map[string]string{"status": "ok"})
	}
}

func queryAlerts(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		var query AlertQuery
		if err := c.BindJSON(&query); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		c.JSON(consts.StatusOK, service.QueryAlerts(query))
	}
}

func listWhitelist(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, service.ListWhitelistRules())
	}
}

func createWhitelist(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		var rule model.WhitelistRule
		if err := c.BindJSON(&rule); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		c.JSON(consts.StatusCreated, service.CreateWhitelistRule(rule))
	}
}

func updateWhitelist(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		id, err := parseID(c.Param("id"))
		if err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_id"})
			return
		}
		var rule model.WhitelistRule
		if err := c.BindJSON(&rule); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		updated, ok := service.UpdateWhitelistRule(id, rule)
		if !ok {
			c.JSON(consts.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		c.JSON(consts.StatusOK, updated)
	}
}

func deleteWhitelist(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		id, err := parseID(c.Param("id"))
		if err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_id"})
			return
		}
		if !service.DeleteWhitelistRule(id) {
			c.JSON(consts.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		c.JSON(consts.StatusOK, map[string]string{"status": "ok"})
	}
}

func listFalsePositiveRecords(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, service.ListFalsePositiveRecords())
	}
}

func createFalsePositiveRecord(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		var record model.FalsePositiveRecord
		if err := c.BindJSON(&record); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		now := time.Now()
		if record.ScenarioKey == "" {
			record.ScenarioKey = scenarioKeyFromRecord(record)
		}
		if record.ScenarioKey == "" {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_scenario_key"})
			return
		}
		if record.HitCount <= 0 {
			record.HitCount = 1
		}
		if record.LastSeenAt.IsZero() {
			record.LastSeenAt = now
		}
		if record.ExpiredAt.IsZero() {
			record.ExpiredAt = now.Add(30 * 24 * time.Hour)
		}
		if record.CreatedAt.IsZero() {
			record.CreatedAt = now
		}
		if err := service.Save(record); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": "save_false_positive"})
			return
		}
		c.JSON(consts.StatusCreated, record)
	}
}

func deleteFalsePositiveRecord(service Service) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		id, err := parseID(c.Param("id"))
		if err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid_id"})
			return
		}
		if !service.DeleteFalsePositiveRecord(id) {
			c.JSON(consts.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		c.JSON(consts.StatusOK, map[string]string{"status": "ok"})
	}
}

func scenarioKeyFromRecord(record model.FalsePositiveRecord) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(record.SensitiveType)),
		strings.ToLower(strings.TrimSpace(record.Operation)),
		strings.ToLower(strings.TrimSpace(record.ProcessName)),
		normalizeScenarioTarget(record.Target),
	}
	for _, part := range parts {
		if part == "" {
			return ""
		}
	}
	return strings.Join(parts, "|")
}

func normalizeScenarioTarget(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Host)
	}
	return strings.ToLower(strings.Trim(value, "/"))
}

package router_test

import (
	"strings"
	"testing"
	"time"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/router"
)

func TestValidateEventRejectsMissingRequiredFields(t *testing.T) {
	event := validEvent(nil)
	event.UserID = ""
	err := router.ValidateEvent(event)
	if err == nil {
		t.Fatal("ValidateEvent returned nil error for missing required fields")
	}
	if !strings.Contains(err.Error(), "user_id") {
		t.Fatalf("error = %v, want user_id validation", err)
	}
}

func TestValidateEventRejectsInvalidRiskLevel(t *testing.T) {
	err := router.ValidateEvent(validEvent(func(event *model.Event) {
		event.RiskLevel = "severe"
	}))
	if err == nil {
		t.Fatal("ValidateEvent returned nil error for invalid risk_level")
	}
}

func TestValidateEventRejectsOverlongEventID(t *testing.T) {
	err := router.ValidateEvent(validEvent(func(event *model.Event) {
		event.EventID = strings.Repeat("x", 65)
	}))
	if err == nil {
		t.Fatal("ValidateEvent returned nil error for overlong event_id")
	}
}

func TestValidateEventRejectsFarFutureTimestamp(t *testing.T) {
	err := router.ValidateEvent(validEvent(func(event *model.Event) {
		event.Timestamp = time.Now().Add(25 * time.Hour).Unix()
	}))
	if err == nil {
		t.Fatal("ValidateEvent returned nil error for far future timestamp")
	}
}

func TestValidateEventAcceptsValidEvent(t *testing.T) {
	if err := router.ValidateEvent(validEvent(nil)); err != nil {
		t.Fatalf("ValidateEvent returned error for valid event: %v", err)
	}
}

func validEvent(mutator func(*model.Event)) model.Event {
	event := model.Event{
		EventID:       "evt-1",
		HostID:        "host-1",
		UserID:        "user-1",
		ProcessName:   "chrome.exe",
		SensitiveType: "customer",
		Operation:     "upload",
		RiskLevel:     "high",
		Timestamp:     time.Now().Unix(),
	}
	if mutator != nil {
		mutator(&event)
	}
	return event
}

package router_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/router"
)

func TestFalsePositiveEndpointListsOnlyActiveRecordsAndDeletesByID(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	service := router.NewMemoryService(map[string]int{"high": 60})
	service.SetNow(func() time.Time { return now })
	service.Save(model.FalsePositiveRecord{
		UserID:    "active-user",
		Reason:    "active",
		ExpiredAt: now.Add(time.Hour),
	})
	service.Save(model.FalsePositiveRecord{
		UserID:    "expired-user",
		Reason:    "expired",
		ExpiredAt: now.Add(-time.Hour),
	})
	h := router.BuildWithService(service)

	listResp := performRequest(h.Engine, "GET", "/api/false-positives", "")
	if listResp.Code != 200 {
		t.Fatalf("list status = %d, body = %s", listResp.Code, listResp.Body.String())
	}
	body := listResp.Body.String()
	if !strings.Contains(body, "active-user") {
		t.Fatalf("list body = %s, want active record", body)
	}
	if strings.Contains(body, "expired-user") {
		t.Fatalf("list body = %s, should not include expired record", body)
	}

	var records []model.FalsePositiveRecord
	if err := json.Unmarshal(listResp.Body.Bytes(), &records); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}

	deleteResp := performRequest(h.Engine, "DELETE", "/api/false-positives/1", "")
	if deleteResp.Code != 200 {
		t.Fatalf("delete status = %d, body = %s", deleteResp.Code, deleteResp.Body.String())
	}

	afterDelete := performRequest(h.Engine, "GET", "/api/false-positives", "")
	if strings.Contains(afterDelete.Body.String(), "active-user") {
		t.Fatalf("after delete body = %s, deleted record still listed", afterDelete.Body.String())
	}
}

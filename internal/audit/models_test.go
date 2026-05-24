package audit

import (
	"testing"
	"time"
)

func TestAuditTimelineEntryValidate(t *testing.T) {
	entry := AuditTimelineEntry{
		OccurredAt: time.Now(),
		Severity:   AuditSeverityEmergency,
		EventType:  "ACTION_START",
		Summary:    "Emergency action started",
	}
	if err := entry.Validate(); err != nil {
		t.Fatalf("validate audit timeline entry: %v", err)
	}
	entry.EventType = ""
	if err := entry.Validate(); err == nil {
		t.Fatal("expected validation error for missing event type")
	}
}

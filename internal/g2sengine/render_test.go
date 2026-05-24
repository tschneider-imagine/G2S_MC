package g2sengine

import (
	"strings"
	"testing"
	"time"
)

func TestParseActionTemplateDocument(t *testing.T) {
	doc, err := ParseActionTemplateDocument(`{"actions":{"queue_only_no_send":{"message_type":"mute","payload_template":"<x/>"}}}`)
	if err != nil {
		t.Fatalf("parse actions_json: %v", err)
	}
	if _, ok := doc.Actions["queue_only_no_send"]; !ok {
		t.Fatalf("expected action key in parsed document: %+v", doc.Actions)
	}
}

func TestRenderActionMessage(t *testing.T) {
	doc, err := ParseActionTemplateDocument(`{"actions":{"queue_only_no_send":{"message_type":"mute","payload_template":"<dryRun egm=\"{{.EGMID}}\" action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\"/>"}}}`)
	if err != nil {
		t.Fatalf("parse actions_json: %v", err)
	}

	rendered, err := RenderActionMessage(doc, RenderRequest{
		TemplateID:        "template-smoke-no-send",
		TemplateVersion:   1,
		TemplateActionKey: "queue_only_no_send",
		ActionID:          "emergency-broadcast-trigger",
		ActionRunID:       "run-1",
		EGMID:             "EGM-SMOKE-001",
		Timestamp:         time.Date(2026, 5, 24, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("render action message: %v", err)
	}
	if !strings.Contains(rendered.RawPayload, `egm="EGM-SMOKE-001"`) {
		t.Fatalf("rendered payload missing egm id: %s", rendered.RawPayload)
	}
	if !strings.Contains(rendered.RawPayload, `run="run-1"`) {
		t.Fatalf("rendered payload missing action run id: %s", rendered.RawPayload)
	}
	if !strings.Contains(rendered.SummaryJSON, `"rendered":true`) {
		t.Fatalf("render summary missing rendered flag: %s", rendered.SummaryJSON)
	}
}

func TestRenderActionMessageMissingActionKey(t *testing.T) {
	doc, err := ParseActionTemplateDocument(`{"actions":{"queue_only_no_send":{"message_type":"mute","payload_template":"<x/>"}}}`)
	if err != nil {
		t.Fatalf("parse actions_json: %v", err)
	}
	_, err = RenderActionMessage(doc, RenderRequest{
		TemplateActionKey: "missing_key",
	})
	if err == nil {
		t.Fatal("expected missing action key error")
	}
}

func TestRenderActionMessageInvalidTemplateSyntax(t *testing.T) {
	doc, err := ParseActionTemplateDocument(`{"actions":{"queue_only_no_send":{"message_type":"mute","payload_template":"<x>{{.EGMID}</x>"}}}`)
	if err != nil {
		t.Fatalf("parse actions_json: %v", err)
	}
	_, err = RenderActionMessage(doc, RenderRequest{
		TemplateActionKey: "queue_only_no_send",
		EGMID:             "EGM-SMOKE-001",
	})
	if err == nil {
		t.Fatal("expected invalid template syntax error")
	}
}

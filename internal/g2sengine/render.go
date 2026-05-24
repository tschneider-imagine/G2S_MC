package g2sengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

type ActionTemplateDocument struct {
	Actions map[string]ActionMessageTemplate `json:"actions"`
}

type ActionMessageTemplate struct {
	MessageType     string            `json:"message_type"`
	ContentType     string            `json:"content_type,omitempty"`
	PayloadTemplate string            `json:"payload_template"`
	Headers         map[string]string `json:"headers,omitempty"`
	Notes           string            `json:"notes,omitempty"`
}

type RenderRequest struct {
	TemplateID        string
	TemplateVersion   int
	TemplateActionKey string
	ActionID          string
	ActionRunID       string
	ActionStepID      string
	EGMID             string
	HostID            string
	Timestamp         time.Time
	IPAddress         string
	EndpointPath      string
	Variables         map[string]string
}

type RenderedMessage struct {
	TemplateID        string            `json:"template_id"`
	TemplateVersion   int               `json:"template_version"`
	TemplateActionKey string            `json:"template_action_key"`
	MessageType       string            `json:"message_type"`
	ContentType       string            `json:"content_type"`
	Headers           map[string]string `json:"headers,omitempty"`
	RawPayload        string            `json:"raw_payload"`
	SummaryJSON       string            `json:"summary_json"`
	Warnings          []string          `json:"warnings,omitempty"`
}

func ParseActionTemplateDocument(raw string) (ActionTemplateDocument, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ActionTemplateDocument{}, fmt.Errorf("actions_json is required")
	}

	var doc ActionTemplateDocument
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return ActionTemplateDocument{}, fmt.Errorf("parse action template document: %w", err)
	}
	if doc.Actions == nil {
		doc.Actions = map[string]ActionMessageTemplate{}
	}
	return doc, nil
}

func RenderActionMessage(doc ActionTemplateDocument, req RenderRequest) (RenderedMessage, error) {
	actionKey := strings.TrimSpace(req.TemplateActionKey)
	if actionKey == "" {
		return RenderedMessage{}, fmt.Errorf("template action key is required")
	}

	actionTemplate, ok := doc.Actions[actionKey]
	if !ok {
		return RenderedMessage{}, fmt.Errorf("template action key %q not found", actionKey)
	}
	if strings.TrimSpace(actionTemplate.PayloadTemplate) == "" {
		return RenderedMessage{}, fmt.Errorf("payload template is required for action key %q", actionKey)
	}

	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	templateValues := map[string]string{
		"ActionID":          strings.TrimSpace(req.ActionID),
		"ActionRunID":       strings.TrimSpace(req.ActionRunID),
		"ActionStepID":      strings.TrimSpace(req.ActionStepID),
		"TemplateActionKey": actionKey,
		"EGMID":             strings.TrimSpace(req.EGMID),
		"HostID":            strings.TrimSpace(req.HostID),
		"TimestampRFC3339":  ts.UTC().Format(time.RFC3339),
		"IPAddress":         strings.TrimSpace(req.IPAddress),
		"EndpointPath":      strings.TrimSpace(req.EndpointPath),
		"TemplateID":        strings.TrimSpace(req.TemplateID),
		"TemplateVersion":   fmt.Sprintf("%d", req.TemplateVersion),
	}
	for k, v := range req.Variables {
		templateValues[k] = v
	}

	tpl, err := template.New("payload").Option("missingkey=zero").Parse(actionTemplate.PayloadTemplate)
	if err != nil {
		return RenderedMessage{}, fmt.Errorf("parse payload template for %q: %w", actionKey, err)
	}
	var rendered bytes.Buffer
	if err := tpl.Execute(&rendered, templateValues); err != nil {
		return RenderedMessage{}, fmt.Errorf("render payload template for %q: %w", actionKey, err)
	}

	messageType := strings.TrimSpace(actionTemplate.MessageType)
	if messageType == "" {
		messageType = actionKey
	}
	contentType := strings.TrimSpace(actionTemplate.ContentType)
	if contentType == "" {
		contentType = "application/xml"
	}

	summaryJSON, err := json.Marshal(map[string]any{
		"rendered":            true,
		"template_id":         strings.TrimSpace(req.TemplateID),
		"template_version":    req.TemplateVersion,
		"template_action_key": actionKey,
		"message_type":        messageType,
		"egm_id":              strings.TrimSpace(req.EGMID),
		"action_id":           strings.TrimSpace(req.ActionID),
		"action_run_id":       strings.TrimSpace(req.ActionRunID),
	})
	if err != nil {
		return RenderedMessage{}, fmt.Errorf("marshal render summary: %w", err)
	}

	return RenderedMessage{
		TemplateID:        strings.TrimSpace(req.TemplateID),
		TemplateVersion:   req.TemplateVersion,
		TemplateActionKey: actionKey,
		MessageType:       messageType,
		ContentType:       contentType,
		Headers:           cloneStringMap(actionTemplate.Headers),
		RawPayload:        rendered.String(),
		SummaryJSON:       string(summaryJSON),
		Warnings:          []string{},
	}, nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

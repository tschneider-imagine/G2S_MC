package configvalidation

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type fakeStore struct {
	actions   map[string]actions.ActionDefinition
	templates map[string]templates.G2STemplate
	versions  map[string][]templates.G2STemplateVersion
	egms      map[string]egms.EGMRecord
	groups    map[string]egms.EGMGroup
}

func (f *fakeStore) GetG2STemplate(_ context.Context, id string) (*templates.G2STemplate, error) {
	row, ok := f.templates[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetG2STemplateVersion(_ context.Context, templateID string, version int) (*templates.G2STemplateVersion, error) {
	for _, row := range f.versions[templateID] {
		if row.VersionLabel == "1" && version == 1 {
			copy := row
			return &copy, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) GetActiveG2STemplateVersion(_ context.Context, templateID string) (*templates.G2STemplateVersion, error) {
	tpl := f.templates[templateID]
	active := tpl.CurrentVersionID
	if active == "" {
		return nil, nil
	}
	for _, row := range f.versions[templateID] {
		if row.VersionLabel == active || row.ID == active {
			copy := row
			return &copy, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) ListG2STemplates(_ context.Context) ([]templates.G2STemplate, error) {
	rows := make([]templates.G2STemplate, 0, len(f.templates))
	for _, row := range f.templates {
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) ListG2STemplateVersions(_ context.Context, templateID string) ([]templates.G2STemplateVersion, error) {
	rows := f.versions[templateID]
	copyRows := make([]templates.G2STemplateVersion, len(rows))
	copy(copyRows, rows)
	return copyRows, nil
}

func (f *fakeStore) GetActionDefinition(_ context.Context, id string) (*actions.ActionDefinition, error) {
	row, ok := f.actions[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) ListActionDefinitions(_ context.Context) ([]actions.ActionDefinition, error) {
	rows := make([]actions.ActionDefinition, 0, len(f.actions))
	for _, row := range f.actions {
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) ListEGMRecords(_ context.Context) ([]egms.EGMRecord, error) {
	rows := make([]egms.EGMRecord, 0, len(f.egms))
	for _, row := range f.egms {
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) GetEGMGroup(_ context.Context, id string) (*egms.EGMGroup, error) {
	row, ok := f.groups[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) ListEGMGroups(_ context.Context) ([]egms.EGMGroup, error) {
	rows := make([]egms.EGMGroup, 0, len(f.groups))
	for _, row := range f.groups {
		rows = append(rows, row)
	}
	return rows, nil
}

func TestValidateValidConfigNotError(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	svc := Service{
		Store:   st,
		Options: Options{DeliveryTopology: "HOST_LISTENER"},
		Now: func() time.Time {
			return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
		},
	}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if result.Status == StatusError {
		t.Fatalf("expected non-error status, got %+v", result)
	}
}

func TestValidateActionMissingStepKeyReturnsError(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	action := st.actions["emergency-broadcast-trigger"]
	action.Steps[0].TemplateActionKey = "missing_key"
	st.actions[action.ID] = action

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.Actions, "emergency-broadcast-trigger", StatusError)
}

func TestValidateActionTargetGroupEmptyReturnsError(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	group := st.groups["group-main-floor"]
	group.EGMIDs = []string{}
	st.groups[group.ID] = group

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.Actions, "emergency-broadcast-trigger", StatusError)
}

func TestValidateEmergencyMissingReturnActionReturnsError(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	action := st.actions["emergency-broadcast-trigger"]
	action.ReturnActionID = ""
	st.actions[action.ID] = action

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.Actions, "emergency-broadcast-trigger", StatusError)
}

func TestValidateEGMMissingTemplateReturnsError(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	egm := st.egms["EGM-001"]
	egm.TemplateID = ""
	st.egms[egm.EGMID] = egm

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.EGMs, "EGM-001", StatusError)
}

func TestValidateEGMDisabledEmergencyEnabledReturnsWarn(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	egm := st.egms["EGM-001"]
	egm.Enabled = false
	egm.EmergencyEnabled = true
	st.egms[egm.EGMID] = egm

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.EGMs, "EGM-001", StatusWarn)
}

func TestValidateGroupUnknownMemberReturnsWarn(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	group := st.groups["group-main-floor"]
	group.EGMIDs = append(group.EGMIDs, "EGM-UNKNOWN")
	st.groups[group.ID] = group

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.Groups, "group-main-floor", StatusWarn)
}

func TestValidateInvalidMatcherJSONReturnsError(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	version := st.versions["template-generic-g2s-action"][0]
	version.ConfirmationRulesJSON = "{"
	st.versions["template-generic-g2s-action"] = []templates.G2STemplateVersion{version}

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.Templates, "template-generic-g2s-action", StatusError)
}

func TestValidateInvalidEndpointQuirksJSONReturnsError(t *testing.T) {
	ctx := context.Background()
	st := seedValidStore()
	version := st.versions["template-generic-g2s-action"][0]
	version.EndpointQuirksJSON = "{"
	st.versions["template-generic-g2s-action"] = []templates.G2STemplateVersion{version}

	svc := Service{Store: st}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	assertItemStatus(t, result.Templates, "template-generic-g2s-action", StatusError)
}

func assertItemStatus(t *testing.T, rows []ItemResult, id string, expected string) {
	t.Helper()
	for _, row := range rows {
		if row.ID == id {
			if row.Status != expected {
				t.Fatalf("item %s status=%s want=%s row=%+v", id, row.Status, expected, row)
			}
			return
		}
	}
	t.Fatalf("item %s not found", id)
}

func seedValidStore() *fakeStore {
	actionTrigger := actions.ActionDefinition{
		ID:               "emergency-broadcast-trigger",
		Name:             "Emergency Broadcast Trigger",
		Severity:         actions.SeverityEmergency,
		Enabled:          true,
		TargetSelector:   "GROUP:group-main-floor",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Emergency Silence",
			Sequence:          0,
			TemplateActionKey: "emergency_broadcast_silence",
		}},
		RetryPolicyJSON: `{"count":1,"delay_ms":100}`,
		ReturnActionID:  "emergency-broadcast-normal",
		Version:         1,
	}
	actionNormal := actions.ActionDefinition{
		ID:               "emergency-broadcast-normal",
		Name:             "Emergency Broadcast Normal",
		Severity:         actions.SeverityRestore,
		Enabled:          true,
		TargetSelector:   "GROUP:group-main-floor",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Emergency Restore",
			Sequence:          0,
			TemplateActionKey: "emergency_broadcast_restore",
		}},
		Version: 1,
	}
	return &fakeStore{
		actions: map[string]actions.ActionDefinition{
			actionTrigger.ID: actionTrigger,
			actionNormal.ID:  actionNormal,
		},
		templates: map[string]templates.G2STemplate{
			"template-generic-g2s-action": {
				ID:               "template-generic-g2s-action",
				Name:             "Generic G2S Action Template",
				Vendor:           "Generic",
				Status:           templates.TemplateStatusActive,
				CurrentVersionID: "1",
			},
		},
		versions: map[string][]templates.G2STemplateVersion{
			"template-generic-g2s-action": {{
				ID:           "template-generic-g2s-action-v1",
				TemplateID:   "template-generic-g2s-action",
				VersionLabel: "1",
				ActionsJSON:  `{"actions":{"emergency_broadcast_silence":{"message_type":"NOTICE","content_type":"application/xml","payload_template":"<x/>"},"emergency_broadcast_restore":{"message_type":"NOTICE","content_type":"application/xml","payload_template":"<y/>"}}}`,
			}},
		},
		egms: map[string]egms.EGMRecord{
			"EGM-001": {
				EGMID:              "EGM-001",
				DisplayName:        "Cabinet 001",
				Enabled:            true,
				EmergencyEnabled:   true,
				TemplateID:         "template-generic-g2s-action",
				CurrentActionState: egms.EGMActionStateNormal,
			},
		},
		groups: map[string]egms.EGMGroup{
			"group-main-floor": {
				ID:     "group-main-floor",
				Name:   "Main Floor",
				EGMIDs: []string{"EGM-001"},
			},
		},
	}
}

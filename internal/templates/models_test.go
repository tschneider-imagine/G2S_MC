package templates

import "testing"

func TestTemplateValidate(t *testing.T) {
	tpl := G2STemplate{ID: "tpl-1", Name: "IGT Lab", Vendor: "IGT", Status: TemplateStatusDraft}
	if err := tpl.Validate(); err != nil {
		t.Fatalf("validate template: %v", err)
	}
	tpl.Vendor = ""
	if err := tpl.Validate(); err == nil {
		t.Fatal("expected validation error for missing vendor")
	}
}

func TestTemplateVersionValidate(t *testing.T) {
	version := G2STemplateVersion{ID: "tplv-1", TemplateID: "tpl-1", VersionLabel: "v1", ActionsJSON: "{}"}
	if err := version.Validate(); err != nil {
		t.Fatalf("validate template version: %v", err)
	}
	version.ActionsJSON = ""
	if err := version.Validate(); err == nil {
		t.Fatal("expected validation error for missing actions_json")
	}
}

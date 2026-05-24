package egms

import "testing"

func TestEGMRecordValidate(t *testing.T) {
	record := EGMRecord{EGMID: "EGM-1", Enabled: true, EmergencyEnabled: true, CurrentActionState: EGMActionStateNormal}
	if err := record.Validate(); err != nil {
		t.Fatalf("validate record: %v", err)
	}
	record.CurrentActionState = "NOPE"
	if err := record.Validate(); err == nil {
		t.Fatal("expected validation error for invalid state")
	}
}

func TestEGMGroupValidate(t *testing.T) {
	group := EGMGroup{ID: "zone-a", Name: "Zone A"}
	if err := group.Validate(); err != nil {
		t.Fatalf("validate group: %v", err)
	}
	group.Name = ""
	if err := group.Validate(); err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

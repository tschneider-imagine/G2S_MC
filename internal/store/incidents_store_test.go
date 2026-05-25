package store

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/incidents"
)

func TestIncidentLifecycleCreateGetListClose(t *testing.T) {
	ctx := context.Background()
	st := newPhaseStore(t, ctx)
	defer st.Close()

	record, err := st.CreateIncidentRecord(ctx, incidents.IncidentRecord{
		OpenedAt:             time.Now().UTC(),
		Status:               incidents.StatusOpen,
		Severity:             "EMERGENCY",
		PrimaryInputID:       "emergency-broadcast",
		OpenedByTransitionID: 10,
		Summary:              "Emergency Broadcast triggered",
	})
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	if record.ID <= 0 {
		t.Fatalf("incident id=%d", record.ID)
	}

	fetched, err := st.GetIncidentRecord(ctx, record.ID)
	if err != nil {
		t.Fatalf("get incident: %v", err)
	}
	if fetched == nil || fetched.PrimaryInputID != "emergency-broadcast" || fetched.Status != incidents.StatusOpen {
		t.Fatalf("unexpected incident: %+v", fetched)
	}

	openByInput, err := st.GetOpenIncidentByInput(ctx, "emergency-broadcast")
	if err != nil {
		t.Fatalf("get open by input: %v", err)
	}
	if openByInput == nil || openByInput.ID != record.ID {
		t.Fatalf("unexpected open incident by input: %+v", openByInput)
	}

	openRows, err := st.ListOpenIncidentRecords(ctx, 10)
	if err != nil {
		t.Fatalf("list open incidents: %v", err)
	}
	if len(openRows) != 1 || openRows[0].ID != record.ID {
		t.Fatalf("unexpected open rows: %+v", openRows)
	}

	closedAt := time.Now().UTC().Add(2 * time.Minute)
	closed, err := st.CloseIncidentRecord(ctx, record.ID, closedAt, 11, "Return to Normal")
	if err != nil {
		t.Fatalf("close incident: %v", err)
	}
	if closed == nil || closed.Status != incidents.StatusClosed || closed.ClosedByTransitionID != 11 {
		t.Fatalf("unexpected closed incident: %+v", closed)
	}
	openByInput, err = st.GetOpenIncidentByInput(ctx, "emergency-broadcast")
	if err != nil {
		t.Fatalf("get open by input after close: %v", err)
	}
	if openByInput != nil {
		t.Fatalf("expected no open incident after close: %+v", openByInput)
	}
}

func TestIncidentLinkActionRunPersists(t *testing.T) {
	ctx := context.Background()
	st := newPhaseStore(t, ctx)
	defer st.Close()

	now := time.Now().UTC()
	run, err := st.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-incident-1",
		ActionDefinitionID: "action-1",
		StartedAt:          now,
		Status:             actions.RunStatusPending,
		TargetCount:        1,
	})
	if err != nil {
		t.Fatalf("create action run: %v", err)
	}
	record, err := st.CreateIncidentRecord(ctx, incidents.IncidentRecord{
		OpenedAt:             now,
		Status:               incidents.StatusOpen,
		Severity:             "EMERGENCY",
		PrimaryInputID:       "emergency-broadcast",
		OpenedByTransitionID: 21,
		Summary:              "Emergency Broadcast triggered",
	})
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	run.IncidentID = strconv.FormatInt(record.ID, 10)
	if err := st.UpdateActionRun(ctx, run); err != nil {
		t.Fatalf("update action run incident id: %v", err)
	}
	if err := st.UpdateIncidentPrimaryActionRun(ctx, record.ID, run.ID); err != nil {
		t.Fatalf("update incident primary run: %v", err)
	}

	fetchedRun, err := st.GetActionRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get action run: %v", err)
	}
	if fetchedRun == nil || fetchedRun.IncidentID != strconv.FormatInt(record.ID, 10) {
		t.Fatalf("unexpected run incident linkage: %+v", fetchedRun)
	}

	runIDs, err := st.ListActionRunsByIncident(ctx, strconv.FormatInt(record.ID, 10), 10)
	if err != nil {
		t.Fatalf("list runs by incident: %v", err)
	}
	if len(runIDs) != 1 || runIDs[0] != run.ID {
		t.Fatalf("unexpected run ids by incident: %+v", runIDs)
	}
}

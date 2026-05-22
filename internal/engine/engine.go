package engine

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
)

type EventType string

const (
	EventBootComplete     EventType = "BOOT_COMPLETE"
	EventSecurityLineDrop EventType = "SECURITY_LINE_DROP"
	EventG2SSessionOnline EventType = "G2S_SESSION_ONLINE"
	EventKeepAlive        EventType = "G2S_KEEPALIVE"
	EventEGMResult        EventType = "EGM_RESULT"
	EventDegraded         EventType = "DEGRADED"
)

const (
	endpointDriftWindow   = 5 * time.Minute
	recentEndpointHistory = 10
)

type Event struct {
	Type       EventType
	At         time.Time
	EGMID      string
	OK         bool
	Detail     string
	SourceIP   string
	SourcePort int
}

type Snapshot struct {
	ControllerID             string                         `json:"controller_id"`
	State                    model.ControllerState          `json:"state"`
	UpdatedAt                time.Time                      `json:"updated_at"`
	Incident                 *model.Incident                `json:"incident,omitempty"`
	EGMs                     []model.EGM                    `json:"egms"`
	EndpointCollisionSummary model.EndpointCollisionSummary `json:"endpoint_collision_summary"`
	EndpointCollisions       []model.EndpointCollision      `json:"endpoint_collisions,omitempty"`
	LastEvent                string                         `json:"last_event,omitempty"`
	AuditError               string                         `json:"audit_error,omitempty"`
}

type AuditSink interface {
	RecordIncident(context.Context, model.Incident) (int64, error)
	RecordEGMStatus(context.Context, model.EGMStatusSnapshot) error
	RecordEGMComplianceLog(context.Context, model.EGMComplianceLog) error
	RecordStateChange(context.Context, model.StateChange) error
}

type Engine struct {
	controllerID string
	events       chan Event
	audit        AuditSink

	mu             sync.RWMutex
	state          model.ControllerState
	updatedAt      time.Time
	incident       *model.Incident
	nextIncidentID int64
	egms           map[string]model.EGM
	endpointSeen   map[string]map[string]endpointObservationState
	lastEvent      string
	auditError     string
}

type endpointObservationState struct {
	ip        string
	port      int
	firstSeen time.Time
	lastSeen  time.Time
}

func New(controllerID string, roster []config.EGM) *Engine {
	return NewWithAuditSink(controllerID, roster, nil)
}

func NewWithAuditSink(controllerID string, roster []config.EGM, audit AuditSink) *Engine {
	egms := make(map[string]model.EGM, len(roster))
	for _, item := range roster {
		egms[item.EGMID] = model.EGM{
			ID:              item.EGMID,
			IPAddress:       item.IPAddress,
			Port:            item.Port,
			Vendor:          item.Vendor,
			CabinetFamily:   item.CabinetFamily,
			GameTitle:       item.GameTitle,
			SoftwareVersion: item.SoftwareVersion,
			Source:          model.EGMSourceConfigured,
			Status:          model.EGMGreen,
		}
	}

	return &Engine{
		controllerID: controllerID,
		events:       make(chan Event, 64),
		audit:        audit,
		state:        model.StateBooting,
		updatedAt:    time.Now(),
		egms:         egms,
		endpointSeen: make(map[string]map[string]endpointObservationState),
	}
}

func (e *Engine) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-e.events:
				e.handle(event)
			}
		}
	}()
}

func (e *Engine) Submit(event Event) {
	if event.At.IsZero() {
		event.At = time.Now()
	}
	e.events <- event
}

func (e *Engine) Snapshot() Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

	egms := make([]model.EGM, 0, len(e.egms))
	for _, egm := range e.egms {
		if len(egm.EndpointDriftIPs) > 0 {
			egm.EndpointDriftIPs = append([]string(nil), egm.EndpointDriftIPs...)
		}
		if len(egm.EndpointCollisionTypes) > 0 {
			egm.EndpointCollisionTypes = append([]model.EndpointCollisionType(nil), egm.EndpointCollisionTypes...)
		}
		if len(egm.RecentEndpoints) > 0 {
			egm.RecentEndpoints = append([]model.EGMEndpointObservation(nil), egm.RecentEndpoints...)
		}
		egms = append(egms, egm)
	}
	endpointSummary, endpointCollisions, driftByEGM, driftIPsByEGM, collisionTypesByEGM := analyzeEndpointIntegrity(egms, e.updatedAt, endpointDriftWindow)
	for idx := range egms {
		egmID := strings.TrimSpace(egms[idx].ID)
		egms[idx].EndpointDrift = driftByEGM[egmID]
		if driftIPs := driftIPsByEGM[egmID]; len(driftIPs) > 0 {
			egms[idx].EndpointDriftIPs = append([]string(nil), driftIPs...)
		} else {
			egms[idx].EndpointDriftIPs = nil
		}
		if collisionTypes := collisionTypesByEGM[egmID]; len(collisionTypes) > 0 {
			egms[idx].EndpointCollisionWarning = true
			egms[idx].EndpointCollisionTypes = append([]model.EndpointCollisionType(nil), collisionTypes...)
		} else {
			egms[idx].EndpointCollisionWarning = false
			egms[idx].EndpointCollisionTypes = nil
		}
	}

	var incident *model.Incident
	if e.incident != nil {
		copy := *e.incident
		incident = &copy
	}

	return Snapshot{
		ControllerID:             e.controllerID,
		State:                    e.state,
		UpdatedAt:                e.updatedAt,
		Incident:                 incident,
		EGMs:                     egms,
		EndpointCollisionSummary: endpointSummary,
		EndpointCollisions:       endpointCollisions,
		LastEvent:                e.lastEvent,
		AuditError:               e.auditError,
	}
}

func (e *Engine) handle(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	oldState := e.state
	e.updatedAt = event.At
	e.lastEvent = string(event.Type)

	switch event.Type {
	case EventBootComplete:
		if e.state == model.StateBooting {
			e.state = model.StateHealthy
		}
	case EventSecurityLineDrop:
		e.openIncident(event)
		e.state = model.StateEmergencyActive
	case EventG2SSessionOnline, EventKeepAlive:
		egmID := strings.TrimSpace(event.EGMID)
		if egmID == "" {
			break
		}
		event.EGMID = egmID
		egm, ok := e.egms[egmID]
		if !ok {
			egm = model.EGM{
				ID:     egmID,
				Source: model.EGMSourceDiscovered,
			}
		}
		if egm.Source == "" {
			egm.Source = model.EGMSourceConfigured
		}
		egm.Status = model.EGMGreen
		egm.LastSeen = event.At
		egm.LastError = ""
		e.observeEndpoint(&egm, event)
		e.egms[egmID] = egm
		e.recordEGMStatus(event, egm)
	case EventEGMResult:
		egmID := strings.TrimSpace(event.EGMID)
		if egmID == "" {
			break
		}
		event.EGMID = egmID
		if egm, ok := e.egms[egmID]; ok {
			egm.LastSeen = event.At
			if event.OK {
				egm.Status = model.EGMRed
				egm.LastError = ""
			} else {
				egm.Status = model.EGMGrey
				egm.LastError = event.Detail
			}
			e.egms[egmID] = egm
			e.recordEGMStatus(event, egm)
			e.recordEGMCompliance(event, egm)
		}
	case EventDegraded:
		e.state = model.StateDegraded
	}

	if oldState != e.state {
		e.recordStateChange(oldState, e.state, event)
	}
}

func (e *Engine) observeEndpoint(egm *model.EGM, event Event) {
	if egm == nil {
		return
	}
	sourceIP := strings.TrimSpace(event.SourceIP)
	sourcePort := event.SourcePort
	if sourceIP != "" {
		egm.LastEndpointIP = sourceIP
	}
	if sourcePort > 0 {
		egm.LastEndpointPort = sourcePort
	}
	if sourceIP != "" || sourcePort > 0 {
		egm.LastEndpointSeenAt = event.At
		egm.RecentEndpoints = updateRecentEndpoints(egm.RecentEndpoints, sourceIP, sourcePort, event.At, recentEndpointHistory)
	}
	if sourceIP == "" {
		return
	}
	observed := e.endpointSeen[egm.ID]
	if observed == nil {
		observed = make(map[string]endpointObservationState)
	}
	endpointKey := formatEndpoint(sourceIP, sourcePort)
	entry := observed[endpointKey]
	if entry.firstSeen.IsZero() || event.At.Before(entry.firstSeen) {
		entry.firstSeen = event.At
	}
	if entry.lastSeen.IsZero() || event.At.After(entry.lastSeen) {
		entry.lastSeen = event.At
	}
	entry.ip = sourceIP
	entry.port = sourcePort
	observed[endpointKey] = entry
	cutoff := event.At.Add(-endpointDriftWindow)
	for endpoint, seen := range observed {
		if !seen.lastSeen.IsZero() && seen.lastSeen.Before(cutoff) {
			delete(observed, endpoint)
		}
	}
	if len(observed) == 0 {
		delete(e.endpointSeen, egm.ID)
		egm.EndpointDrift = false
		egm.EndpointDriftIPs = nil
		return
	}
	e.endpointSeen[egm.ID] = observed
	ipsByEndpoint := map[string]struct{}{}
	for _, seen := range observed {
		ip := strings.TrimSpace(seen.ip)
		if ip == "" {
			continue
		}
		ipsByEndpoint[ip] = struct{}{}
	}
	ips := make([]string, 0, len(ipsByEndpoint))
	for ip := range ipsByEndpoint {
		ips = append(ips, ip)
	}
	sort.Strings(ips)
	egm.EndpointDrift = len(observed) > 1
	if egm.EndpointDrift {
		egm.EndpointDriftIPs = ips
	} else {
		egm.EndpointDriftIPs = nil
	}
}

func analyzeEndpointIntegrity(
	egms []model.EGM,
	reference time.Time,
	window time.Duration,
) (
	model.EndpointCollisionSummary,
	[]model.EndpointCollision,
	map[string]bool,
	map[string][]string,
	map[string][]model.EndpointCollisionType,
) {
	cutoff := time.Time{}
	if !reference.IsZero() && window > 0 {
		cutoff = reference.Add(-window)
	}

	type endpointClaim struct {
		egmID     string
		endpoint  string
		firstSeen time.Time
		lastSeen  time.Time
	}
	claimsByEndpoint := map[string][]endpointClaim{}
	driftByEGM := map[string]bool{}
	driftIPsByEGM := map[string][]string{}
	collisionTypeSetByEGM := map[string]map[model.EndpointCollisionType]struct{}{}
	collisions := make([]model.EndpointCollision, 0)

	addCollisionType := func(egmID string, collisionType model.EndpointCollisionType) {
		egmID = strings.TrimSpace(egmID)
		if egmID == "" {
			return
		}
		set := collisionTypeSetByEGM[egmID]
		if set == nil {
			set = map[model.EndpointCollisionType]struct{}{}
		}
		set[collisionType] = struct{}{}
		collisionTypeSetByEGM[egmID] = set
	}

	for _, egm := range egms {
		egmID := strings.TrimSpace(egm.ID)
		if egmID == "" {
			continue
		}
		activeEndpoints := activeEndpointObservations(egm, cutoff)
		if len(activeEndpoints) == 0 {
			driftByEGM[egmID] = false
			driftIPsByEGM[egmID] = nil
			continue
		}

		ipsSet := map[string]struct{}{}
		for endpoint, entry := range activeEndpoints {
			claimsByEndpoint[endpoint] = append(claimsByEndpoint[endpoint], endpointClaim{
				egmID:     egmID,
				endpoint:  endpoint,
				firstSeen: entry.FirstSeenAt,
				lastSeen:  entry.LastSeenAt,
			})
			ip := strings.TrimSpace(entry.IPAddress)
			if ip != "" {
				ipsSet[ip] = struct{}{}
			}
		}
		driftByEGM[egmID] = len(activeEndpoints) > 1
		if driftByEGM[egmID] {
			addCollisionType(egmID, model.EndpointCollisionIDEndpointDrift)
			ips := make([]string, 0, len(ipsSet))
			for ip := range ipsSet {
				ips = append(ips, ip)
			}
			sort.Strings(ips)
			driftIPsByEGM[egmID] = ips

			endpointKeys := make([]string, 0, len(activeEndpoints))
			for endpoint := range activeEndpoints {
				endpointKeys = append(endpointKeys, endpoint)
			}
			sort.Strings(endpointKeys)
			for _, endpoint := range endpointKeys {
				entry := activeEndpoints[endpoint]
				collisions = append(collisions, model.EndpointCollision{
					CollisionType:  model.EndpointCollisionIDEndpointDrift,
					InvolvedEGMIDs: []string{egmID},
					Endpoint:       endpoint,
					FirstSeenAt:    entry.FirstSeenAt,
					LastSeenAt:     entry.LastSeenAt,
				})
			}
		} else {
			driftIPsByEGM[egmID] = nil
		}
	}

	for endpoint, claims := range claimsByEndpoint {
		seenEGMIDs := map[string]struct{}{}
		firstSeen := time.Time{}
		lastSeen := time.Time{}
		for _, claim := range claims {
			seenEGMIDs[claim.egmID] = struct{}{}
			if firstSeen.IsZero() || (!claim.firstSeen.IsZero() && claim.firstSeen.Before(firstSeen)) {
				firstSeen = claim.firstSeen
			}
			if lastSeen.IsZero() || claim.lastSeen.After(lastSeen) {
				lastSeen = claim.lastSeen
			}
		}
		if len(seenEGMIDs) < 2 {
			continue
		}
		ids := make([]string, 0, len(seenEGMIDs))
		for id := range seenEGMIDs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		collisions = append(collisions, model.EndpointCollision{
			CollisionType:  model.EndpointCollisionSharedEndpoint,
			InvolvedEGMIDs: ids,
			Endpoint:       endpoint,
			FirstSeenAt:    firstSeen,
			LastSeenAt:     lastSeen,
		})
		for _, id := range ids {
			addCollisionType(id, model.EndpointCollisionSharedEndpoint)
		}
	}

	sort.Slice(collisions, func(i, j int) bool {
		if collisions[i].CollisionType != collisions[j].CollisionType {
			return collisions[i].CollisionType < collisions[j].CollisionType
		}
		if collisions[i].Endpoint != collisions[j].Endpoint {
			return collisions[i].Endpoint < collisions[j].Endpoint
		}
		if collisions[i].LastSeenAt.Equal(collisions[j].LastSeenAt) {
			return strings.Join(collisions[i].InvolvedEGMIDs, ",") < strings.Join(collisions[j].InvolvedEGMIDs, ",")
		}
		return collisions[i].LastSeenAt.After(collisions[j].LastSeenAt)
	})

	collisionTypesByEGM := map[string][]model.EndpointCollisionType{}
	affectedIDSet := map[string]struct{}{}
	sharedCount := 0
	driftCount := 0
	for _, collision := range collisions {
		if collision.CollisionType == model.EndpointCollisionSharedEndpoint {
			sharedCount++
		}
		if collision.CollisionType == model.EndpointCollisionIDEndpointDrift {
			driftCount++
		}
	}
	for egmID, typeSet := range collisionTypeSetByEGM {
		if len(typeSet) == 0 {
			continue
		}
		affectedIDSet[egmID] = struct{}{}
		types := make([]model.EndpointCollisionType, 0, len(typeSet))
		if _, ok := typeSet[model.EndpointCollisionSharedEndpoint]; ok {
			types = append(types, model.EndpointCollisionSharedEndpoint)
		}
		if _, ok := typeSet[model.EndpointCollisionIDEndpointDrift]; ok {
			types = append(types, model.EndpointCollisionIDEndpointDrift)
		}
		collisionTypesByEGM[egmID] = types
	}
	affected := make([]string, 0, len(affectedIDSet))
	for egmID := range affectedIDSet {
		affected = append(affected, egmID)
	}
	sort.Strings(affected)

	return model.EndpointCollisionSummary{
			Total:                len(collisions),
			SharedEndpointCount:  sharedCount,
			IDEndpointDriftCount: driftCount,
			AffectedEGMIDs:       affected,
		},
		collisions,
		driftByEGM,
		driftIPsByEGM,
		collisionTypesByEGM
}

func activeEndpointObservations(egm model.EGM, cutoff time.Time) map[string]model.EGMEndpointObservation {
	active := map[string]model.EGMEndpointObservation{}
	recent := append([]model.EGMEndpointObservation(nil), egm.RecentEndpoints...)
	for _, entry := range recent {
		ip := strings.TrimSpace(entry.IPAddress)
		port := entry.Port
		lastSeen := entry.LastSeenAt
		if ip == "" {
			continue
		}
		if lastSeen.IsZero() {
			lastSeen = egm.LastEndpointSeenAt
		}
		if !cutoff.IsZero() && !lastSeen.IsZero() && lastSeen.Before(cutoff) {
			continue
		}
		firstSeen := entry.FirstSeenAt
		if firstSeen.IsZero() {
			firstSeen = lastSeen
		}
		endpoint := formatEndpoint(ip, port)
		existing, ok := active[endpoint]
		if !ok {
			active[endpoint] = model.EGMEndpointObservation{
				IPAddress:   ip,
				Port:        port,
				FirstSeenAt: firstSeen,
				LastSeenAt:  lastSeen,
				SeenCount:   maxInt(1, entry.SeenCount),
			}
			continue
		}
		if existing.FirstSeenAt.IsZero() || (!firstSeen.IsZero() && firstSeen.Before(existing.FirstSeenAt)) {
			existing.FirstSeenAt = firstSeen
		}
		if existing.LastSeenAt.IsZero() || lastSeen.After(existing.LastSeenAt) {
			existing.LastSeenAt = lastSeen
		}
		existing.SeenCount += maxInt(1, entry.SeenCount)
		active[endpoint] = existing
	}
	if len(active) == 0 && strings.TrimSpace(egm.LastEndpointIP) != "" && !egm.LastEndpointSeenAt.IsZero() {
		if cutoff.IsZero() || !egm.LastEndpointSeenAt.Before(cutoff) {
			endpoint := formatEndpoint(egm.LastEndpointIP, egm.LastEndpointPort)
			active[endpoint] = model.EGMEndpointObservation{
				IPAddress:   strings.TrimSpace(egm.LastEndpointIP),
				Port:        egm.LastEndpointPort,
				FirstSeenAt: egm.LastEndpointSeenAt,
				LastSeenAt:  egm.LastEndpointSeenAt,
				SeenCount:   1,
			}
		}
	}
	return active
}

func formatEndpoint(ip string, port int) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if port > 0 {
		return ip + ":" + strconv.Itoa(port)
	}
	return ip + ":0"
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func updateRecentEndpoints(history []model.EGMEndpointObservation, ip string, port int, at time.Time, limit int) []model.EGMEndpointObservation {
	if limit <= 0 || at.IsZero() {
		return history
	}
	ip = strings.TrimSpace(ip)
	if ip == "" && port <= 0 {
		return history
	}
	records := make([]model.EGMEndpointObservation, 0, len(history)+1)
	records = append(records, history...)
	idx := -1
	for i := range records {
		if records[i].IPAddress == ip && records[i].Port == port {
			idx = i
			break
		}
	}
	if idx >= 0 {
		entry := records[idx]
		if entry.FirstSeenAt.IsZero() || at.Before(entry.FirstSeenAt) {
			entry.FirstSeenAt = at
		}
		if entry.LastSeenAt.IsZero() || at.After(entry.LastSeenAt) {
			entry.LastSeenAt = at
		}
		if entry.SeenCount < 0 {
			entry.SeenCount = 0
		}
		entry.SeenCount++
		records = append(records[:idx], records[idx+1:]...)
		records = append([]model.EGMEndpointObservation{entry}, records...)
	} else {
		records = append([]model.EGMEndpointObservation{{
			IPAddress:   ip,
			Port:        port,
			FirstSeenAt: at,
			LastSeenAt:  at,
			SeenCount:   1,
		}}, records...)
	}
	if len(records) > limit {
		records = records[:limit]
	}
	return records
}

func (e *Engine) openIncident(event Event) {
	if e.incident != nil && e.incident.ResolvedAt == nil {
		return
	}

	e.nextIncidentID++
	e.incident = &model.Incident{
		ID:            e.nextIncidentID,
		TriggerType:   string(event.Type),
		TriggerSource: event.Detail,
		CreatedAt:     event.At,
		FinalState:    model.StateEmergencyActive,
	}
	if e.audit != nil {
		id, err := e.audit.RecordIncident(context.Background(), *e.incident)
		if err != nil {
			e.setAuditError("record incident", err)
			return
		}
		e.incident.ID = id
	}
}

func (e *Engine) recordEGMStatus(event Event, egm model.EGM) {
	if e.audit == nil {
		return
	}
	if err := e.audit.RecordEGMStatus(context.Background(), model.EGMStatusSnapshot{
		EGMID:     egm.ID,
		Status:    egm.Status,
		EventType: string(event.Type),
		Detail:    event.Detail,
		LastError: egm.LastError,
		CreatedAt: event.At,
	}); err != nil {
		e.setAuditError("record egm status", err)
	}
}

func (e *Engine) recordEGMCompliance(event Event, egm model.EGM) {
	if e.audit == nil || e.incident == nil || e.incident.ID == 0 {
		return
	}
	status := "SUCCESS"
	if !event.OK {
		status = "GREY_TIMEOUT"
		if event.Detail != "" {
			status = event.Detail
		}
	}
	if err := e.audit.RecordEGMComplianceLog(context.Background(), model.EGMComplianceLog{
		IncidentID:   e.incident.ID,
		EGMID:        egm.ID,
		IPAddress:    egm.IPAddress,
		ActionSent:   string(event.Type),
		StatusResult: status,
		CreatedAt:    event.At,
	}); err != nil {
		e.setAuditError("record egm compliance", err)
	}
}

func (e *Engine) recordStateChange(oldState model.ControllerState, newState model.ControllerState, event Event) {
	if e.audit == nil {
		return
	}
	if err := e.audit.RecordStateChange(context.Background(), model.StateChange{
		OldState:  oldState,
		NewState:  newState,
		Reason:    string(event.Type),
		CreatedAt: event.At,
	}); err != nil {
		e.setAuditError("record state change", err)
	}
}

func (e *Engine) setAuditError(action string, err error) {
	e.auditError = action + ": " + err.Error()
	log.Printf("audit error: %s", e.auditError)
}

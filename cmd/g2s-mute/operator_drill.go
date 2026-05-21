package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
)

type operatorDrillState struct {
	SelectedEGMID        string    `json:"selected_egm_id"`
	AvailableEGMIDs      []string  `json:"available_egm_ids"`
	AutoHeartbeatRunning bool      `json:"auto_heartbeat_running"`
	AutoHeartbeatPaused  bool      `json:"auto_heartbeat_paused"`
	IntervalMS           int       `json:"interval_ms"`
	BurstCount           int       `json:"burst_count"`
	LastAction           string    `json:"last_action"`
	LastActionAt         time.Time `json:"last_action_at,omitempty"`
}

type operatorDrillCommand struct {
	Action     string `json:"action"`
	EGMID      string `json:"egm_id"`
	IntervalMS int    `json:"interval_ms"`
	BurstCount int    `json:"burst_count"`
}

type operatorDrillManager struct {
	engine  *engine.Engine
	egmIDs  []string

	mu                 sync.Mutex
	selectedEGMID      string
	autoHeartbeatRun   bool
	autoHeartbeatPause bool
	intervalMS         int
	burstCount         int
	lastAction         string
	lastActionAt       time.Time
	stopAutoHeartbeat  chan struct{}
}

func newOperatorDrillManager(eng *engine.Engine, roster []config.EGM, defaultIntervalMS int) *operatorDrillManager {
	egmIDs := make([]string, 0, len(roster))
	for _, egm := range roster {
		if strings.TrimSpace(egm.EGMID) != "" {
			egmIDs = append(egmIDs, strings.TrimSpace(egm.EGMID))
		}
	}
	selected := ""
	if len(egmIDs) > 0 {
		selected = egmIDs[0]
	}
	interval := defaultIntervalMS
	if interval <= 0 {
		interval = 5000
	}
	return &operatorDrillManager{
		engine:         eng,
		egmIDs:         egmIDs,
		selectedEGMID:  selected,
		intervalMS:     interval,
		burstCount:     5,
		lastAction:     "idle",
		stopAutoHeartbeat: nil,
	}
}

func (m *operatorDrillManager) snapshot() operatorDrillState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return operatorDrillState{
		SelectedEGMID:        m.selectedEGMID,
		AvailableEGMIDs:      append([]string{}, m.egmIDs...),
		AutoHeartbeatRunning: m.autoHeartbeatRun,
		AutoHeartbeatPaused:  m.autoHeartbeatPause,
		IntervalMS:           m.intervalMS,
		BurstCount:           m.burstCount,
		LastAction:           m.lastAction,
		LastActionAt:         m.lastActionAt,
	}
}

func (m *operatorDrillManager) apply(command operatorDrillCommand) (operatorDrillState, error) {
	action := strings.ToLower(strings.TrimSpace(command.Action))
	m.mu.Lock()
	if command.IntervalMS > 0 {
		m.intervalMS = command.IntervalMS
	}
	if command.BurstCount > 0 {
		m.burstCount = command.BurstCount
	}
	intervalMS := m.intervalMS
	burstCount := m.burstCount
	m.mu.Unlock()

	switch action {
	case "pause":
		m.stopAutoHeartbeatLoop(true)
	case "clear":
		m.stopAutoHeartbeatLoop(false)
		m.mu.Lock()
		m.lastAction = "cleared"
		m.lastActionAt = time.Now()
		m.burstCount = 5
		m.mu.Unlock()
		return m.snapshot(), nil
	case "comms_online":
		egmID, err := m.normalizeEGMID(command.EGMID)
		if err != nil {
			return operatorDrillState{}, err
		}
		m.setSelectedEGMID(egmID)
		m.submit(engine.Event{Type: engine.EventG2SSessionOnline, EGMID: egmID, At: time.Now(), Detail: "operator drill comms online"})
	case "keepalive":
		egmID, err := m.normalizeEGMID(command.EGMID)
		if err != nil {
			return operatorDrillState{}, err
		}
		m.setSelectedEGMID(egmID)
		m.submit(engine.Event{Type: engine.EventKeepAlive, EGMID: egmID, At: time.Now(), Detail: "operator drill keepalive"})
	case "keepalive_burst":
		egmID, err := m.normalizeEGMID(command.EGMID)
		if err != nil {
			return operatorDrillState{}, err
		}
		m.setSelectedEGMID(egmID)
		for i := 0; i < burstCount; i++ {
			m.submit(engine.Event{Type: engine.EventKeepAlive, EGMID: egmID, At: time.Now().Add(time.Duration(i) * time.Millisecond), Detail: "operator drill keepalive burst"})
		}
	case "resume":
		egmID, err := m.normalizeEGMID(command.EGMID)
		if err != nil {
			return operatorDrillState{}, err
		}
		m.setSelectedEGMID(egmID)
		m.startAutoHeartbeat(egmID, intervalMS)
	default:
		return operatorDrillState{}, errInvalidOperatorDrillAction
	}
	m.recordAction(action)
	return m.snapshot(), nil
}

func (m *operatorDrillManager) normalizeEGMID(raw string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value := strings.TrimSpace(raw)
	if value == "" {
		if m.selectedEGMID == "" {
			return "", errOperatorDrillNoEGM
		}
		return m.selectedEGMID, nil
	}
	for _, id := range m.egmIDs {
		if id == value {
			return value, nil
		}
	}
	return "", errOperatorDrillUnknownEGM
}

func (m *operatorDrillManager) submit(event engine.Event) {
	m.engine.Submit(event)
}

func (m *operatorDrillManager) setSelectedEGMID(egmID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if egmID != "" {
		m.selectedEGMID = egmID
	}
}

func (m *operatorDrillManager) recordAction(action string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastAction = action
	m.lastActionAt = time.Now()
}

func (m *operatorDrillManager) startAutoHeartbeat(egmID string, intervalMS int) {
	if intervalMS <= 0 {
		intervalMS = 5000
	}
	m.stopAutoHeartbeatLoop(false)
	stopCh := make(chan struct{})
	m.mu.Lock()
	m.autoHeartbeatRun = true
	m.autoHeartbeatPause = false
	m.stopAutoHeartbeat = stopCh
	m.intervalMS = intervalMS
	m.mu.Unlock()

	go func(selected string, interval time.Duration, stop <-chan struct{}) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.submit(engine.Event{Type: engine.EventKeepAlive, EGMID: selected, At: time.Now(), Detail: "operator drill auto heartbeat"})
			case <-stop:
				return
			}
		}
	}(egmID, time.Duration(intervalMS)*time.Millisecond, stopCh)
}

func (m *operatorDrillManager) stopAutoHeartbeatLoop(paused bool) {
	m.mu.Lock()
	stopCh := m.stopAutoHeartbeat
	m.stopAutoHeartbeat = nil
	m.autoHeartbeatRun = false
	m.autoHeartbeatPause = paused
	m.mu.Unlock()
	if stopCh != nil {
		close(stopCh)
	}
}

var (
	errInvalidOperatorDrillAction = httpError("action must be comms_online, keepalive, keepalive_burst, resume, pause, or clear")
	errOperatorDrillNoEGM         = httpError("no EGM is configured for operator drill")
	errOperatorDrillUnknownEGM    = httpError("selected egm_id is not in the configured roster")
)

type httpError string

func (e httpError) Error() string { return string(e) }

func operatorDrillHandler(manager *operatorDrillManager, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, manager.snapshot(), nil)
		case http.MethodPost:
			if !requireMutationAuth(w, r, cfg) {
				return
			}
			var command operatorDrillCommand
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&command); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			state, err := manager.apply(command)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, state, nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (m *operatorDrillManager) shutdown(ctx context.Context) {
	_ = ctx
	m.stopAutoHeartbeatLoop(false)
}

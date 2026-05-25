# Phase 5E: Emergency Incident Lifecycle Lite

## Scope

Add incident lifecycle support that links emergency evidence across input transitions, action runs, messages, and audit timeline entries.

## Included

- Open incident on triggered input transition.
- Reuse existing open incident for duplicate trigger transitions on the same input.
- Link queued action runs to the active incident.
- Keep manual-clear incidents open until explicit clear-to-normal transition.
- Close incident on return-to-normal transition and record closure evidence.
- Show active incident context on Live and support `incident_id` filtering on Audit.
- Export incident evidence package via:
  - `/operator/audit/evidence-export?incident_id=<id>`

## Not Included

- No new top-level Operator Console page.
- No delivery behavior change.
- No action execution behavior change beyond incident linkage metadata.
- No simulator or fake EGM behavior.

## MVP Alignment

- Emergency Audit Timeline
- Exportable emergency evidence
- Electrical Input Monitor
- Action Builder Lite
- G2S Comms Journal
- Return-to-normal action flow


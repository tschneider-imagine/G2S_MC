# Phase 5D: Operator Live View

## Scope

Improve `/operator` as the live operational landing view using existing runtime/store data.

This phase adds display-only summaries for:

- Current operation and active input priority
- Active inputs and latch visibility
- Active action runs requiring operator attention
- EGM attention items
- Recent message journal events
- Recent audit timeline events

## What This Phase Includes

- Live page data builder for operational summaries
- Updated `/operator` panels for current operation, active inputs/actions, EGM attention, recent messages, and recent audit events
- Optional lightweight JSON refresh route (`/operator/live.json`) for periodic in-page updates

## What This Phase Does Not Include

- No new top-level Operator Console page
- No delivery behavior changes
- No action execution behavior changes
- No input runtime behavior changes
- No legacy dashboard routes

## MVP Alignment

This phase advances:

- Electrical Input Monitor
- Action Builder Lite
- EGM Registry
- G2S Comms Journal
- Emergency Audit Timeline
- Network/Cert Settings


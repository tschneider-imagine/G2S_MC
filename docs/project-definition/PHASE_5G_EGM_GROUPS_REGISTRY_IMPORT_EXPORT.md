# Phase 5G: EGM Groups And Registry Import/Export Lite

## Scope
- Advances EGM Registry management in the existing `EGMs` area.
- Adds group management under `/operator/egms` with explicit group membership.
- Adds registry import and export for EGM and group configuration.

## Included
- EGM registry table and edit forms include template assignment, emergency participation, and group membership visibility.
- Group table and forms under `EGMs`:
  - Group ID
  - Name
  - Description
  - Group Members
- Registry export endpoint:
  - `GET /operator/egms/export`
- Registry import endpoint:
  - `POST /operator/egms/import`
- Import is upsert-only in this phase (no delete/replace behavior).

## Guardrails
- No new top-level page.
- No fake EGM behavior.
- No delivery, action execution, or input runtime behavior changes.
- Group membership is explicit and planner-compatible.

## Definition Of Done
- `/operator/egms` supports EGM and group review/edit.
- Registry import/export works with validation and mutation auth for import.
- Template and emergency-participation warnings are visible.

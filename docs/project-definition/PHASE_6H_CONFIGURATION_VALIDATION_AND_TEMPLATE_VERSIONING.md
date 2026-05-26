# Phase 6H: Configuration Validation and Template Versioning

This pass advances MVP configuration safety before runtime use.

- Adds `Configuration Validation` for Actions, Templates, EGMs, and Groups.
- Validates action target selection, template action keys, matcher JSON, endpoint quirks JSON, return-action coverage, and group membership.
- Surfaces validation status in existing Operator pages:
  - `/operator/actions`
  - `/operator/templates`
  - `/operator/egms`
- Adds read-only API:
  - `GET /api/v2/config-validation`
- Protects active template versions from silent overwrite:
  - active versions in use must be replaced by creating a new version, not edited in place.

Scope notes:

- No new top-level page.
- No runtime delivery/input execution behavior changes.
- No fake or simulator behavior.

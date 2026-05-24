# Phase 1B: API-Backed Configuration Surface

## Scope

Phase 1B exposes the Phase 1A domain foundation through minimal, validated, store-backed `/api/v2` endpoints.

Implemented in this phase:
- store coverage for get/upsert/list across inputs, actions, templates, and EGMs,
- store-backed list endpoints for message journal and audit timeline,
- small `internal/api` server with mutation auth hook,
- `/api/v2` route handlers with JSON validation and path/body ID checks,
- focused store and API tests.

## Endpoints

Read endpoints:
- `GET /api/v2/inputs`
- `GET /api/v2/actions`
- `GET /api/v2/templates`
- `GET /api/v2/egms`
- `GET /api/v2/comms/messages`
- `GET /api/v2/audit/timeline`

Mutation endpoints:
- `PUT /api/v2/inputs/{id}`
- `PUT /api/v2/actions/{id}`
- `PUT /api/v2/templates/{id}`
- `PUT /api/v2/egms/{id}`

Behavior rules:
- list endpoints return JSON arrays,
- mutation endpoints validate payloads before storage,
- body ID must match path ID when provided,
- bad JSON returns `400`,
- validation failure returns `400`,
- unauthorized mutation returns `401`,
- store/runtime failures return `500`.

## Intentionally Not Included

Not implemented in Phase 1B:
- GPIO hardware reading,
- input transition runtime engine,
- action execution runtime,
- real G2S transport sending,
- UI rebuild,
- fake/virtual EGM product features,
- compliance-gating behavior that blocks emergency operation.

## Definition Of Done

Phase 1B is done when:
- `/api/v2` handlers compile and pass tests,
- store coverage and tests exist for required entities,
- mutation auth is enforced via hook on PUT routes,
- formatting and full `go test ./...` pass,
- runtime behavior changes are limited to optional `/api/v2` route exposure.

## Next Expected Phase

Phase 2: Input engine implementation (GPIO abstraction, normal/triggered evaluation, transition generation, and transition audit integration), while keeping action execution and G2S transport work for later phases.

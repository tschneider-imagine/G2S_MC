# Release Gate Run (2026-05-20T18:17:29Z)

- API_BASE: `http://127.0.0.1:8444`
- API_TOKEN: set

## Summary

```text
check                 result detail
healthz               PASS   HTTP 200
readyz                PASS   HTTP 200
status                PASS   HTTP 200
cabinet-preflight     PASS   overall=PASS
api-auth-smoke        PASS   auth_required=YES; POST /api/certificates/import (with token) -> 400 (PASS auth passed + payload validation failed)
overall               PASS   release gate passed
```

## Cabinet Preflight JSON

```json
{"overall":"PASS","checks":[{"id":"service_readiness","result":"PASS","message":"Readiness check is healthy for /readyz policy","detail":"overall=READY_LAB"},{"id":"cabinet_profile","result":"PASS","message":"Cabinet profile is complete","detail":"wire_host_url=https://tspi4.local:8444/g2s; host_id=HOST-TSPI4-001"},{"id":"profile_source","result":"PASS","message":"Profile source is explicit","detail":"profile_source=file"},{"id":"certificate_mode_requirements","result":"PASS","message":"No certificates are required by the current runtime mode","detail":"g2s.require_tls=false; g2s.require_client_cert=false"},{"id":"certificate_san_wire_identity","result":"PASS","message":"Wire identity SAN check is skipped for certificate-optional runtime mode","detail":"g2s.require_tls=false; g2s.require_client_cert=false; reason=web_server_cert path is empty"}],"blockers":[],"timestamp":"2026-05-20T18:17:29.142327999Z"}

```

## API Auth Smoke Output

```text
GET /healthz (no token) -> 200 (PASS)
GET /api/status (no token) -> 200 (PASS)
PUT /api/cabinet-profile (no token) -> 401 (PASS auth denied)
DELETE /api/cabinet-profile (no token) -> 401 (PASS auth denied)
POST /api/certificates/import (no token) -> 401 (PASS auth denied)
auth_required_by_runtime -> YES
PUT /api/cabinet-profile (with token) -> 200 (PASS strict success)
DELETE /api/cabinet-profile (with token) -> 200 (PASS strict success)
POST /api/certificates/import (with token) -> 400 (PASS auth passed + payload validation failed)
overall -> PASS

```

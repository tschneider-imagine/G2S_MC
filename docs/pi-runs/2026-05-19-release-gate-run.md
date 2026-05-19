# Release Gate Run (2026-05-19T21:47:06Z)

- API_BASE: `http://127.0.0.1:8444`
- API_TOKEN: not set

## Summary

```text
check                 result detail
healthz               FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
readyz                FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
status                FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
cabinet-preflight     FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
api-auth-smoke        FAIL   GET /healthz (no token) -> 000 (FAIL expected 200)
overall               FAIL   release gate failed
```

## Cabinet Preflight JSON

_Unavailable in this run._

## API Auth Smoke Output

```text
curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
GET /healthz (no token) -> 000 (FAIL expected 200)
GET /api/status (no token) -> 000 (FAIL expected 200)
PUT /api/cabinet-profile (no token) -> 000 (FAIL expected 401/403 auth denied)
DELETE /api/cabinet-profile (no token) -> 000 (FAIL expected 401/403 auth denied)
POST /api/certificates/import (no token) -> 000 (FAIL expected 401/403 auth denied)
auth_required_by_runtime -> NO
PUT /api/cabinet-profile (with token) -> SKIP (SKIP token not provided)
DELETE /api/cabinet-profile (with token) -> SKIP (SKIP token not provided)
POST /api/certificates/import (with token) -> SKIP (SKIP token not provided)
overall -> FAIL

```

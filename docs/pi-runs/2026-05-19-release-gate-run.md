# Release Gate Run (2026-05-19T21:25:47Z)

- API_BASE: `http://127.0.0.1:8444`
- API_TOKEN: not set

## Summary

```text
check                 result detail
healthz               FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
readyz                FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
status                FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
cabinet-preflight     FAIL   HTTP 000; curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
api-auth-smoke        FAIL   curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server
overall               FAIL   release gate failed
```

## Cabinet Preflight JSON

_Unavailable in this run._

## API Auth Smoke Output

```text
curl: (7) Failed to connect to 127.0.0.1 port 8444 after 0 ms: Could not connect to server

```

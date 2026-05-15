# Development Notes

## Local prerequisites

- Go 1.22 or newer
- OpenSSL or equivalent certificate inspection tooling
- Git

## First commands

```powershell
go test ./...
go run ./cmd/g2s-mute -config .\configs\config.example.json -simulate-trigger
```

The default development listener binds to `127.0.0.1:8444`.

Useful endpoints:

- `GET http://127.0.0.1:8444/healthz`
- `GET http://127.0.0.1:8444/api/status`
- `POST http://127.0.0.1:8444/g2s`

## Fake comms-on-line request

```xml
<g2sBody egmId="EGM-01">
  <commsOnLine/>
</g2sBody>
```

The listener returns a minimal `commsOnLineAck` fixture and updates the in-memory EGM status.

## Current dependency stance

The first scaffold intentionally uses only the Go standard library. SQLite migrations are present, but the actual database driver is not pinned yet. Add the SQLite driver after Go is installed and we can run module and build verification locally.

# Development Notes

## Local prerequisites

- Go 1.25 or newer
- OpenSSL or equivalent certificate inspection tooling
- Git

## First commands

```powershell
go test ./...
go run ./cmd/g2s-mute -config .\configs\config.example.json -simulate-trigger
```

In a second terminal:

```powershell
go run ./cmd/g2s-fake-egm -host-url http://127.0.0.1:8444/g2s -egm-id EGM-01 -keepalive-count 3
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

The fake EGM command sends this request and then follows with one or more `keepAlive` messages.

## Current dependency stance

The service uses `modernc.org/sqlite` for a pure-Go SQLite driver. This keeps the Windows and Raspberry Pi development path simpler than a CGO-backed driver.

The development config writes the audit database to `./data/controller.db`, which is intentionally ignored by Git.

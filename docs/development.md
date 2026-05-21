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

For a sustained multi-EGM lab run from the roster:

```bash
bash ./scripts/multi-fake-egm.sh
```

The launcher reads `egm_roster` from `configs/config.pi.example.json` by default. For the 3-EGM lab example:

```bash
CONFIG_PATH=./configs/config.pi.multi-fake.example.json KEEPALIVE_COUNT=-1 bash ./scripts/multi-fake-egm.sh
```

Use `Ctrl+C` to stop the continuous keepAlive clients. Set `EGM_IDS=EGM-01,EGM-03` to limit the run to a subset of the roster.

The default development listener binds to `127.0.0.1:8444`.

Useful endpoints:

- `GET http://127.0.0.1:8444/`
- `GET http://127.0.0.1:8444/dashboard`
- `GET http://127.0.0.1:8444/healthz`
- `GET http://127.0.0.1:8444/api/status`
- `GET http://127.0.0.1:8444/api/incidents`
- `GET http://127.0.0.1:8444/api/egms/history`
- `GET http://127.0.0.1:8444/api/compliance`
- `GET http://127.0.0.1:8444/api/state-history`
- `GET http://127.0.0.1:8444/api/certificates`
- `POST http://127.0.0.1:8444/g2s`

## Local mutual TLS flow

Generate a throwaway development CA, host certificate, and fake EGM client certificate:

```powershell
go run ./cmd/g2s-dev-certs -out .\certs
```

Start the controller with HTTPS and client-certificate verification enabled:

```powershell
go run ./cmd/g2s-mute -config .\configs\config.tls.example.json
```

In a second terminal, connect the fake EGM with the generated CA and client certificate:

```powershell
go run ./cmd/g2s-fake-egm -host-url https://localhost:8444/g2s -egm-id EGM-01 -ca .\certs\ca.crt -cert .\certs\client.crt -key .\certs\client.key -keepalive-count 3
```

The generated certificate files live under `./certs`, which is ignored by Git. Regenerate them whenever the lab host URL changes, because the DNS name or IP used on the wire must match the host certificate SAN.

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

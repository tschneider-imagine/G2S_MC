# Windows Setup

## Required tools

- Go 1.25 or newer
- Git
- OpenSSL or equivalent TLS inspection tooling

## Verify local setup

```powershell
go version
git status --short
.\scripts\check.ps1
```

## Run the development controller

```powershell
go run ./cmd/g2s-mute -config ./configs/config.example.json -simulate-trigger
```

The default listener binds to `127.0.0.1:8444`.

Useful checks:

```powershell
Invoke-WebRequest http://127.0.0.1:8444/healthz
Invoke-WebRequest http://127.0.0.1:8444/api/status
```

## Install hints

If `go` is missing, install Go from https://go.dev/dl/ or through a package manager already trusted on the workstation.

If `openssl` is missing, install OpenSSL through the workstation's approved package/source path. We only need it for certificate and TLS bring-up checks; it is not required to compile the first Go scaffold.

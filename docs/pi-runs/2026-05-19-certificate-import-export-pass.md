# Pi Run Report: Certificate Import/Export API (2026-05-19)

## Commands Run

1. `go test ./...`
2. `sudo bash ./scripts/pi-install.sh`
3. `sudo systemctl restart g2s-mute.service`
4. `sudo g2s-dev-certs -out /tmp/g2s-cert-api`
5. Build import payload JSON from generated PEM files:
   - `CERT_JSON=$(sed ':a;N;$!ba;s/\n/\\n/g' /tmp/g2s-cert-api/client.crt)`
   - `KEY_JSON=$(sed ':a;N;$!ba;s/\n/\\n/g' /tmp/g2s-cert-api/client.key)`
   - `printf '{"role":"g2s_client_cert","certificate_pem":"%s","private_key_pem":"%s"}' "$CERT_JSON" "$KEY_JSON" > /tmp/g2s-cert-import.json`
6. `curl -fsS -X POST http://127.0.0.1:8444/api/certificates/import -H 'Content-Type: application/json' --data-binary @/tmp/g2s-cert-import.json`
7. `curl -fsS 'http://127.0.0.1:8444/api/certificates/export?role=g2s_client_cert'`
8. `curl -sS -i 'http://127.0.0.1:8444/api/certificates/export?role=g2s_client_cert&include_key=true'`
9. `curl -fsS http://127.0.0.1:8444/api/certificates`
10. `systemctl status g2s-mute.service --no-pager --full`

## Results

- `go test ./...`: PASS
- Install + restart: PASS
- `POST /api/certificates/import`: PASS
  - imported role: `g2s_client_cert`
  - cert path: `/etc/g2s-mute/certs/client.crt`
  - key path: `/etc/g2s-mute/certs/client.key`
  - backups created:
    - `/etc/g2s-mute/certs/client.crt.bak-20260519T145606Z`
    - `/etc/g2s-mute/certs/client.key.bak-20260519T145606Z`
- `GET /api/certificates/export?role=g2s_client_cert`: PASS (certificate returned)
- `GET /api/certificates/export?...&include_key=true`: PASS (expected deny)
  - HTTP `403 Forbidden`
  - message: `private key export is disabled by web_ui.allow_private_key_export`
- `GET /api/certificates`: PASS
  - `g2s_ca_cert` status `VALID`
  - `g2s_client_cert` status `VALID`
  - `web_server_cert` status `NOT_CONFIGURED` (lab mode)
- `systemctl status`: PASS
  - service `active (running)`

## Files Changed

- `cmd/g2s-mute/certificates_api.go`
- `cmd/g2s-mute/certificates_api_test.go`
- `cmd/g2s-mute/main.go`
- `configs/config.example.json`
- `configs/config.pi.example.json`
- `configs/config.tls.example.json`
- `docs/raspberry-pi.md`
- `docs/pi-runs/2026-05-19-certificate-import-export-pass.md`
- `internal/config/config.go`
- `packaging/systemd/g2s-mute.service`


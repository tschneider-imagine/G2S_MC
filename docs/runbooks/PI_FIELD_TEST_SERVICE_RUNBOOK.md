# Pi Appliance Service Runbook

## Install

From the repository root on the Pi:

```bash
sudo ./packaging/install/install-pi-field-test.sh --enable --start
```

Dry run:

```bash
./packaging/install/install-pi-field-test.sh --dry-run
```

## Service Operations

```bash
sudo systemctl start g2s-mute.service
sudo systemctl stop g2s-mute.service
sudo systemctl restart g2s-mute.service
systemctl status g2s-mute.service --no-pager --full
journalctl -u g2s-mute.service -n 120 --no-pager
```

## Runtime Paths

- config: `/etc/g2s-mute/config.json`
- database: `/var/lib/g2s-mute/controller.db`
- binary: `/usr/local/bin/g2s-mute`
- unit: `/etc/systemd/system/g2s-mute.service`

## Verify Operator Console and Routes

```bash
./packaging/install/verify-pi-field-test.sh
```

Strict revision mismatch check:

```bash
./packaging/install/verify-pi-field-test.sh --strict
```

Expected key results:

- `200 /healthz`
- `200 /operator`
- `200 /operator/inputs`
- `200 /operator/actions`
- `200 /operator/comms`
- `200 /operator/egms`
- `200 /operator/templates`
- `200 /operator/audit`
- `200 /operator/settings`
- `404 /field-test`
- `404 /dashboard`
- `404 /static/dashboard.js`
- `404 /static/dashboard.css`
- `404 /operator/readiness`
- `404 /operator/readiness.json`
- `404 /operator/settings/system-check`
- `404 /operator/settings/system-check.json`

## Verify Input Runtime

The service unit enables input runtime polling and default input seeding:

- `-input-runtime`
- `-input-runtime-seed-defaults`
- `-input-runtime-interval 100ms`

Inspect recent runtime and transition logs:

```bash
journalctl -u g2s-mute.service -n 200 --no-pager
```

## Verify Running Service Revision

Check service runtime fingerprint from the running appliance:

```bash
curl -s http://127.0.0.1:8444/api/v2/runtime
```

Compare with local repository head:

```bash
git rev-parse --short HEAD
```

If they differ, the running service is likely an older installed binary and should be rebuilt/restarted.

## Read-Only Message Delivery Check API

Run a service-side read-only delivery check (no G2S payload send):

```bash
curl -s -X POST http://127.0.0.1:8444/api/v2/settings/message-delivery-check \
  -H 'Content-Type: application/json' \
  -d '{"egm_id":"EGM-001","template_id":"template-generic-g2s-action","template_action_key":"emergency_broadcast_silence","include_network_check":false,"include_tls_check":false,"timeout_ms":5000}'
```

This check uses the running service context and does not require shell read access to service-owned config/database files.

## Delivery and Execution Defaults

- delivery remains disabled by default unless explicitly configured at runtime.
- service defaults do not enable runtime action execution.

## Rollback / Uninstall

```bash
sudo ./packaging/install/uninstall-pi-field-test.sh
```

Optional full cleanup:

```bash
sudo ./packaging/install/uninstall-pi-field-test.sh --remove-config --purge-data
```

# Packaging

This folder contains appliance service packaging for Raspberry Pi deployment.

## Systemd Unit

- `packaging/systemd/g2s-mute.service`

The unit starts `g2s-mute` with:

- `/etc/g2s-mute/config.json`

Runtime mode now comes from config (`runtime.*`) rather than hidden systemd flags.
CLI flags remain available and override config only when explicitly supplied.
Message delivery remains disabled by default unless explicitly configured.

## Install Scripts

- `packaging/install/install-pi-field-test.sh`
- `packaging/install/uninstall-pi-field-test.sh`
- `packaging/install/verify-pi-field-test.sh`

Install example:

```bash
sudo ./packaging/install/install-pi-field-test.sh --enable --start
```

Dry run:

```bash
./packaging/install/install-pi-field-test.sh --dry-run
```

Verification:

```bash
./packaging/install/verify-pi-field-test.sh
```

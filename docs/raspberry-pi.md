# Raspberry Pi Bring-Up

This guide assumes the target Pi already has Git, Go, and Codex available.

## Install

From the repo on the Pi:

```bash
sudo bash ./scripts/pi-install.sh
```

To install and immediately start the systemd service:

```bash
sudo bash ./scripts/pi-install.sh --start
```

The installer builds the Go binaries, installs the service, creates the service user, and creates these paths:

- `/usr/local/bin/g2s-mute`
- `/usr/local/bin/g2s-fake-egm`
- `/usr/local/bin/g2s-dev-certs`
- `/etc/g2s-mute/config.json`
- `/etc/g2s-mute/certs`
- `/var/lib/g2s-mute`
- `/var/log/g2s-mute`

## First Local Smoke Test

Stop the service if it is already running, then run the foreground smoke test:

```bash
sudo systemctl stop g2s-mute.service
sudo CONFIG_PATH=/etc/g2s-mute/config.json bash ./scripts/pi-smoke.sh
```

Expected result:

```text
commsOnLine -> HTTP 200
keepAlive 1 -> HTTP 200
Pi smoke test passed
```

## Service Commands

```bash
sudo systemctl restart g2s-mute.service
sudo systemctl status g2s-mute.service
journalctl -u g2s-mute.service -f
```

The dashboard should be available at:

```text
http://<pi-ip>:8444/dashboard
```

## TLS Lab Mode

For a local certificate test on the Pi:

```bash
sudo g2s-dev-certs -out /etc/g2s-mute/certs
sudo chown -R g2s-mute:g2s-mute /etc/g2s-mute/certs
```

Then edit `/etc/g2s-mute/config.json` to use the certificate paths from `configs/config.tls.example.json`, update `g2s.host_url` to the real Pi DNS name or IP, and ensure that value exists in the host certificate SAN before expecting real cabinet TLS to work.

## Next Hardware Checks

Before enabling real GPIO behavior, confirm:

- security line GPIO pin, voltage level, polarity, and isolation
- PSU 1 and PSU 2 GPIO pins and polarity
- buzzer driver circuit, GPIO pin, and safe current limits
- whether the Pi will use SSD storage and a UPS/power-loss strategy

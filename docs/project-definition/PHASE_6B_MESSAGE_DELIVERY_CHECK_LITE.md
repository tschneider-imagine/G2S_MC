# Phase 6B: Message Delivery Check Lite

## Scope

- Add a service-side Message Delivery Check under Settings.
- Use the running appliance store/config/certificate paths to evaluate configured delivery readiness.
- Report endpoint resolution, certificate/trust status, and optional TCP/TLS connectivity checks.

## Included

- Message Delivery Check panel on `/operator/settings`.
- Endpoint/method/content-type/header/timeout resolution using configured EGM and template data.
- Certificate and trust material status included in check result.
- Optional network and TLS checks only when explicitly requested.

## Not Included

- No G2S payload send from this check.
- No new top-level Operator page.
- No fake EGM behavior.
- No private key exposure.

## Notes

- Network/TLS checks are auth-gated because they initiate outbound connections.
- Missing endpoint/certificate configuration is reported as clear Error output.
- The check runs from the appliance service context using injected Store/Options; operator shell access to `/etc/g2s-mute/config.json` or `/var/lib/g2s-mute/controller.db` is not required.
- The check never sends a G2S payload and does not mutate action runs, message journal rows, or audit rows in read-only mode.
- Runtime output and error text are sanitized so private key material is never exposed.

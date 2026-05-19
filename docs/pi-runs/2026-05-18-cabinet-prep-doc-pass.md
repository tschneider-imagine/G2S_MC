# Pi Run Report - Cabinet Prep Doc Pass

Date: 2026-05-18

## Commands Run

- `cd ~/projects/G2S_MC`
- `git status --short`
- `sed -n '1,260p' docs/G2S_First_Cabinet_Lab_Runbook.md`
- `sed -n '1,260p' docs/G2S_Pre_NextStep_Readiness_Check.md`
- `sed -n '1,320p' docs/raspberry-pi.md`
- `sed -n '1,320p' docs/G2S_Complete_Executable_Project_Plan.md`
- `sed -n '1,260p' configs/config.example.json`
- `sed -n '1,260p' configs/config.tls.example.json`
- `sed -n '1,260p' configs/config.pi.example.json`
- `sed -n '1,260p' docs/First-Cabinet-Prep-Checklist.md`
- `sed -n '1,200p' docs/pi-runs/2026-05-18-cabinet-prep-doc-pass.md`
- `git status --short`

## Docs Changed

- `docs/First-Cabinet-Prep-Checklist.md`
- `docs/pi-runs/2026-05-18-cabinet-prep-doc-pass.md`

## What Is Now Explicit

- Frozen current identity defaults (host ID, host URL variants, endpoint path, EGM seed ID) and a parallel to-confirm column.
- Concrete certificate ownership/process expectations, file paths, and rotation checks.
- Network/registration/site-approval prerequisites before first cabinet attempt.
- Exact first-session request/response evidence to capture (`commsOnLine`, ACK, keepAlive, descriptor/option outputs).
- Ten yes/no go/no-go gates for first real cabinet session.

## Still Unknown / To Confirm

- Final cabinet-facing DNS/IP and corresponding SAN set.
- Final host ID and EGM test set IDs for real cabinet bank.
- Registered-host role and permission profile required by the target platform.
- Certificate source-of-truth owner and issuance process for cabinet environment.
- Site firewall/routing and formal approval dependencies for the first live session.


# Pipeline State

## Rule
PC is the writer.
Pi is deploy/verify.
One writer only.
No reset/rebase/stash/force without explicit approval.
Repo state and installed runtime state are separate.

## Locations
- GitHub remote: origin/rebuild/input-action-engine
- PC checkout: C:\Users\SnowM\Documents\GitHub\G2S_MC
- Pi checkout: /home/ts/projects/G2S_MC
- Pi runtime binary: /usr/local/bin/g2s-mute
- Pi config: /etc/g2s-mute/config.json
- Pi DB: /var/lib/g2s-mute/controller.db

## Current Known State
- Remote HEAD: 71b5fbd
- PC HEAD: 71b5fbd
- Pi repo HEAD: unknown
- Pi runtime revision: unknown
- Runtime service: g2s-mute.service
- Current PC task: Add PIPE_STATE.md to track repo/runtime pipeline state.
- Current Pi task: Deploy/verify only for the currently pushed PC commit.
- Last proven runtime flow: HOST_LISTENER prepared/pending flow, manual-clear return action, inbound confirmation, exports.
- Open blocker: none recorded.

## Pipeline
1. PC codes and pushes.
2. Pi pulls fast-forward only.
3. Pi builds binary.
4. Pi verifies binary metadata with go version -m.
5. Pi installs binary.
6. Pi restarts service.
7. Pi verifies /api/v2/runtime.
8. Pi performs only the direct acceptance check for that commit.

## Direct Acceptance Checks
- Runtime revision matches target commit.
- Changed route/API exists.
- No private key material exposed.
- No old binary running.
- No unrelated proof loops.

## Completed Facts
- All four GPIO inputs tested.
- GPIO21 trigger/manual-clear tested.
- HOST_LISTENER topology works.
- HOST_LISTENER prepared/pending flow works.
- Manual clear return action works.
- Return action inbound confirmation works.
- Exports work.
- Private key scans passed.

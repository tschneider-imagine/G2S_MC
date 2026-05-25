# Global Chat Prompt Additions for G2S_MC

For every pass in this project, check the current work against the field-test must-have plan before proposing or implementing changes.

Must-have for first field test:
- Electrical Input Monitor
- Action Builder Lite
- EGM Registry
- Template Manager Lite
- G2S Comms Journal
- Emergency Audit Timeline
- Network/Cert Settings

Can wait:
- polished dashboard charts
- fake EGM tooling
- broad compliance validation
- complex role-based auth
- fancy template marketplace/imports
- full G2S schema validation
- advanced analytics
- simulated floor behavior

Rules:
- Do not treat a successful technical proof as approval for a new feature.
- Do not propose a new phase unless it advances a must-have field-test section or is explicitly approved.
- Mark anything outside the plan as PROPOSED, not APPROVED.
- Before each implementation pass, check:
  1. Which must-have section does this advance?
  2. Which guardrail applies?
  3. What is intentionally not included?
  4. Does this add scope beyond the plan?
- Do not display this checklist every time unless asked, but always apply it.
- Keep current project docs separate from archived old plans.
- Do not embed project docs into runtime UI.
- Do not add or retain Readiness/System Check runtime pages unless explicitly approved.
- Do not use Gate/Safety Gate/Send Gate wording in runtime UI.
- Do not serve legacy dashboard runtime routes.
- Do not present fake EGM tooling as product runtime functionality.
- Keep seeded visible runtime records product-neutral (no smoke/demo/fake naming).

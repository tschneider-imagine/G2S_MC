# internal/inbound

This package captures inbound controller messages, records them in the message journal, and applies best-effort correlation to action runs and target results.

Scope:

- best-effort inbound parsing from headers/query/body,
- inbound message journal entry creation,
- optional action run/target correlation,
- matcher-based target confirmation/failure updates,
- audit timeline evidence for receive/correlation/match outcomes.

Out of scope:

- transport send behavior,
- action planning,
- GPIO/input polling,
- fake/simulated EGM behavior.

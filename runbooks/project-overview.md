# Project overview

Moved verbatim from the root `AGENTS.md`; the root keeps the summary
paragraph and routes here.

**bus-accounts** maintains the chart of accounts and canonical account-group hierarchy as schema-validated repository data and provides stable account references for downstream modules (journal, invoice, budget, reporting). Implement only this module’s scope.

- **Purpose:** Own and maintain `accounts.csv` plus `account-groups.csv` and their beside-the-table schemas at the effective workspace root; enforce uniqueness, allowed account types, deterministic group hierarchy rules, and first-class CLI flows for `init`, `list`, `add`, `set`, `groups`, `validate`, and `sole-proprietor` (withdrawal/investment).
- **Inputs and outputs:** Reads and writes `accounts.csv` and `accounts.schema.json` at the effective workspace root (directory set by `-C <dir>` when supplied, otherwise the process working directory). Command results → stdout or the file given by `--output`. Help and version → stdout; diagnostics, validation messages, and errors → stderr. No network I/O; no Git execution ([Error handling, dry-run, and diagnostics](https://docs.busdk.com/cli/error-handling-dry-run-diagnostics)).
- **Binary and invocation:** Binary `bus-accounts`. Invoked by the dispatcher as `bus accounts …`. Follow [CLI command naming](https://docs.busdk.com/cli/command-naming).
- **Non-goals:** Do not implement other modules’ logic; do not access the network; do not execute Git commands. Do not introduce breaking identifier changes without a documented migration plan and coordination with dependent modules.
- **Spec compliance:** https://docs.busdk.com is canonical. When the design document and local code or layout conflict, change the implementation to match the spec. Do not document or preserve behavior that diverges from the spec.

---

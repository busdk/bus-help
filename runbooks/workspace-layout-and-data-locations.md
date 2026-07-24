# Workspace layout and data locations

Moved verbatim from the root `AGENTS.md`; the root keeps the summary
paragraph and routes here.

- **Accounts dataset:** `accounts.csv` with beside-the-table schema `accounts.schema.json`. Both live at the **workspace root** (the effective working directory, e.g. set by `-C <dir>`). The [accounts area](https://docs.busdk.com/layout/accounts-area) is this chart-of-accounts data at workspace root; the module does **not** create or use an `accounts/` subdirectory ([bus-accounts SDD Data Design](sdd/docs/modules/bus-accounts.md), [bus-accounts CLI Files](https://docs.busdk.com/modules/bus-accounts)).
- **Finnish reporting hierarchy:** `account-groups.csv` is the only canonical reporting hierarchy owned by this module. Keep reporting meaning on the group tree plus `report_profiles`; do not reintroduce semantic-classification, statement-target, or layout-specific mapping datasets.
- **Data package:** After a successful `bus accounts init`, the workspace data package (`datapackage.json`) MUST contain a resource entry for the accounts table (path to `accounts.csv` and schema reference) so that workspace-level validation and discovery see the accounts dataset ([bus-accounts SDD](sdd/docs/modules/bus-accounts.md)).
- **Path ownership:** This module owns the path to the chart of accounts. Other modules that need read-only access to the accounts dataset MUST obtain the path from this module’s Go library (path accessors), not by hardcoding file names ([Data path contract for read-only cross-module access](sdd/docs/modules/modules.md#data-path-contract-for-read-only-cross-module-access), [bus-accounts SDD IF-ACC-002, NFR-ACC-002](sdd/docs/modules/bus-accounts.md)). Path accessors MUST be designed so that future dynamic path configuration can be supported without breaking consumers.
- **Optional reference datasets:** e.g. `entities.csv` may exist at workspace root; path ownership for each dataset follows the owning module. Keep diagnostics and examples aligned with [repository README expectations](https://docs.busdk.com/layout/repository-readme-expectations).
- **Schemas:** Table Schema JSON beside each dataset (no top-level `schemas/` directory). If present, keep root `datapackage.json` consistent with dataset changes ([Schemas area](https://docs.busdk.com/layout/schemas-area)).
- **Working directory:** Global `-C <dir>` / `--chdir` sets the effective working directory for resolving all paths ([Standard global flags](https://docs.busdk.com/cli/global-flags)).

---

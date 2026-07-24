# Code and design conventions

Moved verbatim from the root `AGENTS.md`; the root keeps the summary
paragraph and routes here.

- **Library-first:** Implement behavior in a Go library; the CLI is a thin wrapper for args, I/O, and output. Test via library APIs; do not rely on other `bus-*` CLIs for core behavior. Modules MUST NOT invoke other `bus-*` CLIs as internal dependencies for core behavior ([bus-accounts SDD](sdd/docs/modules/bus-accounts.md), [Module repository structure and dependency rules](https://docs.busdk.com/implementation/module-repository-structure)).
- **Init via bus-data library only:** All initialization of the accounts dataset and schema MUST go through the [bus-data](sdd/docs/modules/bus-data.md) Go library; the module must not invoke the bus-data CLI.
- **Path accessors:** The module MUST expose a Go library API that returns the workspace-relative path(s) to its owned data file(s) (e.g. accounts CSV and optionally the beside-the-table schema). Other modules that need read-only access to the chart of accounts MUST use this accessor, not hardcoded paths. The API MUST be designed so that future dynamic path configuration can be supported without breaking consumers ([bus-accounts SDD NFR-ACC-002, IF-ACC-002](sdd/docs/modules/bus-accounts.md), [Data path contract](sdd/docs/modules/modules.md#data-path-contract-for-read-only-cross-module-access)).
- **Mapping dataset contract:** The module defines and owns the schema contract for the account-to-statement mapping dataset (FR-ACC-006); bus-reports consumes it for fi-* statutory layouts.
- **Global flags:** Support [Standard global flags](https://docs.busdk.com/cli/global-flags) before the subcommand.
- **Reporting:** List and query commands must be deterministic and human-readable; provide a machine-readable format option when practical ([Reporting and query commands](https://docs.busdk.com/cli/reporting-and-queries)).
- **Code style:** Use the Makefile for format and lint (`make fmt`, `make lint`); keep code gofmt-formatted and vet-clean.
- **Spec compliance:** https://docs.busdk.com is canonical. Align implementation with the documented contract. When the design document and local code or layout conflict, change the implementation to match the spec (including migration plans for breaking changes). Do not document or preserve behavior that diverges from the spec.

---

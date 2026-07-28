# AGENTS.md — bus-help

## Highest-Priority Rule: Precise Language And Precise Reasoning

Precise language is part of precise reasoning. Before acting or reporting, name
the exact object, action, scope, evidence, and uncertainty. State what changed
and what did not change. Never use a broader claim than the evidence supports.
If an exact, unambiguous sentence cannot be written, inspect the evidence or
ask for clarification before proceeding.


Local module guidance:

- `bus-help` owns shared OpenCLI-compatible structs, Bus namespaced metadata structs, conversion helpers, and live metadata discovery helpers for the metadata-driven help/configuration feature.
- `bus-configure` may import reusable packages from `bus-help`, but `bus-help` must not import `bus-configure`.
- Do not make `bus-configure` a dependency of any other Bus module. Modules that expose metadata should depend on `bus-help` shared metadata packages or serialize their own compatible output.
- The source of truth for machine-readable help is live command stdout such as `bus journal help --format opencli` and `bus help --format opencli journal`; generated files are export/documentation outputs only.
- E2E tests that use fake `bus-*` binaries must run commands from the isolated
  fixture workspace, not from the module checkout, because discovery can also
  find built sibling `bus-*/bin/bus-*` binaries in a superproject checkout.

# Previous scaffold guidance

Agent-facing spec and conventions for the **bus-accounts** BusDK module. This file is the primary reference for AI coding agents in this repository. Follow the [AGENTS.md open format](https://agents.md/) (project overview, build and test, conventions, testing). **Canonical design:** [BusDK Design Document](https://docs.busdk.com/). **Module spec:** [bus-accounts SDD](sdd/docs/modules/bus-accounts.md), [bus-accounts CLI reference](https://docs.busdk.com/modules/bus-accounts). Treat https://docs.busdk.com as canonical; implement and document behavior that matches the module SDD and linked spec pages.

---

## Project overview

**bus-accounts** maintains the chart of accounts and canonical account-group hierarchy as schema-validated repository data and provides stable account references for downstream modules (journal, invoice, budget, reporting). Implement only this module’s scope.

Details: `runbooks/project-overview.md`.

## Build and test commands

Use the repository Makefile as the standard interface. Tests must be hermetic, deterministic, and require no network or external services, per [Testing strategy](https://docs.busdk.com/testing/testing-strategy).

| Target         | Action |
|----------------|--------|
| `make build`   | Produces `bin/bus-accounts` |
| `make test`    | `go test ./...` |
| `make test-e2e`| Runs `tests/e2e.sh` (after build) |
| `make fmt`     | `gofmt -w .` |
| `make lint`    | `go vet ./...` |
| `make check`   | fmt, lint, test, test-e2e |

The agent must use this Makefile for build, test, format, and lint and follow BusDK’s deterministic workflow expectations.

---

## Invocation and I/O contract

Details: `runbooks/invocation-and-io-contract.md`.

## Commands (spec surface)

Implement and maintain these under `bus accounts`:

Details: `runbooks/commands-spec-surface.md`.

## Workspace layout and data locations

Details: `runbooks/workspace-layout-and-data-locations.md`.

## Data contract

- **Encoding and format:** UTF-8 CSV, header row, comma delimiter ([CSV conventions](https://docs.busdk.com/data/csv-conventions)).
- **Schema:** Frictionless Table Schema JSON beside the CSV. Schema is authoritative for types, constraints, primary keys, and foreign keys ([Table Schema contract](https://docs.busdk.com/data/table-schema-contract)).
- **Account types (allowed):** asset, liability, equity, income, expense ([bus-accounts CLI](https://docs.busdk.com/modules/bus-accounts), [bus-accounts SDD](sdd/docs/modules/bus-accounts.md)).
- **Identifiers:** Enforce uniqueness of account identifiers. Use stable `*_id` columns and declared primary/foreign keys in Table Schema where applicable ([CSV conventions](https://docs.busdk.com/data/csv-conventions)). Do not change identifier semantics without an explicit migration plan and coordination with dependent modules.
- **Schema foreign key contract (FR-ACC-005):** `accounts.schema.json` MUST be valid Table Schema. If `foreignKeys` is present for `parent_code`, every entry’s `reference` MUST include both `resource` and `fields` (self-referencing: `reference.resource` empty string, `reference.fields` `"code"`). Missing or malformed `reference` is an immediate validation error. If the project does not enforce hierarchy via foreign keys, omit `foreignKeys` entirely; never include an incomplete `foreignKeys` entry. See [bus-accounts SDD FR-ACC-005](sdd/docs/modules/bus-accounts.md).

---

## Code and design conventions

Details: `runbooks/code-and-design-conventions.md`.

## Testing instructions

Details: `runbooks/testing-instructions.md`.

## Spec and reference links

| Topic | URL |
|-------|-----|
| BusDK design spec | https://docs.busdk.com/ |
| BusDK SDD (single-page) | sdd/docs/sdd.md |
| bus-accounts SDD | sdd/docs/modules/bus-accounts.md |
| bus-accounts CLI (end-user) | https://docs.busdk.com/modules/bus-accounts |
| bus-reports SDD (mapping consumer) | sdd/docs/modules/bus-reports.md |
| bus-data SDD (init library) | sdd/docs/modules/bus-data.md |
| bus-init (module init contract FR-INIT-003) | sdd/docs/modules/bus-init.md |
| Data path contract (read-only cross-module) | sdd/docs/modules/modules.md#data-path-contract-for-read-only-cross-module-access |
| Accounts area | https://docs.busdk.com/layout/accounts-area |
| Data directory layout (index) | https://docs.busdk.com/layout/index |
| Data directory layout (principles) | https://docs.busdk.com/layout/layout-principles |
| Minimal example layout | https://docs.busdk.com/layout/minimal-example-layout |
| Schemas beside datasets | https://docs.busdk.com/layout/schemas-area |
| CSV conventions | https://docs.busdk.com/data/csv-conventions |
| Table Schema contract | https://docs.busdk.com/data/table-schema-contract |
| Error handling, dry-run, diagnostics | https://docs.busdk.com/cli/error-handling-dry-run-diagnostics |
| Validation and safety checks | https://docs.busdk.com/cli/validation-and-safety-checks |
| CLI command naming | https://docs.busdk.com/cli/command-naming |
| Standard global flags | https://docs.busdk.com/cli/global-flags |
| Non-interactive use and scripting | https://docs.busdk.com/cli/interactive-and-scripting-parity |
| Reporting and queries | https://docs.busdk.com/cli/reporting-and-queries |
| Testing strategy | https://docs.busdk.com/testing/testing-strategy |
| Testing index | https://docs.busdk.com/testing/index |
| Module repository structure | https://docs.busdk.com/implementation/module-repository-structure |

## Gitignore Rule

1. .bus MUST be tracked; never add .bus or .bus/ to .gitignore.
2. In private repositories, .bus/ must be tracked; .bus/secrets may be tracked in private repositories only and must not be tracked otherwise.
3. Runtime lock artifacts such as .bus-dev.lock may be ignored.

## Shared Superproject Conventions

Details: `runbooks/shared-superproject-conventions.md`.

## Global unit documentation traceability rule

- Every top-level production-code unit (`func`, `type`, `var`, and `const` blocks when they define global API/behavior) must include an inline comment that states its purpose.
- For each top-level global unit, also include concise `Used by:` traceability in the inline comment (or immediately adjacent comment) that names the primary caller(s), owning flow, or integration point.
- Keep `Used by:` comments accurate when refactoring: update or remove stale references in the same change set.
- Do not add new undocumented top-level global units.

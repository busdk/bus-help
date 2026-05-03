# AGENTS.md — bus-help

Local module guidance:

- `bus-help` owns shared OpenCLI-compatible structs, Bus namespaced metadata structs, conversion helpers, and live metadata discovery helpers for the metadata-driven help/configuration feature.
- `bus-configure` may import reusable packages from `bus-help`, but `bus-help` must not import `bus-configure`.
- Do not make `bus-configure` a dependency of any other Bus module. Modules that expose metadata should depend on `bus-help` shared metadata packages or serialize their own compatible output.
- The source of truth for machine-readable help is live command stdout such as `bus journal help --format opencli` and `bus help --format opencli journal`; generated files are export/documentation outputs only.
- E2E tests that use fake `bus-*` binaries must run commands from the isolated
  fixture workspace, not from the module checkout, because discovery can also
  find built sibling `bus-*/bin/bus-*` binaries in a superproject checkout.

# Previous scaffold guidance

Merged guidance from `.cursor/rules/*.mdc`.

Agent-facing spec and conventions for the **bus-accounts** BusDK module. This file is the primary reference for AI coding agents in this repository. Follow the [AGENTS.md open format](https://agents.md/) (project overview, build and test, conventions, testing). **Canonical design:** [BusDK Design Document](https://docs.busdk.com/). **Module spec:** [bus-accounts SDD](sdd/docs/modules/bus-accounts.md), [bus-accounts CLI reference](https://docs.busdk.com/modules/bus-accounts). Treat https://docs.busdk.com as canonical; implement and document behavior that matches the module SDD and linked spec pages.

---

## Project overview

**bus-accounts** maintains the chart of accounts and canonical account-group hierarchy as schema-validated repository data and provides stable account references for downstream modules (journal, invoice, budget, reporting). Implement only this module’s scope.

- **Purpose:** Own and maintain `accounts.csv` plus `account-groups.csv` and their beside-the-table schemas at the effective workspace root; enforce uniqueness, allowed account types, deterministic group hierarchy rules, and first-class CLI flows for `init`, `list`, `add`, `set`, `groups`, `validate`, and `sole-proprietor` (withdrawal/investment).
- **Inputs and outputs:** Reads and writes `accounts.csv` and `accounts.schema.json` at the effective workspace root (directory set by `-C <dir>` when supplied, otherwise the process working directory). Command results → stdout or the file given by `--output`. Help and version → stdout; diagnostics, validation messages, and errors → stderr. No network I/O; no Git execution ([Error handling, dry-run, and diagnostics](https://docs.busdk.com/cli/error-handling-dry-run-diagnostics)).
- **Binary and invocation:** Binary `bus-accounts`. Invoked by the dispatcher as `bus accounts …`. Follow [CLI command naming](https://docs.busdk.com/cli/command-naming).
- **Non-goals:** Do not implement other modules’ logic; do not access the network; do not execute Git commands. Do not introduce breaking identifier changes without a documented migration plan and coordination with dependent modules.
- **Spec compliance:** https://docs.busdk.com is canonical. When the design document and local code or layout conflict, change the implementation to match the spec. Do not document or preserve behavior that diverges from the spec.

---

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

- **Command results** → stdout, or to the file given by `--output`. Listings and machine-readable output only on stdout.
- **Help and version** → stdout; exit 0. When help or version is requested, ignore all other flags and arguments ([Standard global flags](https://docs.busdk.com/cli/global-flags)).
- **Diagnostics, validation messages, errors** → stderr. Verbose output to stderr only; verbose output must never be required for correctness.
- **Exit codes:** 0 success; 2 invalid usage (concise usage error on stderr); non-zero for schema/invariant violations or I/O failures ([Error handling, dry-run, and diagnostics](https://docs.busdk.com/cli/error-handling-dry-run-diagnostics)).
- **Quiet and output:** When `--quiet` is set, do not print command result output or informational messages; only errors may go to stderr. If both `--quiet` and `--output` are given, do not write to the output file or to stdout; still run the command and exit with the correct status ([Standard global flags](https://docs.busdk.com/cli/global-flags)).
- **Determinism:** Listings ordered by stable account identifiers; diagnostics use workspace-relative paths and stable identifiers ([Validation and safety checks](https://docs.busdk.com/cli/validation-and-safety-checks)).
- **Dry-run:** Mutating commands (`add`, `set`) must support `--dry-run` to preview file changes without writing ([Error handling, dry-run, and diagnostics](https://docs.busdk.com/cli/error-handling-dry-run-diagnostics)).
- **Scriptability:** Every command must be fully scriptable. All required input via arguments or flags. Missing required parameters → concise usage error on stderr, exit 2; no interactive prompts ([Non-interactive use and scripting](https://docs.busdk.com/cli/interactive-and-scripting-parity)).

Global flags (help, version, verbose, quiet, chdir, output, format, color) are defined in [Standard global flags](https://docs.busdk.com/cli/global-flags). Quiet and verbose are mutually exclusive; both supplied is invalid usage (exit 2). Parse flags in a testable module (e.g. `internal/cli/flags.go`) and pass parsed config into the run path. A lone `--` terminates global flag parsing; everything after it is positional for the subcommand.

---

## Commands (spec surface)

Implement and maintain these under `bus accounts`:

| Command     | Purpose |
|-------------|---------|
| `init`     | Create baseline accounts dataset and schema when absent. MUST use the [bus-data](sdd/docs/modules/bus-data.md) Go library only (no CLI invocation): (1) ensure workspace data package is initialized (create `datapackage.json` at workspace root when missing, per bus-data init); (2) create `accounts.csv` and `accounts.schema.json` via bus-data; (3) ensure `datapackage.json` contains a resource entry for the accounts table (path to `accounts.csv` and schema reference). Emitted schema MUST conform to [FR-ACC-005](sdd/docs/modules/bus-accounts.md): if `foreignKeys` is present for `parent_code`, each entry’s `reference` MUST include both `resource` and `fields` (self-referencing: `reference.resource` empty string, `reference.fields` `"code"`); if hierarchy is not enforced via foreign keys, omit `foreignKeys` entirely. If both `accounts.csv` and `accounts.schema.json` already exist and are consistent and the data package already contains the accounts resource, print a warning to stderr and exit 0 without modifying. If only one exists, or data is inconsistent, or the data package is missing when it should exist, fail with a clear error to stderr, do not write any file, and exit non-zero. Contract: [bus-init FR-INIT-003](sdd/docs/modules/bus-init.md), [bus-accounts SDD FR-ACC-003](sdd/docs/modules/bus-accounts.md). |
| `list`     | Print the chart of accounts in deterministic order (by stable account identifiers). Output format selectable via `--format`; default stable and documented (e.g. tsv). See [Reporting and query commands](https://docs.busdk.com/cli/reporting-and-queries). |
| `add`      | Add a new account. Parameters: `--code <account-id>`, `--name <account-name>`, `--type <asset|liability|equity|income|expense>`. MUST fail if an account with the same `--code` already exists: exit non-zero, emit a clear diagnostic to stderr, and do not modify the dataset ([FR-ACC-004](sdd/docs/modules/bus-accounts.md)). Validate before writing; support `--dry-run`. See [bus-accounts CLI](https://docs.busdk.com/modules/bus-accounts). |
| `set`      | Modify an existing account identified by `--code`. Accepts optional `--name` and `--type` to update those attributes. MUST fail if no account with the given code exists. Creation only via `add`; updates only via `set`. Support `--dry-run`. See [bus-accounts SDD](sdd/docs/modules/bus-accounts.md), [bus-accounts CLI](https://docs.busdk.com/modules/bus-accounts). |
| `validate` | Check both the accounts CSV content and the schema document (`accounts.schema.json`) against Table Schema and module invariants. MUST exit non-zero and print a clear error pointing to the schema file and the offending path when the schema is malformed (e.g. `foreignKeys` with missing `reference.resource` or malformed `reference`). See [bus-accounts SDD](sdd/docs/modules/bus-accounts.md) Error handling, [bus-accounts CLI](https://docs.busdk.com/modules/bus-accounts). |
| `sole-proprietor` | Suggest balanced double-entry for owner withdrawal (yksityisotto) or investment (yksityissijoitus). Subcommands: `withdrawal`, `investment`. Requires `--equity-code`, `--cash-code`, `--amount`. Does not read or write `accounts.csv`. Output default TSV (code, side, amount per line) for use with `bus journal add`. See [bus-accounts CLI](https://docs.busdk.com/modules/bus-accounts). |

Refuse to write invalid data; validate before any mutation ([Validation and safety checks](https://docs.busdk.com/cli/validation-and-safety-checks)).

---

## Workspace layout and data locations

- **Accounts dataset:** `accounts.csv` with beside-the-table schema `accounts.schema.json`. Both live at the **workspace root** (the effective working directory, e.g. set by `-C <dir>`). The [accounts area](https://docs.busdk.com/layout/accounts-area) is this chart-of-accounts data at workspace root; the module does **not** create or use an `accounts/` subdirectory ([bus-accounts SDD Data Design](sdd/docs/modules/bus-accounts.md), [bus-accounts CLI Files](https://docs.busdk.com/modules/bus-accounts)).
- **Finnish reporting hierarchy:** `account-groups.csv` is the only canonical reporting hierarchy owned by this module. Keep reporting meaning on the group tree plus `report_profiles`; do not reintroduce semantic-classification, statement-target, or layout-specific mapping datasets.
- **Data package:** After a successful `bus accounts init`, the workspace data package (`datapackage.json`) MUST contain a resource entry for the accounts table (path to `accounts.csv` and schema reference) so that workspace-level validation and discovery see the accounts dataset ([bus-accounts SDD](sdd/docs/modules/bus-accounts.md)).
- **Path ownership:** This module owns the path to the chart of accounts. Other modules that need read-only access to the accounts dataset MUST obtain the path from this module’s Go library (path accessors), not by hardcoding file names ([Data path contract for read-only cross-module access](sdd/docs/modules/modules.md#data-path-contract-for-read-only-cross-module-access), [bus-accounts SDD IF-ACC-002, NFR-ACC-002](sdd/docs/modules/bus-accounts.md)). Path accessors MUST be designed so that future dynamic path configuration can be supported without breaking consumers.
- **Optional reference datasets:** e.g. `entities.csv` may exist at workspace root; path ownership for each dataset follows the owning module. Keep diagnostics and examples aligned with [repository README expectations](https://docs.busdk.com/layout/repository-readme-expectations).
- **Schemas:** Table Schema JSON beside each dataset (no top-level `schemas/` directory). If present, keep root `datapackage.json` consistent with dataset changes ([Schemas area](https://docs.busdk.com/layout/schemas-area)).
- **Working directory:** Global `-C <dir>` / `--chdir` sets the effective working directory for resolving all paths ([Standard global flags](https://docs.busdk.com/cli/global-flags)).

---

## Data contract

- **Encoding and format:** UTF-8 CSV, header row, comma delimiter ([CSV conventions](https://docs.busdk.com/data/csv-conventions)).
- **Schema:** Frictionless Table Schema JSON beside the CSV. Schema is authoritative for types, constraints, primary keys, and foreign keys ([Table Schema contract](https://docs.busdk.com/data/table-schema-contract)).
- **Account types (allowed):** asset, liability, equity, income, expense ([bus-accounts CLI](https://docs.busdk.com/modules/bus-accounts), [bus-accounts SDD](sdd/docs/modules/bus-accounts.md)).
- **Identifiers:** Enforce uniqueness of account identifiers. Use stable `*_id` columns and declared primary/foreign keys in Table Schema where applicable ([CSV conventions](https://docs.busdk.com/data/csv-conventions)). Do not change identifier semantics without an explicit migration plan and coordination with dependent modules.
- **Schema foreign key contract (FR-ACC-005):** `accounts.schema.json` MUST be valid Table Schema. If `foreignKeys` is present for `parent_code`, every entry’s `reference` MUST include both `resource` and `fields` (self-referencing: `reference.resource` empty string, `reference.fields` `"code"`). Missing or malformed `reference` is an immediate validation error. If the project does not enforce hierarchy via foreign keys, omit `foreignKeys` entirely; never include an incomplete `foreignKeys` entry. See [bus-accounts SDD FR-ACC-005](sdd/docs/modules/bus-accounts.md).

---

## Code and design conventions

- **Library-first:** Implement behavior in a Go library; the CLI is a thin wrapper for args, I/O, and output. Test via library APIs; do not rely on other `bus-*` CLIs for core behavior. Modules MUST NOT invoke other `bus-*` CLIs as internal dependencies for core behavior ([bus-accounts SDD](sdd/docs/modules/bus-accounts.md), [Module repository structure and dependency rules](https://docs.busdk.com/implementation/module-repository-structure)).
- **Init via bus-data library only:** All initialization of the accounts dataset and schema MUST go through the [bus-data](sdd/docs/modules/bus-data.md) Go library; the module must not invoke the bus-data CLI.
- **Path accessors:** The module MUST expose a Go library API that returns the workspace-relative path(s) to its owned data file(s) (e.g. accounts CSV and optionally the beside-the-table schema). Other modules that need read-only access to the chart of accounts MUST use this accessor, not hardcoded paths. The API MUST be designed so that future dynamic path configuration can be supported without breaking consumers ([bus-accounts SDD NFR-ACC-002, IF-ACC-002](sdd/docs/modules/bus-accounts.md), [Data path contract](sdd/docs/modules/modules.md#data-path-contract-for-read-only-cross-module-access)).
- **Mapping dataset contract:** The module defines and owns the schema contract for the account-to-statement mapping dataset (FR-ACC-006); bus-reports consumes it for fi-* statutory layouts.
- **Global flags:** Support [Standard global flags](https://docs.busdk.com/cli/global-flags) before the subcommand.
- **Reporting:** List and query commands must be deterministic and human-readable; provide a machine-readable format option when practical ([Reporting and query commands](https://docs.busdk.com/cli/reporting-and-queries)).
- **Code style:** Use the Makefile for format and lint (`make fmt`, `make lint`); keep code gofmt-formatted and vet-clean.
- **Spec compliance:** https://docs.busdk.com is canonical. Align implementation with the documented contract. When the design document and local code or layout conflict, change the implementation to match the spec (including migration plans for breaking changes). Do not document or preserve behavior that diverges from the spec.

---

## Testing instructions

- **Unit tests:** Cover flag parsing, validation logic, schema enforcement, and deterministic listing. Use `go test ./...`; no network or external services ([Testing strategy](https://docs.busdk.com/testing/testing-strategy)).
- **Command-level / E2E:** Exercise `init`, `add`, `set`, `list`, `validate`, and `sole-proprietor` against fixture workspaces; assert exit codes, stdout/stderr, and (for mutating commands) resulting files. Use isolated fixture directories (e.g. under `tests/` or temp dirs) so tests do not share state. Tests MUST verify that `add` fails with non-zero exit and no dataset change when the account code already exists (FR-ACC-004), and that `set` can modify an existing account and fails when the account does not exist ([bus-accounts SDD Testing strategy](sdd/docs/modules/bus-accounts.md)).
- **Required regression and compatibility (per SDD):** (1) E2E or command-level test runs `bus accounts init` in an empty workspace, then `bus accounts validate`, and asserts success. (2) **Journal-add regression test:** The SDD-required test (after `bus accounts init` and minimal init for journal/period if needed, run a simple `bus journal add` that references accounts and assert it does not fail due to schema parsing or foreign key reference errors) is maintained in the **BusDK superproject** e2e when the bus-journal module is available. See the [BusDK superproject](https://github.com/busdk/busdk) and its e2e or integration test suite for the cross-module regression. This module’s e2e does not invoke other modules; do not add a duplicate test that depends on a `bus-journal` binary in this repo unless the superproject does not cover it. (3) Negative test: from an intentionally malformed `accounts.schema.json` (e.g. `foreignKeys` entry with missing `reference.resource`), assert `bus accounts validate` fails with a clear diagnostic pointing to the schema file and the offending foreign key. Implementers MUST keep these tests in place so the invalid-schema bug cannot recur.
- **Coverage:** Do not decrease coverage; add tests for new behavior and for any bug fix that can be reproduced by a test.

---

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

- Prefer minimal, deterministic, script-friendly behavior.
- This module AGENTS.md must remain usable when the module is checked out on its own; duplicate any operational guidance needed to work in this module instead of assuming the superproject root is present.
- Prefer reusable, approval-friendly commands over ad hoc shell recipes: use this module's standard interfaces first (for example make test, make e2e, make check, go test ./...) instead of long bash -lc, pipelines, temporary trace files, or chained command sequences.
- When a standard Makefile target exists for the needed action in this module, use that target as the default command surface before falling back to lower-level commands.
- Keep commands simple and repeatable so they are easy to rerun locally and easy to approve; avoid one-off compound shell invocations unless no reusable interface exists.
- For read-only journal access from bus-accounts, use `bus-journal/paths` plus local CSV reading; do not import `bus-journal/store`, because it pulls in bus-journal validation paths that depend back on bus-accounts and creates an import cycle.
- When a command's default output depends on whether `--format` was explicitly supplied, preserve the explicitness bit when merging pre-command and post-command global flags; do not only copy the final format string.
- For `bus accounts groups`, keep global flags before the subcommand in docs and examples (`bus accounts --format tsv groups ...`) because command-specific long flags such as `--group-id`, `--as-of`, and `--opening-as-of` must not be routed through the simple trailing-global parser.
- Account-owned dataset bootstrap, read, and write helpers must go through the shared `bus-data` managed-table interface. Do not reintroduce raw schema/table I/O in storage or report helpers when `bus-data` already exposes the needed operation.
- Deletion safety: tracked paths use `git rm` (or `git rm --cached`), untracked paths use `rm`.
- When a system-level CLI command fails due to incorrect parameters, record the correct invocation in the most relevant `AGENTS.md`.
- On macOS/BSD `cat`, `-A` is unsupported; use `cat -vet` or `sed -n 'l'` to visualize tabs and line endings instead.
- On macOS/BSD `awk`, avoid using `in` as a variable name (`in` is reserved in `for (x in y)`); use names like `inside` instead.
- When running shell commands that contain backticks in regex/pattern arguments (for example with `rg`), wrap the full command in single quotes or escape backticks to avoid command-substitution parse errors.
- `rg` does not support look-around by default; use `rg --pcre2` when patterns require look-ahead/look-behind.
- Use `python3` (not `python`) for Python scripting in this environment.

## Global unit documentation traceability rule

- Every top-level production-code unit (`func`, `type`, `var`, and `const` blocks when they define global API/behavior) must include an inline comment that states its purpose.
- For each top-level global unit, also include concise `Used by:` traceability in the inline comment (or immediately adjacent comment) that names the primary caller(s), owning flow, or integration point.
- Keep `Used by:` comments accurate when refactoring: update or remove stale references in the same change set.
- Do not add new undocumented top-level global units.

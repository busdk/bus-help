# Commands (spec surface)

Moved verbatim from the root `AGENTS.md`; the root keeps the summary
paragraph and routes here.

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

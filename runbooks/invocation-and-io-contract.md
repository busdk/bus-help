# Invocation and I/O contract

Moved verbatim from the root `AGENTS.md`; the root keeps the summary
paragraph and routes here.

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

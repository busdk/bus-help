# Shared Superproject Conventions

Moved verbatim from the root `AGENTS.md`; the root keeps the summary
paragraph and routes here.

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

# bus-help

`bus-help` renders Bus command help and live machine-readable command metadata.
It is exposed through the dispatcher as `bus help`.

The primary metadata interface is command output, not generated files. A module
that opts in exposes metadata from its own binary, for example:

```sh
bus journal help --format opencli
bus help --format opencli journal
```

`--format opencli` emits an OpenCLI-compatible JSON document on stdout. Bus
extensions are stored under namespaced metadata keys such as:

- `io.busdk.profile`
- `io.busdk.environment`
- `io.busdk.config`
- `io.busdk.effects`
- `io.busdk.security`

The `io.busdk.environment` extension includes versioned dotenv and variable
metadata for tools such as `bus configure`.

## Usage

```sh
bus help
bus help journal
bus help env journal
bus help config journal
bus help --format opencli journal
bus help --format json journal
```

If a module has not opted into structured metadata, `bus-help` reports that
clearly and leaves normal module help behavior unchanged.

SDD: sdd/docs/modules/bus-help.md

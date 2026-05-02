# PLAN.md

- [x] Implement metadata-driven help MVP end to end: provide `bus-help` as a thin CLI over reusable OpenCLI/Bus metadata packages, support `bus help`, `bus help MODULE`, `bus help MODULE COMMAND...`, `bus help env MODULE`, `bus help config MODULE`, and `bus help --format opencli|json MODULE`, discover module-owned metadata by executing live module help commands with explicit argv and timeouts, degrade gracefully when metadata is unavailable, document the OpenCLI-compatible Bus metadata extensions, and verify with unit plus e2e coverage.

## Why
The orchestrator cannot run useful coding tasks until harness pods have a reproducible runtime/toolchain contract. The runtime image and pod contract establish the execution boundary for every run.

## What Changes
- Define the baseline harness image composition and runtime/tooling matrix required by the MVP.
- Specify the harness pod contract for workspace mounts, startup behavior, and secure credential injection.
- Add image validation expectations to prevent drift across rebuilds.

## Capabilities

### New Capabilities
- `harness-runtime`: reproducible harness image and pod execution contract for coding runs.

### Modified Capabilities
- None.

## Impact
- Affects Docker/build assets, runtime bootstrap scripts, Kubernetes pod templates, and secret handling paths.
- Creates the executable foundation for all session/run workloads.

## Why
The project has architecture and PRD docs but no executable platform baseline. We need a concrete bootstrap change so implementation can start in a predictable structure.

## What Changes
- Create the initial repository layout for API, operator, harness, and web surfaces.
- Add shared local development entry points (build/test/lint/bootstrap) and baseline configuration loading.
- Define deployment skeletons for Kubernetes resources and environment overlays.
- Standardize local Kubernetes development on `kind` with cluster lifecycle commands.
- Define a local image workflow for OrbStack (`docker build` + `kind load docker-image`) so local builds run without pushing to a remote registry.

## Capabilities

### New Capabilities
- `control-plane-bootstrap`: initial workspace, build, and deployment scaffolding for the coding-agent orchestrator.

### Modified Capabilities
- None.

## Impact
- Affects repository structure, CI/bootstrap scripts, deployment manifests, and local cluster/image developer workflow.
- Creates the baseline that all subsequent control-plane and runtime changes build on.

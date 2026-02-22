## Why
The orchestrator needs a Kubernetes-native contract for sessions and runs before API and UI work can safely proceed. Without CRDs and reconcile behavior, run lifecycle is undefined.

## What Changes
- Define Session and HarnessRun CRDs for desired state and observed status.
- Implement controller-runtime reconcile flows that create, monitor, and clean up run pods.
- Add status transition and condition reporting used by API and UI consumers.

## Capabilities

### New Capabilities
- `operator-reconcile`: declarative run orchestration via Session and HarnessRun resources.

### Modified Capabilities
- None.

## Impact
- Affects Kubernetes API surface (CRDs), operator controllers, and status semantics consumed by higher layers.
- Introduces core lifecycle boundaries for pod provisioning and teardown.

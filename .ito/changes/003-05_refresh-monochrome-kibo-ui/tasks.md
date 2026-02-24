# Tasks for: 003-05_refresh-monochrome-kibo-ui

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for all status transitions.
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`
- **Scope guardrail**: UI/UX-only; no backend/API behavior changes.

```bash
ito tasks status 003-05_refresh-monochrome-kibo-ui
ito tasks next 003-05_refresh-monochrome-kibo-ui
ito tasks start 003-05_refresh-monochrome-kibo-ui 1.1
ito tasks complete 003-05_refresh-monochrome-kibo-ui 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add Kibo-compatible frontend foundation

- **Files**: `web/package.json`, `web/pnpm-lock.yaml`, `web/src/ui/styles.css`, `web/src/main.tsx`
- **Dependencies**: None
- **Action**: Add and configure the Kibo/shadcn-compatible frontend foundation required to consume Kibo components in the current Vite app.
- **Verify**: `pnpm -C web install && pnpm -C web lint`
- **Done When**: The web app installs and type-checks with the new UI foundation in place.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 1.2: Implement monochrome dark tokens and shared layout primitives

- **Files**: `web/src/ui/styles.css`, `web/src/ui/components/Shell.tsx`, `web/src/ui/components/Topbar.tsx`, `web/src/ui/components/StatusPill.tsx`
- **Dependencies**: Task 1.1
- **Action**: Define the new monochrome token system, single-accent state styling, and shared shell/topbar/status primitives used by every route.
- **Verify**: `pnpm -C web lint && pnpm -C web build`
- **Done When**: Shared navigation, topbar, and status visuals are consistent and compile cleanly across routes.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Migrate sessions and runs list workflows to Kibo-based UI

- **Files**: `web/src/ui/pages/SessionsPage.tsx`, `web/src/ui/pages/RunsPage.tsx`
- **Dependencies**: None
- **Action**: Replace list-page form/table/action surfaces with Kibo-based components and refreshed hierarchy while preserving existing API interactions.
- **Verify**: `pnpm -C web test -- --runInBand`
- **Done When**: Session creation/listing and harness-run listing/filtering work unchanged with refreshed visuals.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 2.2: Migrate session and run detail workflows to Kibo-based UI

- **Files**: `web/src/ui/pages/SessionDetailPage.tsx`, `web/src/ui/pages/RunDetailPage.tsx`
- **Dependencies**: Task 2.1
- **Action**: Refresh detail-page cards, forms, action rows, and metadata presentation using the shared design system and Kibo composition.
- **Verify**: `pnpm -C web test -- --runInBand`
- **Done When**: Start/stop/resume/attach-entry workflows remain functional with updated visual and interaction consistency.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Refresh attach page with parity and role-safe affordances

- **Files**: `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/components/Topbar.tsx`, `web/src/ui/styles.css`
- **Dependencies**: None
- **Action**: Bring attach UI into the same monochrome/Kibo design language while preserving viewer read-only and driver control/send behavior.
- **Verify**: `pnpm -C web test -- --runInBand`
- **Done When**: Attach page styling matches the rest of the console and role gating behavior remains intact.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 3.2: Apply technical copy pass across refreshed routes

- **Files**: `web/src/ui/pages/SessionsPage.tsx`, `web/src/ui/pages/SessionDetailPage.tsx`, `web/src/ui/pages/RunsPage.tsx`, `web/src/ui/pages/RunDetailPage.tsx`, `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/components/Shell.tsx`
- **Dependencies**: Task 3.1
- **Action**: Tighten labels, helper text, and surface copy to concise technical language aligned with the product voice.
- **Verify**: `pnpm -C web test -- --runInBand`
- **Done When**: Copy remains unambiguous, action-oriented, and consistent with existing workflow semantics.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 3.3: Update UI tests and run final verification gates

- **Files**: `web/src/ui/workflow.test.tsx`, `web/src/ui/test/*` (if needed)
- **Dependencies**: Task 3.2
- **Action**: Expand and update UI tests to cover refreshed route rendering, key workflow actions, and attach role behavior; run lint/build/test gates.
- **Verify**: `pnpm -C web lint && pnpm -C web test && pnpm -C web build`
- **Done When**: All web verification commands pass with test coverage maintained at project target expectations.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

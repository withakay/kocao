# Design: 009-02 Remote Agent Dashboard Information Architecture

## Goal

Define an implementation-ready dashboard structure for Wave 3 that is centered on named remote agents, durable task dispatch records, transcripts, and artifacts, while still exposing pool membership as an operator-facing grouping and filter. The UI must not treat raw harness-run ids as the primary navigation model.

## Existing UI Patterns

- The web UI uses shell navigation plus a page-level `Topbar`.
- Data-heavy pages follow a list/detail shape using `Card`, `Table`, and `CollapsibleSection` primitives.
- Existing detail pages drill into a durable resource id and then expose related records as sections rather than separate disconnected dashboards.

## Dashboard Entry

- Sidebar nav label: `Remote Agents`
- Base route: `/remote-agents`
- Landing route: `/remote-agents/tasks`
- Layout: top-level overview cards above a split view with a primary resource list and a selected detail pane.

## Primary Resources

The dashboard is organized around four first-class resources. Pools remain visible throughout the IA, but Wave 3 keeps them subordinate to agents and tasks because pool membership is a grouping/filtering affordance rather than a standalone operator workflow yet.

### 1. Agents

- Purpose: show named workers and pool membership.
- Primary columns: agent, pool, availability, current task, last activity, workspace session.
- Detail sections: summary, assignment, session, timeline, metadata.
- Primary drill-down: agent -> current task.
- Pool treatment: pool name stays on the list/detail surface so operators can filter or compare assignments without leaving the agent/task workflow.

### 2. Tasks

- Purpose: operational queue for in-flight and recently terminal work.
- Primary columns: task id, state, agent, pool, attempt, last transition.
- Active states: `assigned`, `running`
- Terminal states: `completed`, `failed`, `timed_out`, `cancelled`
- Explicitly excluded state: `queued`
- Detail sections: summary, assignment, session, timeline, transcript, artifacts, metadata.
- Primary drill-downs: task -> transcript, task -> artifacts, task -> assigned agent.
- Pool treatment: task records retain `pool` as assignment metadata, but there is no dedicated `/remote-agents/pools/*` route until pool-level actions exist.

### 3. Transcripts

- Purpose: persisted task conversation and tool/event history.
- Scope: transcript view is always nested under a task route.
- Primary columns: sequence, timestamp, role, kind, event ref.
- Detail sections: summary, transcript, metadata.
- Primary drill-downs: transcript -> parent task, transcript -> related artifacts.

### 4. Artifacts

- Purpose: generated files, patches, bundles, and reports persisted with the task result.
- Scope: artifact view is always nested under a task route.
- Primary columns: artifact name, kind, media type, size, created time.
- Detail sections: summary, artifacts, metadata.
- Primary drill-downs: artifact -> parent task, artifact -> transcript context.

## Route Hierarchy

- `/remote-agents/tasks`
- `/remote-agents/tasks/$taskId`
- `/remote-agents/tasks/$taskId/transcript`
- `/remote-agents/tasks/$taskId/artifacts`
- `/remote-agents/agents`
- `/remote-agents/agents/$agentId`

This route structure keeps the operator anchored on durable task and agent ids. Harness-run and session bindings remain supporting metadata inside detail sections.

Pools are intentionally omitted from the first-class route hierarchy for Wave 3. The dashboard must show pool identity anywhere assignment context matters, but a dedicated pool route would add a navigation surface without a distinct operator action model yet.

## Overview Cards

The landing page should expose a compact summary row before the split view:

- active agents
- active tasks
- terminal tasks in the last 24 hours
- artifacts generated in the last 24 hours

These cards are summary affordances only. Selecting a card pivots into the matching resource list instead of opening a separate dashboard mode.

## Detail Pane Rules

- Agent detail emphasizes current assignment and session binding.
- Task detail is the canonical operator drill-down view.
- Transcript and artifact previews are subordinate to task detail, not peer top-level dashboards.
- When transcript or artifact data is missing, the UI keeps the task detail visible and shows an empty-state panel rather than navigating away.

## Data Contract For Wave 3

Wave 3 can implement directly against the typed UI contract in `web/src/ui/lib/api.ts` and `web/src/ui/lib/remoteAgentDashboard.ts`.

- Resource types mirror the orchestration API from task 1.1.
- `RemoteAgentTask` is the lean list/base contract. Transcript and artifact payloads are modeled as task-detail expansions rather than implied inline fields on every task response.
- Dashboard collections encode table columns, empty states, detail sections, and drill-down targets.
- Helper functions generate routes from durable `agent.id` and `task.id` values.

## Out Of Scope For Task 1.2

- Live API integration
- Rendering the actual pages and sidebar entry
- Polling behavior and mutations

Those belong to Wave 3 once the API and CLI work from Wave 2 are in place.

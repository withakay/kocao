<!-- ITO:START -->
# Tasks for: 008-02_add-showboat-terminal-recordings

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 008-02_add-showboat-terminal-recordings
ito tasks next 008-02_add-showboat-terminal-recordings
ito tasks start 008-02_add-showboat-terminal-recordings 0.1
ito tasks complete 008-02_add-showboat-terminal-recordings 0.1
```

______________________________________________________________________

## Wave 0

- **Depends On**: None

### Task 0.1: Record a terminal demo PoC and capture findings

- **Files**: `.local/docs/agents/*` or `docs/agents/*`, candidate `demos/*.cast`
- **Dependencies**: None
- **Action**: Record one representative terminal workflow with Asciinema before implementation approval, then capture the findings: command shape, cast size, readability, whether a sibling `.cast` file works well in-repo, and any issues that should change the implementation plan.
- **Verify**: `asciinema convert -f txt <cast> -`
- **Done When**: The proposal has a concrete PoC result that either validates the approach or surfaces changes needed before implementation.
- **Requirements**: showboat-demo-recording:scripted-capture-helper, showboat-demo-recording:checked-in-recording-artifact
- **Updated At**: 2026-04-15
- **Status**: [x] complete

______________________________________________________________________

## Wave 1

- **Depends On**: Wave 0

### Task 1.1: Add a repo-local Asciinema recording helper

- **Files**: `scripts/*` or `demos/*`
- **Dependencies**: None
- **Action**: Add a small non-interactive helper that wraps `asciinema record --command` with repo conventions for output path, overwrite behavior, idle-time limit, and failure propagation so agents can generate `.cast` files repeatably.
- **Verify**: Record a short command and confirm `asciinema convert -f txt <cast> -` succeeds.
- **Done When**: Demo authors can generate a `.cast` file from a scripted command without manually driving an interactive recording session.
- **Requirements**: showboat-demo-recording:scripted-capture-helper
- **Updated At**: 2026-04-15
- **Status**: [x] complete

### Task 1.2: Document the dual-artifact demo workflow

- **Files**: `docs/demos/*`, `demos/*`, or related skill guidance
- **Dependencies**: Task 1.1
- **Action**: Document how a Showboat demo and its companion `.cast` file are created, where the recording lives, and why recording verification stays separate from `showboat verify`.
- **Verify**: Markdown doc renders cleanly and references the helper command accurately.
- **Done When**: Future demo authors have one repo-local reference for creating markdown-plus-recording demo artifacts.
- **Requirements**: showboat-demo-recording:authoring-guidance
- **Updated At**: 2026-04-15
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Update existing demos to reference companion recordings

- **Files**: `demos/*.md`
- **Dependencies**: None
- **Action**: Update the selected Showboat demos to include a standard recording section or link pointing at a sibling `.cast` artifact, with wording that explains what the terminal playback adds beyond the captured output blocks.
- **Verify**: Markdown preview shows valid relative links and the demo text remains readable without the recording.
- **Done When**: The target demos clearly advertise their terminal recording companion without depending on a non-portable renderer.
- **Requirements**: showboat-demo-recording:portable-demo-reference
- **Updated At**: 2026-04-15
- **Status**: [x] complete

### Task 2.2: Generate and commit the recording artifacts

- **Files**: `demos/*.cast`
- **Dependencies**: Task 2.1
- **Action**: Record the target terminal workflows with the helper, store the resulting `.cast` files next to their demo markdown, and keep the artifacts small enough to review in git.
- **Verify**: `asciinema convert -f txt <cast> -` succeeds for each committed recording.
- **Done When**: The repository contains checked-in `.cast` files that correspond to the updated demo documents.
- **Requirements**: showboat-demo-recording:checked-in-recording-artifact
- **Updated At**: 2026-04-15
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Verify markdown demos and recording artifacts together

- **Files**: `demos/*`, helper/docs files from Waves 1-2
- **Dependencies**: None
- **Action**: Run the normal Showboat verification for the updated markdown demos and separately validate the recordings via Asciinema conversion/parsing so both artifact types are covered by an explicit workflow.
- **Verify**: `showboat verify <demo>` and `asciinema convert -f txt <cast> -`
- **Done When**: The markdown walkthroughs still replay cleanly and the committed recordings are parseable by Asciinema.
- **Requirements**: showboat-demo-recording:verification-boundary
- **Updated At**: 2026-04-15
- **Status**: [x] complete
<!-- ITO:END -->

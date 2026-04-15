<!-- ITO:START -->
## Why

Showboat gives Kocao readable, executable markdown demos, but those demos only preserve final command output. Reviewers still miss the live terminal behavior that matters during product demos and operator walkthroughs: pacing, screen clearing, progressive logs, and other TTY-visible transitions. Adding checked-in Asciinema casts beside Showboat demos makes terminal playback a first-class artifact without weakening Showboat's deterministic verification model.

## What Changes

- Add a short research/PoC step before implementation approval that records one representative terminal workflow and confirms the artifact size, readability, and in-repo ergonomics are acceptable.
- Add a repo-local recording helper that captures scripted terminal runs into checked-in `.cast` artifacts using `asciinema record --command`.
- Use direct Asciinema PTY capture as the default path; treat `tmux` as an optional follow-on transport for more interactive or storyboard-driven recordings rather than a required v1 dependency.
- Define a standard convention for locating recording assets next to their demo markdown so demos remain portable in the repository.
- Update selected Showboat demos to link to their companion terminal recordings and explain what the recording proves.
- Keep `showboat verify` focused on replaying markdown code blocks; validate recordings separately through non-interactive Asciinema conversion/parsing rather than timing-sensitive playback.
- Document the authoring workflow so future demos can be created with both Showboat markdown and Asciinema recordings from the same repo-local process.

## Capabilities

### New Capabilities

- `showboat-demo-recording`: repo-local workflow for producing and checking in Asciinema terminal recordings alongside Showboat demo documents.

### Modified Capabilities

- `agent-management-skill`: its demo artifact can now include a checked-in terminal recording companion in addition to the markdown walkthrough.

## Impact

- Affected code: repo-local demo tooling under `scripts/` or `demos/`, selected `demos/*.md` documents, and any related skill/docs references.
- Dependencies: requires `asciinema` to be installed for recording generation and cast validation.
- Approval gate: implementation should be reviewed against a small proof-of-concept recording before the broader workflow is adopted.
- Artifact policy: `.cast` files become committed demo assets rather than transient local files.
- Verification boundary: Showboat markdown replay remains deterministic; cast verification uses Asciinema file conversion/parsing instead of interactive playback.

## Research

- PoC artifact: `demos/showboat-terminal-recording-poc.cast`
- Text export used for review: `docs/agents/showboat-terminal-recording-poc.txt`
- Recording command shape tested successfully: `asciinema record --headless --overwrite --return --idle-time-limit 1.25 --window-size 100x28 --title ... --command "<scripted shell workflow>"`
- Result: the candidate cast is about `1.1K`, parseable via `asciinema convert -f txt`, and small enough to review comfortably in git for short workflows.
- Result: a sibling `.cast` file in `demos/` works fine with repository-relative references; no external Asciinema hosting is required for basic review.
- Caveat: headless scripted recordings do not show typed commands automatically, so the eventual helper should print shell-style prompt lines such as `$ ...` before each recorded command to keep playback readable.
- External reference review: the `terminal-recording` skill example uses `tmux + asciinema + agg`, which is a good fit for heavily scripted TUI/GIF pipelines. For Kocao's current need, the right adaptation is narrower: keep `asciinema` as the required recording layer, do not require `zellij`, and defer `tmux`/`agg` to a later enhancement unless we need synchronized multi-step TUI driving or GIF export.
<!-- ITO:END -->

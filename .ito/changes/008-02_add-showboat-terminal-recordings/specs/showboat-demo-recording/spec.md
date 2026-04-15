<!-- ITO:START -->
## ADDED Requirements

### Requirement: Checked-In Recording Artifact
Showboat demos SHALL be able to include a checked-in Asciinema recording artifact stored in the repository next to the demo markdown.

- **Requirement ID**: showboat-demo-recording:checked-in-recording-artifact

#### Scenario: Demo references a sibling cast file
- **WHEN** a demo author publishes a Showboat demo with terminal playback
- **THEN** the demo markdown references a relative `.cast` file that is committed alongside the demo document

#### Scenario: Recording artifact is reviewable offline
- **WHEN** a reviewer clones the repository without external hosting
- **THEN** they can access the recording artifact directly from the working tree without needing an uploaded Asciinema URL

### Requirement: Scripted Capture Helper
The repository SHALL provide a scripted capture workflow for demo authors that records terminal output with Asciinema without requiring manual interactive recording steps.

- **Requirement ID**: showboat-demo-recording:scripted-capture-helper

#### Scenario: Helper records a command into a cast
- **WHEN** a demo author invokes the repo-local recording helper with a command and output path
- **THEN** the helper runs `asciinema record --command` with the repository's default recording options and writes a `.cast` file to the requested location

#### Scenario: Helper preserves command failure
- **WHEN** the recorded command exits non-zero
- **THEN** the helper exits non-zero as well so agents and CI can detect the failed recording run

### Requirement: Portable Demo Reference
Demo markdown SHALL reference companion recordings in a portable way that still reads correctly in plain repository markdown viewers.

- **Requirement ID**: showboat-demo-recording:portable-demo-reference

#### Scenario: Markdown remains useful without an embedded player
- **WHEN** a demo markdown file is viewed on a renderer that does not support custom Asciinema playback components
- **THEN** the recording is still discoverable through a standard relative reference in the document

### Requirement: Authoring Guidance
The repository SHALL document how to create and maintain paired Showboat markdown demos and Asciinema recording artifacts.

- **Requirement ID**: showboat-demo-recording:authoring-guidance

#### Scenario: New demo author follows repo guidance
- **WHEN** an engineer or agent needs to create a new terminal-focused demo
- **THEN** the repository guidance explains how to record the cast, where to store it, and how to reference it from the markdown demo

### Requirement: Verification Boundary
Showboat markdown verification and Asciinema recording verification SHALL remain separate workflows.

- **Requirement ID**: showboat-demo-recording:verification-boundary

#### Scenario: Showboat verification ignores cast playback timing
- **WHEN** `showboat verify` is run for a demo that has a companion `.cast` file
- **THEN** the markdown code blocks are replayed and compared without treating the recording playback timeline as part of Showboat's deterministic verification contract

#### Scenario: Recording verification uses Asciinema parsing
- **WHEN** the repository verifies a committed recording artifact
- **THEN** the workflow validates the `.cast` file through Asciinema conversion or parsing rather than requiring interactive playback

<!-- ITO:END -->

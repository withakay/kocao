<!-- ITO:START -->
## ADDED Requirements

### Requirement: Persistent Agent Transcripts

The system SHALL persist task-level transcripts and summaries for remote-agent work.

- **Requirement ID**: agent-artifacts:persistent-agent-transcripts

#### Scenario: Completed task retains transcript

- **WHEN** a remote agent completes a task
- **THEN** the transcript and summary remain retrievable after the live session ends

### Requirement: Attached Task Artifacts

The system SHALL persist references to files, patches, or generated outputs produced by remote agents.

- **Requirement ID**: agent-artifacts:attached-task-artifacts

#### Scenario: Agent produces a patch or file bundle

- **WHEN** a remote agent emits a patch, report, or generated file set
- **THEN** the orchestration layer records an artifact reference that can be viewed or downloaded later
<!-- ITO:END -->

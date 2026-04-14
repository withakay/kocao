<!-- ITO:START -->
## ADDED Requirements

### Requirement: Startup Performance Metrics

The system SHALL record startup timing metrics for harness runs so image-profile decisions can be evaluated.

- **Requirement ID**: run-execution:startup-performance-metrics

#### Scenario: Cold start is measured

- **WHEN** a harness run pulls an image and reaches readiness
- **THEN** the system records image pull duration, time-to-ready, and time-to-first-prompt for that run
<!-- ITO:END -->

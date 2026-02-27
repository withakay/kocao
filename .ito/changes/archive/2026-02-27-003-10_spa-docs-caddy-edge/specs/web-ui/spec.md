## ADDED Requirements

### Requirement: Embedded Markdown Docs Portal
The system SHALL publish project Markdown documentation as static HTML pages under `/docs` from the same web edge image that serves the SPA.

#### Scenario: User opens docs portal from UI
- **WHEN** a user opens `/docs`
- **THEN** a docs index page lists available documentation pages and links to API docs endpoints

### Requirement: Topbar Links to Docs and API Reference
The workflow UI topbar SHALL expose direct links to `/docs` and `/api/v1/scalar`.

#### Scenario: User opens docs links from workflow route
- **WHEN** a user is on a workflow page and clicks docs or API reference links
- **THEN** the corresponding docs page opens without requiring manual URL entry

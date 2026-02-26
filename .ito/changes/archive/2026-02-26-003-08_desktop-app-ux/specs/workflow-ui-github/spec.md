<!-- ITO:START -->
## MODIFIED Requirements

### Requirement: Detail pages use collapsible sections

The session detail and run detail pages SHALL render each logical group of information (connection info, start-run form, runs list, GitHub outcome, logs) inside individually collapsible sections with smooth expand/collapse animation.

#### Scenario: User collapses a section on a detail page

- **WHEN** a user clicks the section header chevron on a detail page
- **THEN** the section body collapses with smooth animation and the chevron rotates to indicate collapsed state

### Requirement: UI uses desktop-app-density spacing

The UI SHALL use tighter spacing, smaller gaps, and compact padding to achieve desktop-application density rather than web-page spacing.

#### Scenario: All content fits in a 1080p viewport

- **WHEN** a user views any page on a 1920x1080 display
- **THEN** the content fits within the viewport without page-level scrollbars
<!-- ITO:END -->

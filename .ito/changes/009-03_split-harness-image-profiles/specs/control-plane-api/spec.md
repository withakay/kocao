<!-- ITO:START -->
## ADDED Requirements

### Requirement: Image Profile Selection Surface

The control-plane API SHALL expose an explicit surface for selecting or observing the chosen harness image profile.

- **Requirement ID**: control-plane-api:image-profile-selection-surface

#### Scenario: Run request specifies a profile

- **WHEN** a client requests a specific image profile
- **THEN** the API accepts the profile selection and the resulting run status exposes which profile was used

#### Scenario: Run request omits a profile

- **WHEN** a client does not specify a profile
- **THEN** the API applies the documented default selection behavior and reports the selected profile in run status
<!-- ITO:END -->

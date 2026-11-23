# VISP Punch List

This document tracks deferred work, outstanding considerations, and improvements that are intentionally excluded from the current implementation scope.

## Punch Item 1 - Frame Submission Validation and Sender Trust

**Status:** Deferred

**Description:** Revisit validation and trust of incoming frame submissions.

**Considerations:**
- Determine whether the Server should verify that submitted camera IDs belong to registered cameras.
- Define how the Server establishes that frame submissions originate from a trusted Camera Acquisition instance.
- Evaluate authentication and authorization requirements for distributed deployments.

**Current Decision:**

Camera Acquisition retrieves registered camera configurations from the Server and submits frames using the camera IDs provided by the Server.

The frame submission handler does not perform an additional database existence check. Additional validation and sender authentication are deferred until platform security requirements are addressed.

**Revisit:** During security hardening and distributed deployment planning.

### Punch item 2 — Systems Completion-Aligned Verification and Documentation

**Status:** Deferred

**Description:**

Establish a structured verification, testing, and completion documentation framework for VISP, drawing on systems completion and commissioning methodologies used in industrial projects.

The framework should define how software testing, physical installation checks, integration testing, subsystem testing, and system completion are documented and managed throughout the project lifecycle.

**Future considerations:**
- Define documentation requirements for physical installation and mechanical checks, including Inspection and Test Records (ITRs).
- Establish conventions for recording unit and integration test results.
- Define subsystem and system test procedures, acceptance criteria, and test records.
- Establish traceability between requirements, tests, completion criteria, and recorded results.
- Define how deficiencies, outstanding work, and completion status are tracked.
- Establish commissioning and operational handover documentation.

**Current decision:**
- This framework is outside the scope of the initial VISP release.
- Existing Go automated tests and project documentation remain sufficient for current development.
- Live integration and subsystem testing can be documented directly within the relevant implementation milestones.
- Revisit this item when planning the platform's deployment, commissioning, and operational handover requirements.

**What we do now:**

Document significant integration and functional verification activities within the relevant milestone. Do not introduce a formal completion documentation framework for the initial release.

This is deferred work, not a blocker for the initial release.

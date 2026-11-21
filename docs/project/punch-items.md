# VISP Punch List

This document tracks deferred work, outstanding considerations, and improvements that are intentionally excluded from the current implementation scope.

## Frame Submission Validation and Sender Trust

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

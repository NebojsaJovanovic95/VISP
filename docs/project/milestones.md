# VISP Implementation Milestones

This document tracks the major implementation milestones for VISP. Each milestone represents a meaningful stage of development with a defined outcome.

The milestones follow the initial software architecture and focus on delivering a functional initial release. Implementation details may evolve as development progresses.

## Milestone 1 – Project Foundation

**Status:** Complete  
**Completed:** November 17, 2026

**Objective:** Establish the development foundation for the Go-based VISP platform.

**Deliverables:**
- [x] Initialize the Go module using the VISP repository module path.
- [x] Establish the initial repository structure.
- [x] Create independent application entry points for VISP Server and Camera Acquisition.
- [x] Establish internal package boundaries for Go components.
- [x] Establish automated testing conventions using Go's standard testing package.
- [x] Create a Makefile for common development commands.
- [x] Establish build output conventions and Git ignore rules.
- [x] Verify that both Go applications build and run independently.

**Outcome:** A clean, runnable Go project foundation ready for application development.

## Milestone 2 – Database Foundation

**Status:** Complete

**Completed:** November 18, 2026

**Objective:** Establish PostgreSQL as the platform's structured data store.

**Deliverables:**
- [x] Configure PostgreSQL for local development.
- [x] Establish database connection management in the VISP Server.
- [x] Implement Goose database migrations.
- [x] Create the initial configuration and registry tables.
- [x] Create the initial operational and historical data tables.
- [x] Implement primary keys, foreign keys, unique constraints, and validation constraints.
- [x] Verify database connectivity through the VISP Server.
- [x] Test the required database relationships and constraints.

**Outcome:** The database schema is implemented, validated, and accessible through the VISP Server.

## Milestone 3 – VISP Server Foundation

**Objective:** Establish the VISP Server as the central platform coordinator.

Deliverables:
- Establish the HTTP server and REST API structure.
- Implement health and readiness endpoints.
- Implement camera registration and management.
- Implement model and subscription management.
- Implement CameraSubscription management.
- Provide camera configuration to Camera Acquisition.
- Establish the Server's internal application and data-access boundaries.

**Outcome:** The VISP Server can manage platform configuration and provide camera configuration to acquisition components.

## Milestone 4 – Camera Acquisition

**Objective:** Implement independent camera connectivity and image acquisition.

Deliverables:
- Establish the Camera Acquisition Go application.
- Implement RTSP connections using gortsplib.
- Implement video decoding and frame extraction.
- Support per-camera acquisition intervals.
- Retrieve camera configuration from the VISP Server.
- Submit acquired frames and capture metadata to the VISP Server.
- Implement heartbeat and individual camera-status reporting.

**Outcome:** VISP can connect to registered network cameras and submit acquired images to the Server.

## Milestone 5 – Image Storage

**Objective:** Implement image storage and retrieval using S3-compatible object storage.

Deliverables:
- Establish the object storage integration.
- Implement image upload and object-reference management.
- Persist image metadata in PostgreSQL.
- Implement image retrieval through the VISP Server.
- Establish the initial image-saving and retention behaviour.

**Outcome:** Acquired images can be stored, referenced, and retrieved through the platform.

## Milestone 6 – Inference Workflow and Messaging

**Objective:** Implement the Server-managed inference scheduling and asynchronous job workflow.

Deliverables:
- Establish RabbitMQ integration.
- Implement CameraSubscription inference scheduling.
- Implement image eligibility checks using `max_image_age`.
- Prevent duplicate processing of an image by the same CameraSubscription.
- Implement InferenceTask creation and lifecycle management.
- Publish inference jobs through RabbitMQ.
- Consume and process inference results.
- Implement initial failure handling and retry behaviour.

**Outcome:** The VISP Server can schedule inference tasks, dispatch jobs, and manage their lifecycle.

## Milestone 7 – Inference Engine

**Objective:** Implement the initial Python-based inference processing component.

Deliverables:
- Establish the independent Inference Engine application.
- Consume inference jobs from RabbitMQ.
- Retrieve input images directly from File Storage.
- Load and execute the initial predefined computer vision model.
- Format inference results and detections.
- Publish results through RabbitMQ.
- Report engine health and processing status.

**Outcome:** Inference jobs can be processed independently and their results returned to the VISP Server.

## Milestone 8 – Results and Historical Retrieval

**Objective:** Complete the persistence and retrieval of inference results and detections.

Deliverables:
- Persist InferenceResult records.
- Persist individual Detection records.
- Implement standard and detailed result retrieval.
- Implement detection queries by camera, time range, object class, and confidence.
- Implement retrieval of associated images.
- Handle images that are no longer physically available.
- Verify historical data relationships and retention behaviour.

**Outcome:** VISP can store, query, and retrieve historical inference results and associated detections.

## Milestone 9 – Integration and Initial Deployment

**Objective:** Validate the complete platform and prepare the initial deployment.

Deliverables:
- Integrate all platform components.
- Implement the Compose-compatible deployment configuration.
- Verify communication between independent components.
- Test the complete camera-to-inference-to-result workflow.
- Test component restart and failure scenarios.
- Verify database integrity and image storage behaviour.
- Document installation, configuration, and operation.

**Outcome:** A functional initial VISP deployment capable of acquiring images from network cameras, processing them through a computer vision model, and retrieving historical results.

---

## Milestone Tracking

Milestones should be marked complete only when their deliverables have been implemented and verified.

Additional milestones may be introduced as the project evolves. Changes to the implementation sequence or scope should be reflected here.

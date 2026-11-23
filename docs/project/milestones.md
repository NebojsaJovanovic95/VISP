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

### Milestone 3 — VISP Server Foundation

**Status:** Complete

**Completed:** November 18, 2026

**Objectives:**
Establish the VISP Server as the central application component responsible for configuration, database connectivity, HTTP API access, and application lifecycle management.

**Deliverables:**
- [x] Implement centralized application configuration.
- [x] Initialize PostgreSQL connection pool through the database package.
- [x] Establish HTTP server startup and lifecycle management.
- [x] Implement HTTP request routing.
- [x] Implement health check endpoint (`GET /health`).
- [x] Implement graceful shutdown on interrupt and termination signals.
- [x] Support configurable HTTP server address.
- [x] Implement HTTP request logging middleware.
- [x] Log request method, path, response status, and duration.
- [x] Add unit tests for health endpoint and request logging middleware.
- [x] Verify application formatting, tests, static analysis, and builds.

**Completion Criteria:**
- VISP Server starts successfully using application configuration.
- PostgreSQL connectivity is established during startup.
- Health endpoint responds successfully.
- HTTP requests are logged consistently.
- Server shuts down gracefully on termination signals.
- Automated tests pass and application builds successfully.

**Outcome:** The VISP Server provides the initial application foundation, with centralized configuration, PostgreSQL connectivity, HTTP API routing, health monitoring, request logging, and graceful application shutdown.

## Milestone 4 – Camera Acquisition

**Status:** In Progress

**Objective:** Implement independent camera connectivity and image acquisition.

### Milestone 4A – Camera Acquisition

**Status:** Complete

**Deliverables:**
- [x] Establish the Camera Acquisition Go application.
- [x] Implement RTSP connections using gortsplib.
- [x] Implement video decoding and frame extraction.
- [x] Support per-camera acquisition intervals.
- [x] Retrieve camera configuration from the VISP Server.

### Milestone 4B – Frame Submission

**Status:** Complete

**Deliverables:**
- [x] Implement the Server frame-submission endpoint (`POST /api/frames`).
- [x] Validate incoming frame metadata and image data.
- [x] Implement the Acquisition HTTP client for frame submission.
- [x] Submit acquired frames and capture metadata to the VISP Server.
- [x] Implement asynchronous frame submission using a worker and channel.
- [x] Add automated tests for frame submission and error handling.
- [x] Verify acquisition frame handoff and run the Go race detector.

**Outcome:** Live frame acquisition and submission were successfully verified using a physical network camera.

### Milestone 4C – Component Heartbeat

**Status:** Not Started

**Deliverables:**
- [ ] Implement periodic heartbeat reporting from Camera Acquisition.
- [ ] Receive and process heartbeat information in the VISP Server.
- [ ] Track the operational status of the Camera Acquisition component.

### Milestone 4D – Individual Camera Health

**Status:** Not Started

**Deliverables:**
- [ ] Implement individual camera connection and health reporting.
- [ ] Report camera connection failures and recovery.
- [ ] Track individual camera operational status in the VISP Server.

**Completion Criteria:**
- Camera Acquisition retrieves its configuration from the VISP Server.
- Active cameras can be connected to and frames can be acquired.
- Acquired frames and capture metadata can be submitted to the VISP Server.
- Component heartbeat reporting is operational.
- Individual camera health is reported to the VISP Server.
- Automated tests pass and the component builds successfully.

**Outcome:** VISP can connect to registered network cameras, acquire images, submit them to the Server, and report the operational health of the acquisition component and individual cameras.

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

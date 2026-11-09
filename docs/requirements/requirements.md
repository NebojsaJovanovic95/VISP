# VISP — System Requirements

## 1. Purpose

This document defines the functional and non-functional requirements for the Vision Industrial Systems Platform (VISP).

It translates the project objectives established in the Project Charter into specific, implementable and testable requirements.

Requirements are assigned unique identifiers to support traceability throughout design, implementation and testing.

## 2. Initial Release Scope

The initial release will demonstrate an end-to-end industrial computer vision workflow, from camera registration through video acquisition, object detection, data storage and retrieval.

The initial workflow consists of:

1. Manually register an RTSP camera.
2. Connect to and acquire frames from its video stream.
3. Run object detection using a predefined model.
4. Store inference results and associated images.
5. Retrieve and view historical detection records.

The initial release will focus on backend functionality and API interfaces. A web-based management dashboard is deferred to a later release.

## 3. Functional Requirements

### 3.1 Camera Management

**FR-001 — Camera Registration**

VISP shall allow an administrator to manually register a network-connected camera by providing:

- Camera name
- RTSP connection URL
- Optional description
- Enabled/disabled status

**FR-002 — Camera Management**

VISP shall allow an administrator to view, edit and remove registered cameras.

**FR-003 — Camera Connectivity Monitoring**

VISP shall continuously monitor the connectivity of registered and enabled cameras and maintain their operational status.

**FR-004 — Automatic Camera Reconnection**

VISP shall automatically attempt to re-establish the RTSP connection when a registered camera becomes disconnected or its video stream becomes unavailable.

### 3.2 Video Acquisition

**FR-005 — Configurable Frame Acquisition**

VISP shall acquire frames from connected camera streams at a configurable interval.

**FR-006 — Camera Acquisition Configuration**

VISP shall allow the acquisition interval to be configured independently for each camera.

### 3.3 Inference Processing

**FR-007 — Object Detection**

VISP shall process acquired camera frames using a predefined object detection model.

**FR-008 — Inference Result Handling**

VISP shall receive and process inference results returned by the inference engine.

**FR-009 — Frame Inference**

VISP shall submit every successfully acquired frame to the inference engine for object detection.

Model selection and management are deferred to a later release.

### 3.4 Data Management

**FR-010 — Detection Result Storage**

VISP shall store detection results, including:

- Detected object class
- Confidence score
- Bounding-box coordinates
- Detection timestamp
- Associated camera

**FR-011 — Detection Image Storage**

VISP shall store the acquired frame associated with each inference result containing at least one detection.

Configurable recording policies are deferred to a later release.

### 3.5 Data Retrieval and Access

**FR-012 — Historical Detection Queries**

VISP shall allow users to retrieve historical detection records using:

- Camera
- Date/time range
- Detected object class

**FR-013 — Detection Details**

VISP shall allow users to view detection details, including the associated image, timestamp, camera and detected objects.

**FR-014 — Platform API**

VISP shall expose an API that allows external clients to:

- Register, configure and manage cameras.
- Retrieve camera connectivity and operational status.
- Retrieve historical detection records and associated images.

The initial release will be operated through API requests and command-line tools. A web-based dashboard is deferred.

## 4. Non-Functional Requirements

### 4.1 Architecture and Scalability

**NFR-001 — Separation of Server and Inference Components**

VISP shall implement the server and inference engine as separate components with defined communication interfaces.

For the initial release, both components may run on the same Linux machine, but they shall be independently executable.

The architecture shall allow the components to be deployed on separate machines in the future.

**NFR-002 — Asynchronous Inference Communication**

VISP shall support asynchronous communication between server and inference components through a messaging mechanism.

The messaging mechanism shall support:

- Submission of inference jobs.
- Retrieval of inference results.
- Decoupling of server and inference execution.
- Future support for multiple inference engines.

The specific messaging technology is not prescribed by this requirement.

### 4.2 Reliability and Failure Handling

**NFR-003 — Inference Job Failure Handling**

VISP shall implement automatic retry handling for inference jobs that fail or do not return a result.

- Jobs shall be retried a limited number of times.
- Jobs that exceed the retry limit shall be marked as failed.
- Failed jobs shall be recorded so their status can be inspected.

The retry limit and retry interval may be configurable.

**NFR-004 — Duplicate Job Handling**

VISP shall assign a unique identifier to each inference job and prevent duplicate processing results from being stored when the same job is received or processed more than once.

### 4.3 Security

**NFR-005 — Secure Credential Storage**

VISP shall encrypt camera connection credentials at rest.

- RTSP usernames and passwords shall not be stored in plain text.
- Encryption keys shall be supplied through environment configuration, not stored alongside encrypted credentials.
- Credentials shall not be exposed in API responses, application logs or error messages.
- Access to decrypted credentials shall be restricted to components that require them to establish camera connections.

**NFR-006 — API Authentication**

VISP shall require authentication for access to protected API endpoints.

- Unauthenticated requests shall be rejected.
- The initial release shall support at least one administrator account.
- Authentication credentials shall be stored securely.
- Authenticated users shall be authorized according to their assigned permissions.

**NFR-007 — API Transport Security**

VISP shall support HTTPS for API communication to protect data and authentication credentials in transit.

- Production deployments shall use HTTPS.
- Local development environments may use HTTP.
- The architecture shall support certificate configuration.

### 4.4 Performance and Resource Management

**NFR-008 — Inference Response Time**

VISP shall target an inference response time of no more than 2 seconds under normal operating conditions, measured from frame submission to receipt of inference results.

**NFR-009 — Resource Management**

VISP shall monitor the resource utilization of its server and inference components, including:

- CPU utilization
- Memory utilization
- GPU utilization and memory, where available

The platform should expose sufficient information to identify resource constraints and processing bottlenecks.

### 4.5 Operations and Maintainability

**NFR-010 — Logging and Diagnostics**

VISP shall maintain structured application logs to support troubleshooting and operational monitoring.

Logs shall:

- Include timestamps and severity levels.
- Identify the component generating the log.
- Record relevant errors and processing failures.
- Avoid exposing sensitive information, including credentials.

**NFR-011 — Linux Compatibility**

VISP shall support deployment and execution on Linux systems.

The initial release shall target Linux environments with the necessary hardware and software dependencies for video acquisition and inference.

**NFR-012 — Automated Testing**

VISP shall include automated tests to verify the correctness of its core functionality.

Testing shall include:

- Unit tests for individual components.
- Integration tests for communication between server, inference and data services.
- Tests for camera connectivity and reconnection.
- Tests for inference job processing and failure handling.
- Tests for data storage and retrieval.

**NFR-013 — Maintainability**

VISP shall use clearly defined component interfaces and modular software architecture to support independent development, testing and maintenance of its components.

## 5. Future Requirements

The following capabilities are identified as future development objectives but are not part of the initial release:

- Automated camera discovery and bulk camera registration.
- Continuous video capture.
- Model selection and model management.
- Configurable recording policies.
- Web-based management dashboard.
- Multiple independently scalable inference engines.
- Deployment of server and inference components across multiple machines.
- Application development, deployment and execution capabilities.
- Advanced analytics and data aggregation.

## 6. Acceptance Criteria

The initial release shall be considered functionally demonstrated when the following workflow can be completed:

1. An administrator registers an RTSP camera.
2. VISP establishes a connection to the camera.
3. VISP continuously monitors camera connectivity and can automatically reconnect after disconnection.
4. VISP acquires frames according to the configured interval.
5. Acquired frames are submitted to the predefined object detection model.
6. Inference results are returned to the server.
7. Detection results and associated images are stored.
8. Users can retrieve historical detections by camera, date/time range and detected object class.
9. Users can inspect detection details and associated images.
10. The server and inference engine operate as separate components communicating through an asynchronous messaging mechanism.

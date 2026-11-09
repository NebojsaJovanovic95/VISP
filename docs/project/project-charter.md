# VISP - Project Charter

## Project Boundaries
VISP is an industrial computer vision platform focused on providing infrastructure for video acquisition, inference, data management and analytics.

The following are explicitly outside the scope of VISP:

1. **Industrial process control:** VISP is not a general-purpose industrial control system and does not directly control industrial equipment or processes.

2. **Model training:** VISP provides model management, deployment and inference capabilities. Model training is not a core platform responsibility.

3. **Operational control:** VISP provides information, insights and analytics to support operators and engineers. Operational decisions and equipment control remain the responsibility of existing industrial systems and personnel.

## Intended Users
These are intended user categories, not necessarily separate permission levels.
- Platform Administrators: Configure and maintain the platform
- System Integrators: Connect cameras, configure models and integrate external systems
- Operators: Monitor camera feeds, detections and analytics
- Process engineers: Use collected data to investigate and understand industrial processes

## Project Motivation and Purpose
VISP originated as a technical exploration of edge computing and industrial computer vision, with the objective of utilizing GPU-accelerated computing on existing Linux hardware.

The project was initially inspired by previous experience assisting with a YOLOv8 computer vision implementation at Onaping Depth Mine. This provided an opportunity to further explore video acquisition, object detection, client-server architecture and distributed computing.

During development, several potential industrial applications were identified, including PPE compliance monitoring, restricted-area access monitoring and wildlife detection.

Rather than developing these applications as independent solutions, the project evolved towards building a reusable industrial computer vision platform.

**The primary purpose of VISP is to provide a common infrastructure that simplifies the development, deployment, management and operation of industrial computer vision applications.**

By providing reusable platform capabilities, VISP aims to reduce the effort required to develop new applications and avoid duplicating infrastructure across individual solutions.

### Acknowledgements

VISP is inspired in part by the industrial computer vision work carried out at Onaping Depth Mine.

I would like to give special recognition to **Paul Bouillon**, who designed and built the original vision system at Onaping from the ground up. As my mentor, he introduced me to the practical applications of computer vision in an industrial environment and helped shape my interest in this field.

In many ways, Paul is the grandfather of VISP. This project is an opportunity to build upon the inspiration and knowledge that came from his work, while exploring new possibilities for industrial vision.

## Project Scope

VISP aims to provide a reusable industrial computer vision platform supporting the acquisition, processing, management and analysis of visual data from industrial environments.

The platform will provide the infrastructure and configuration capabilities required to deploy and operate multiple computer vision applications without requiring each application to independently implement camera acquisition, inference processing, data storage and other common functionality.

### Core Platform Capabilities

The following capabilities define the intended scope of VISP.

1. **Camera Management**
   - Register, configure and manage network-connected video sources.
   - Support RTSP streams as the initial camera connection interface.
   - Maintain camera information, connection details and operational status.
   - Support multiple subscriptions per camera.
   - Provide manual camera registration initially, with the architecture allowing for future bulk registration and automated camera discovery.

2. **Subscription Management**
   - Create and manage reusable camera processing configurations.
   - Configure the inference model, objects of interest, detection frequency and recording policies.
   - Associate subscriptions with the applications they support.

3. **Camera-Subscription Management**
   - Associate registered cameras with subscriptions.
   - Allow a camera to have multiple subscriptions.
   - Allow a subscription to be assigned to multiple cameras.

4. **Inference Management**
   - Manage available inference models and their deployment.
   - Distribute inference requests to available processing instances.
   - Support multiple models and independently scalable inference engines.

5. **Request Processing and Service Coordination**
   - Provide server interfaces for receiving and processing platform requests.
   - Coordinate communication between cameras, inference engines and data services.
   - Support multiple interchangeable server instances to accommodate increasing workloads.

6. **Data Management**
   - Collect, store and retrieve images, video and inference results.
   - Maintain relationships between cameras, subscriptions and collected data.
   - Support configurable recording and data retention policies.

7. **Analytics and Data Access**
   - Provide interfaces for querying and retrieving historical detection data.
   - Support access to collected computer vision data by external applications and reporting systems.
   - Provide optional in-platform data processing and aggregation capabilities to reduce data transfer and support efficient data retrieval.
   - Support analysis of collected computer vision data.

8. **Application Development, Deployment and Execution**
   - Support deployable application packages containing application-specific logic, standardized VISP interfaces and configuration files.
   - Allow applications to declare their required cameras, subscriptions, models, objects of interest and processing policies.
   - Validate application configurations and establish the required platform resources and camera subscriptions.
   - Provide an execution environment for application logic within VISP server instances.
   - Support event-driven execution of application logic in response to inference results and other platform events, allowing system integrators to implement real-time business logic.
   - Provide standardized interfaces for applications to query and retrieve platform data, with the option to execute supported data processing and aggregation operations within VISP.
   - Make application-generated results accessible through standardized interfaces.

9. **Web-Based Management Dashboard**
   - Provide a user interface for managing cameras, subscriptions, models and platform configuration.
   - Allow users to monitor camera feeds, detections and historical events.
   - Provide visibility into platform health and resource utilization.

10. **Platform Monitoring**
    - Monitor the operational status of platform services and processing components.
    - Provide visibility into workload, resource utilization and processing performance.

### Example Applications

VISP is intended to support the development and deployment of industrial computer vision applications, including:

- **PPE Compliance Monitoring:** Detect personnel and required personal protective equipment to support compliance monitoring.
- **Restricted-Area Monitoring:** Detect personnel or objects entering designated restricted areas.
- **Wildlife Detection:** Identify and monitor wildlife activity in industrial environments.

These applications serve as examples of the types of solutions that can be developed using VISP. Their application-specific rules and interpretation logic are not necessarily part of the core platform.

### Scalability

VISP should be designed to support horizontal scaling of server instances and inference processing independently.

The initial implementation may operate on a single Linux machine. However, the architecture should allow additional server and inference instances to be deployed across multiple machines as processing demands increase.

Core data services, including database and file storage, are intended to remain logically centralized, although their eventual scaling requirements will be evaluated as the platform develops.

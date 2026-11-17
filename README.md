# VISP
**Vision Industrial Systems Platform**

## Overview

VISP is a platform for developing and deploying industrial computer vision solutions on edge infrastructure.

The platform provides reusable infrastructure for video acquisition, model inference, data management, and analytics. Its purpose is to simplify the development of industrial vision applications without requiring each application to implement its own underlying infrastructure.

VISP is designed primarily for on-premises Linux environments, with the ability to distribute platform components across multiple machines as deployment requirements evolve.

## Vision and Objectives

Industrial environments can benefit from computer vision in applications ranging from safety compliance to process monitoring.

VISP aims to provide a common platform through which these applications can be developed, deployed, and managed.

### Example Applications

- **PPE Compliance:** Detecting whether personnel are wearing required personal protective equipment.
- **Restricted-Area Monitoring:** Detecting personnel or other objects entering designated areas.
- **Wildlife Monitoring:** Detecting wildlife around industrial facilities.
- **Process Monitoring:** Extracting visual information such as material size, fluid levels, or flotation conditions.

These applications demonstrate potential uses of the platform and provide opportunities to expand its capabilities.

## Architecture

VISP separates video acquisition, platform coordination, inference processing, messaging, structured data storage, and file storage into distinct components.

This separation allows components to operate independently and scale according to their respective workloads.

Refer to [Software Architecture](docs/architecture/architecture.md) for the detailed architecture and design decisions.

## Technology Stack

The initial implementation uses:

- **Go:** Core platform components.
- **Python:** Initial inference engine.
- **PostgreSQL:** Structured data storage.
- **RabbitMQ:** Asynchronous messaging.
- **S3-compatible object storage:** Image and file storage.
- **RTSP:** Network camera video acquisition.
- **Linux:** Deployment environment.
- **Docker / Podman:** Containerized deployment.

## Project Status

VISP is under active development. The current focus is establishing the Go project foundation and implementing the platform incrementally according to the documented architecture and milestones.

## Documentation

- [Project Charter](docs/project/project-charter.md)
- [Requirements](docs/requirements/requirements.md)
- [Software Architecture](docs/architecture/architecture.md)
- [Implementation Milestones](docs/project/milestones.md)

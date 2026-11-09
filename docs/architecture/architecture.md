## 1. Logical Architecture

- **Camera Acquisition** – Maintains camera connections and acquires video frames.
- **VISP Server** – Coordinates platform operations, manages requests, and provides access to platform services.
- **Inference Engine** – Executes computer vision models and returns inference results.
- **Messaging System** – Facilitates asynchronous communication between the VISP Server and inference engines.
- **Database** – Stores platform configuration, camera information, subscriptions, and inference metadata.
- **File Storage** – Stores images and other files associated with inference results.

## 2. Component Interaction

### 2.1. Real-time Inference

1. Camera Acquisition periodically retrieves camera and subscription configuration from the VISP Server.
2. Camera Acquisition acquires a frame from a registered camera.
3. The frame is submitted to the VISP Server along with its camera identifier.
4. The VISP Server determines which subscriptions and inference models apply.
5. The VISP Server creates and submits the appropriate inference jobs to the Messaging System.
6. An available Inference Engine consumes the job and executes the required model.
7. The inference result is returned through the Messaging System.
8. The VISP Server processes the result and coordinates its storage and availability to applications.

### 2.2. VISP Query

1. A request is submitted to the VISP Server.
2. The VISP Server processes the request and retrieves the required information from the Database and/or File Storage.
3. The VISP Server returns the requested information to the requester.

## 3. Data Architecture

## 4. Deployment Architecture

# Acquisition–Server API

This document defines the HTTP interface between the VISP Camera Acquisition component and the VISP Server.

The API uses synchronous HTTP requests. Both components maintain their own internal data structures and communicate through the documented request and response formats.

## Base URL

The Server address is configured through the Camera Acquisition configuration.

## Endpoints

### GET /api/cameras

Retrieves registered camera configurations from the VISP Server.

**Request**

```http
GET /api/cameras
```

**Success Response**

Status: `200 OK`

Content-Type: `application/json`

Returns an array of camera configurations.

```json
[
  {
    "camera_id": "camera-uuid",
    "camera_name": "Camera 1",
    "rtsp_url": "rtsp://example.com/stream",
    "location": "Production Area",
    "status": "active",
    "acquisition_interval_seconds": 5
  }
]
```

An empty array is returned when no cameras are registered.

### POST /api/frames

Submits an acquired camera frame to the VISP Server.

**Request**

Content-Type: `multipart/form-data`

The request contains two parts:

| Field | Content-Type | Description |
|---|---|---|
| `metadata` | `application/json` | Frame metadata |
| `image` | `image/png` | PNG-encoded camera frame |

**Metadata**

```json
{
  "camera_id": "camera-uuid",
  "captured_at": "2026-11-21T12:00:00Z"
}
```

| Field | Type | Description |
|---|---|---|
| `camera_id` | string | ID of the camera that produced the frame |
| `captured_at` | timestamp | Time the frame was acquired |

The timestamp uses RFC 3339 format.

**Success Response**

Status: `201 Created`

Content-Type: `application/json`

```json
{
  "image_id": "image-uuid"
}
```

The Server generates and returns a unique image ID.

**Validation**

- Both multipart fields are required.
- Metadata must contain a valid camera ID and timestamp.
- The image must be a valid PNG.
- Maximum request size: 50 MiB.

**Error Responses**

The Server returns an appropriate HTTP error status when the request is invalid or cannot be processed.

## Notes

- Camera Acquisition retrieves camera configurations from the Server.
- Camera Acquisition submits acquired frames using the camera IDs provided by the Server.
- The Server is responsible for assigning image IDs.
- Each component maintains its own internal data structures. Shared Go packages are not required for API communication.
- Frame storage currently uses local disk. The storage implementation may change independently of this API contract.

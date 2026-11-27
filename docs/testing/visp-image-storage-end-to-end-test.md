# VISP End-to-End Image Storage and Retrieval Test

**Purpose:** Verify that the running VISP system acquires frames from a registered RTSP camera, stores image files in SeaweedFS, records image metadata in PostgreSQL, and retrieves an image through the VISP Server API.

This is an end-to-end integration check for the completed Milestone 5 work. It should be completed before starting Milestone 6 (Inference Workflow and Messaging), and repeated when changes affect camera acquisition, the Server, database persistence, object storage, or image retrieval.

## 1. Expected data flow

1. Camera Acquisition retrieves enabled camera configuration from the VISP Server.
2. Camera Acquisition connects to the camera's RTSP stream and captures frames.
3. Camera Acquisition submits frames to the VISP Server.
4. The Server stores image files in SeaweedFS using the S3-compatible storage interface.
5. The Server stores image metadata and the object reference in PostgreSQL.
6. The Server retrieves an image from SeaweedFS when an image is requested through its HTTP API.

PostgreSQL stores image metadata and object references; the image bytes are stored in SeaweedFS.

## 2. Prerequisites

Have the following available:

- PostgreSQL running, with the VISP database and schema migrated.
- SeaweedFS running and its S3-compatible endpoint available.
- The VISP Server configured to use the PostgreSQL database and SeaweedFS.
- Camera Acquisition configured to communicate with the VISP Server.
- A reachable RTSP camera and its URL configured in `.env`.

Do not paste or commit database passwords, S3 credentials, or RTSP credentials into test notes or source control.

## 3. Start and verify the dependencies

### PostgreSQL

Check that PostgreSQL is accepting connections:

```bash
pg_isready
```

Verify that the VISP database is accessible:

```bash
psql -h localhost -p 5432 -U visp -d visp -c '\dt'
```

The VISP tables, including `cameras` and `images`, should be present.

### SeaweedFS

Start SeaweedFS using the project's configured development settings. For the current local setup, the command used was:

```bash
source .env

AWS_ACCESS_KEY_ID="$S3_ACCESS_KEY" \
AWS_SECRET_ACCESS_KEY="$S3_SECRET_KEY" \
S3_BUCKET="$S3_BUCKET" \
~/.local/bin/weed mini -dir="$HOME/.local/share/seaweedfs"
```

Leave SeaweedFS running in its own terminal.

The local SeaweedFS Filer interface is available at:

<http://localhost:8888/>

The S3-compatible endpoint used by VISP is configured in `.env` (currently port `8333`).

## 4. Register a camera

After recreating the database or when no camera is registered, insert a camera record. The following example uses the existing `RTSP_URL` from `.env`.

Run this in Bash from the VISP project root. The extraction removes surrounding double quotes if the URL value in `.env` is quoted:

```bash
RTSP_URL=$(sed -n 's/^RTSP_URL=//p' .env | tr -d '"')

psql -h localhost -U visp -d visp -v rtsp_url="$RTSP_URL" <<'SQL'
INSERT INTO cameras (
    camera_name,
    rtsp_url,
    location,
    status,
    acquisition_interval
)
VALUES (
    'Oko Svetog Nikole',
    :'rtsp_url',
    'Home',
    'active',
    INTERVAL '5 seconds'
)
RETURNING camera_id, camera_name, location, status, acquisition_interval;
SQL
```

The returned row should contain a new `camera_id`. Do not print or share the RTSP URL because it may contain camera credentials.

If a camera already exists, inspect it rather than inserting a duplicate:

```sql
SELECT camera_id, camera_name, status, acquisition_interval
FROM cameras;
```

If the stored RTSP URL was accidentally inserted with literal surrounding double quotes, correct that value before starting acquisition:

```sql
UPDATE cameras
SET rtsp_url = trim(both '"' from rtsp_url)
WHERE camera_name = 'Oko Svetog Nikole';
```

## 5. Start the VISP Server

Start the Server using the project's normal run target:

```bash
make run-server
```

Keep it running in its own terminal. Confirm that it starts successfully and is listening on the configured address (the local test used port `8080`).

The Server should be able to connect to both PostgreSQL and SeaweedFS using the settings in `.env`.

## 6. Start Camera Acquisition

In another terminal, run:

```bash
make run-acquisition
```

Expected startup output includes:

- Camera Acquisition starts.
- It fetches camera configurations from the VISP Server.
- It reports the registered camera (for example, `Oko Svetog Nikole`).
- The camera connects successfully.
- Frames are captured and submitted.

If it reports `Retrieved 0 camera(s)`, check the `cameras` table and ensure a camera is registered and enabled/active according to the current acquisition implementation.

If RTSP URL parsing fails, inspect the stored URL for accidental literal quote characters.

Let acquisition run for approximately 15–20 seconds to submit several frames, then stop it with `Ctrl+C`.

## 7. Verify image files in SeaweedFS

Open the SeaweedFS Filer interface:

<http://localhost:8888/>

Navigate to the configured bucket (currently `visp`) and the `frames/` directory.

Confirm that image objects have been created. The expected object-key format is:

```text
frames/<image_id>.png
```

The image files should be visible in SeaweedFS. This confirms that the Server is storing the acquired image bytes in object storage rather than only on the local filesystem.

## 8. Verify image metadata in PostgreSQL

Run:

```bash
psql -h localhost -U visp -d visp
```

Then query the most recent image records:

```sql
SELECT
    image_id,
    camera_id,
    image_reference,
    captured_at,
    file_available
FROM images
ORDER BY captured_at DESC
LIMIT 10;
```

Verify that:

- Image records are being created.
- `camera_id` matches the registered camera.
- `image_reference` points to the corresponding `frames/<image_id>.png` object.
- `file_available` is `true` for the newly stored images.
- `captured_at` values reflect the recent acquisition run.

The `image_id` in a row should match the filename portion of its `image_reference`.

## 9. Retrieve an image through the VISP Server

Choose a recent `image_id` from the PostgreSQL query. Use the Server API route, including the `/api` prefix:

```text
GET /api/images/{image_id}
```

For the local development Server on port `8080`, run:

```bash
curl -v \
  http://localhost:8080/api/images/<image_id> \
  -o /tmp/visp-test-image.png
```

Replace `<image_id>` with the actual UUID from PostgreSQL (do not include angle brackets).

Check the HTTP response. A successful retrieval should return `200 OK`.

Verify that the downloaded file is a valid image:

```bash
file /tmp/visp-test-image.png
```

Open it:

```bash
xdg-open /tmp/visp-test-image.png
```

Confirm that the image displays correctly. This verifies retrieval through the VISP Server, which reads the object from SeaweedFS and returns the image to the client.

## 10. Pass criteria

The end-to-end test passes when all of the following are true:

- [ ] PostgreSQL is running and the VISP database is accessible.
- [ ] SeaweedFS is running and accessible.
- [ ] The VISP Server starts with the configured database and object-storage settings.
- [ ] Camera Acquisition retrieves the registered camera configuration.
- [ ] The camera connects and frames are submitted to the Server.
- [ ] New image objects appear in the SeaweedFS `visp/frames/` location.
- [ ] Corresponding rows appear in PostgreSQL `images`.
- [ ] Each new row references the expected `frames/<image_id>.png` object and has `file_available = true`.
- [ ] A selected image can be retrieved through `GET /api/images/{image_id}`.
- [ ] The retrieved file is recognized as an image and opens successfully.

## 11. Troubleshooting notes from the first end-to-end run

| Symptom | Check |
|---|---|
| `pg_isready` reports no response | Start PostgreSQL and check its service status. |
| Camera Acquisition reports `Retrieved 0 camera(s)` | Check that the camera row exists in PostgreSQL and is eligible to be returned by the Server's camera-configuration endpoint. |
| RTSP URL parsing fails with a message mentioning a quote character | Check whether the database value contains literal surrounding `"` characters. |
| Frames are submitted but no objects appear in SeaweedFS | Check the Server's storage configuration and logs, and confirm the configured bucket and endpoint. |
| Objects exist but no rows appear in `images` | Check the Server's image metadata persistence and database logs. |
| `GET /images/{id}` returns `404` | The route is incorrect for this API. The registered route is `GET /api/images/{image_id}`. |
| Image retrieval returns `410 Gone` | Check `file_available` in PostgreSQL and whether the object still exists in SeaweedFS. |
| Retrieval succeeds but the file does not open | Check the HTTP response and the downloaded file with `file`; ensure the output is not an error response saved with a `.png` extension. |

## 12. Result of the November 27, 2026 test

The end-to-end test was completed successfully:

- PostgreSQL was running and the recreated camera record was registered.
- SeaweedFS was running.
- Camera Acquisition retrieved the camera configuration and connected to the RTSP camera.
- Frames were submitted to the VISP Server.
- Image files were visible in SeaweedFS under the `visp` bucket's `frames/` directory.
- Corresponding image metadata records were present in PostgreSQL.
- The records referenced `frames/<image_id>.png` and showed `file_available = true`.
- An image was successfully retrieved through `GET /api/images/{image_id}` and opened locally.

**Result: Passed.** The live acquisition → Server → SeaweedFS/PostgreSQL → Server retrieval path was verified.

---

This test verifies the Milestone 5 image-storage and retrieval path using the actual running services. It does not test inference processing or RabbitMQ; those belong to Milestone 6.

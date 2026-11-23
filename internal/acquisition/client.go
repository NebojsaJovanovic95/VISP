package acquisition

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type CameraConfig struct {
	CameraID                   string  `json:"camera_id"`
	CameraName                 string  `json:"camera_name"`
	RTSPURL                    string  `json:"rtsp_url"`
	Location                   *string `json:"location"`
	Status                     string  `json:"status"`
	AcquisitionIntervalSeconds float64 `json:"acquisition_interval_seconds"`
}

type cameraStatusRequest struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type frameMetadata struct {
	CameraID   string    `json:"camera_id"`
	CapturedAt time.Time `json:"captured_at"`
}

type frameSubmission struct {
	Metadata frameMetadata
	Frame    *image.RGBA
}

type frameResponse struct {
	ImageId string `json:"image_id"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ListCameras retrieves registered camera configurations from the VISP Server.
func (c *Client) ListCameras(ctx context.Context) ([]CameraConfig, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/cameras",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create camera request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request camera configurations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"server returned unexpected status: %s",
			resp.Status,
		)
	}

	var cameras []CameraConfig
	if err := json.NewDecoder(resp.Body).Decode(&cameras); err != nil {
		return nil, fmt.Errorf("decode camera configurations: %w", err)
	}

	return cameras, nil
}

// SubmitFrame sends an acquired camera frame to the VISP Server.
func (c *Client) SubmitFrame(ctx context.Context, metadata frameMetadata, frame *image.RGBA) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	metadataPart, err := writer.CreateFormField("metadata")
	if err != nil {
		return "", fmt.Errorf("create metadata field: %w", err)
	}

	if err := json.NewEncoder(metadataPart).Encode(metadata); err != nil {
		return "", fmt.Errorf("encode frame metadata: %w", err)
	}

	imagePart, err := writer.CreateFormFile("image", "frame.png")
	if err != nil {
		return "", fmt.Errorf("create image field: %w", err)
	}

	if err := png.Encode(imagePart, frame); err != nil {
		return "", fmt.Errorf("encode frame image: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/frames", &body)
	if err != nil {
		return "", fmt.Errorf("create frame request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("submit frame: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("server returned unexpected status: %s", resp.Status)
	}

	var result frameResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode frame response: %w", err)
	}

	if result.ImageId == "" {
		return "", fmt.Errorf("server returned an empty image ID")
	}

	return result.ImageId, nil
}

// SendHeartbeat reports that the Camera Acquisition component is running.
func (c *Client) SendHeartbeat(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/heartbeat",
		nil,
	)
	if err != nil {
		return fmt.Errorf("create heartbeat request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned unexpected heartbeat status: %s", resp.Status)
	}

	return nil
}

// SendCameraStatus reports an individual camera's connection status.
func (c *Client) SendCameraStatus(ctx context.Context, cameraID string, status string, statusErr error) error {
	requestBody := cameraStatusRequest{
		Status: status,
	}

	if statusErr != nil {
		requestBody.Error = statusErr.Error()
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode camera status: %w", err)
	}

	url := fmt.Sprintf("%s/api/cameras/%s/status", c.baseURL, cameraID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create camera status request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send camera status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned unexpected camera status response: %s", resp.Status)
	}

	return nil
}

package acquisition

import (
	"context"
	"encoding/json"
	"fmt"
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

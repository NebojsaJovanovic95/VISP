package server

import (
	"encoding/json"
	"net/http"
)

// cameraResponse represents a camera configuration returned by the API.
type cameraResponse struct {
	CameraID                   string  `json:"camera_id"`
	CameraName                 string  `json:"camera_name"`
	RTSPURL                    string  `json:"rtsp_url"`
	Location                   *string `json:"location"`
	Status                     string  `json:"status"`
	AcquisitionIntervalSeconds float64 `json:"acquisition_interval_seconds"`
}

// listCamerasHandler returns all registered camera configurations.
func (s *Server) listCamerasHandler(w http.ResponseWriter, r *http.Request) {
	cameras, err := s.db.ListCameras(r.Context())
	if err != nil {
		http.Error(w, "failed to retrieve cameras", http.StatusInternalServerError)
		return
	}

	response := make([]cameraResponse, 0, len(cameras))

	for _, camera := range cameras {
		response = append(response, cameraResponse{
			CameraID:                   camera.CameraID,
			CameraName:                 camera.CameraName,
			RTSPURL:                    camera.RTSPURL,
			Location:                   camera.Location,
			Status:                     camera.Status,
			AcquisitionIntervalSeconds: camera.AcquisitionIntervalSeconds,
		})
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

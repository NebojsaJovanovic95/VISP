package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxFrameSize = 50 << 20 // 50 MiB

// frameMetadata describes the metadata accompanying an acquired frame.
type frameMetadata struct {
	CameraID   string    `json:"camera_id"`
	CapturedAt time.Time `json:"captured_at"`
}

// receiveFrameHandler receives an acquired image and its capture metadata.
func (s *Server) receiveFrameHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFrameSize)

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "frame exceeds maximum allowed size", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "invalid multipart request", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	var metadata frameMetadata

	if err := json.Unmarshal([]byte(r.FormValue("metadata")), &metadata); err != nil {
		http.Error(w, "invalid frame metadata", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(metadata.CameraID) == "" || metadata.CapturedAt.IsZero() {
		http.Error(w, "camera_id and captured_at are required", http.StatusBadRequest)
		return
	}

	imageFile, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer imageFile.Close()

	// Verify that the uploaded file is a PNG image.
	header := make([]byte, 512)
	n, err := imageFile.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}

	if http.DetectContentType(header[:n]) != "image/png" {
		http.Error(w, "image must be a PNG", http.StatusBadRequest)
		return
	}

	if _, err := imageFile.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "failed to read image", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(s.frameDir, 0755); err != nil {
		http.Error(w, "failed to prepare image storage", http.StatusInternalServerError)
		return
	}

	dst, err := os.CreateTemp(s.frameDir, "frame-*.png")
	if err != nil {
		http.Error(w, "failed to create image file", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(dst.Name())
	imageID := strings.TrimSuffix(strings.TrimPrefix(filename, "frame-"), ".png")

	if _, err := io.Copy(dst, imageFile); err != nil {
		dst.Close()
		os.Remove(dst.Name())
		http.Error(w, "failed to save image", http.StatusInternalServerError)
		return
	}

	if err := dst.Close(); err != nil {
		os.Remove(dst.Name())
		http.Error(w, "failed to save image", http.StatusInternalServerError)
		return
	}

	log.Printf(
		"Received frame: camera_id=%s captured_at=%s image=%s",
		metadata.CameraID,
		metadata.CapturedAt.Format(time.RFC3339Nano),
		filename,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"image_id": imageID,
	}); err != nil {
		log.Printf("Encode frame response: %v", fmt.Errorf("%w", err))
	}
}

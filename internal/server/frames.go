package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	imageFile, imageHeader, err := r.FormFile("image")
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

	imageID := uuid.NewString()
	objectKey := "frames/" + imageID + ".png"

	imageReference, err := s.storage.Upload(
		r.Context(),
		objectKey,
		imageFile,
		imageHeader.Size,
		"image/png",
	)
	if err != nil {
		log.Printf("Failed to upload frame %s: %v", imageID, err)
		http.Error(w, "failed to save image", http.StatusInternalServerError)
		return
	}

	if err := s.db.CreateImage(
		r.Context(),
		imageID,
		imageReference,
		metadata.CameraID,
		metadata.CapturedAt,
	); err != nil {
		log.Printf("Failed to create image record %s: %v", imageID, err)
		http.Error(w, "failed to save image metadata", http.StatusInternalServerError)
		return
	}
	log.Printf(
		"Received frame: camera_id=%s captured_at=%s image_id=%s image_reference=%s",
		metadata.CameraID,
		metadata.CapturedAt.Format(time.RFC3339Nano),
		imageID,
		imageReference,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"image_id": imageID,
	}); err != nil {
		log.Printf("Encode frame response: %v", fmt.Errorf("%w", err))
	}
}

// getImageHandler retrieves a stored image by its ID.
func (s *Server) getImageHandler(w http.ResponseWriter, r *http.Request) {
	imageID := r.PathValue("image_id")

	if _, err := uuid.Parse(imageID); err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	image, err := s.db.GetImage(r.Context(), imageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}

		log.Printf("Failed to retrieve image metadata %s: %v", imageID, err)
		http.Error(w, "failed to retrieve image", http.StatusInternalServerError)
		return
	}

	if !image.FileAvailable {
		http.Error(w, "image file is no longer availabel", http.StatusGone)
		return
	}

	body, err := s.storage.Download(r.Context(), image.ImageReference)
	if err != nil {
		log.Printf("Failed to download image %s: %v", imageID, err)
		http.Error(w, "failed to retrieve image", http.StatusInternalServerError)
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	written, err := io.Copy(w, body)
	if err != nil {
		log.Printf(
			"Failed to stream image %s after writing %d bytes: %v",
			imageID,
			written,
			err,
		)

		if written == 0 {
			http.Error(w, "failed to retrieve image", http.StatusInternalServerError)
		}
		return
	}
}

package server

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RunImageRetention identifies expired images and removes their stored files.
func (s *Server) RunImageRetention(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-s.imageRetentionPeriod)

	images, err := s.db.ListImagesEligibleForRetention(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list images eligible for retention: %w", err)
	}

	for _, image := range images {
		if err := s.storage.Delete(ctx, image.ImageReference); err != nil {
			log.Printf(
				"failed to delete expired image %q: %v",
				image.ImageID,
				err,
			)
			continue
		}

		if err := s.db.MarkImageUnavailable(ctx, image.ImageID); err != nil {
			log.Printf(
				"deleted image file %q but failed to update database: %v",
				image.ImageID,
				err,
			)
		}
	}

	return nil
}

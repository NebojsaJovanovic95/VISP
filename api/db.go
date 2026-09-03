package api

import "context"

type Frame struct {
	ID        int64
	ImagePath string
	CameraID  string
	ModelName string
	CreatedAt int64
}

type Detection struct {
	ID         int64
	FrameID    int64
	ClassName  string
	Confidence float64
	BBox       []float64
	Attributes map[string]any
	CreatedAt  int64
}

type FrameWithDetections struct {
	Frame      Frame
	Detections []Detection
}

type FrameQuery struct {
	CameraID     string
	ModelName    string
	CreatedAfter int64
	Limit        int
	Offset       int
}

type DetectionQuery struct {
	FrameID       *int64
	ClassName     string
	MinConfidence *float64
	Limit         int
	Offset        int
}

type DB interface {
	InsertFrame(ctx context.Context, frame Frame) (int64, error)
	InsertDetection(ctx context.Context, detection Detection) (int64, error)

	GetFrameByID(ctx context.Context, id int64) (*FrameWithDetections, error)
	GetDetectionByID(ctx context.Context, id int64) (*Detection, error)

	GetFrames(ctx context.Context, query FrameQuery) ([]Frame, error)
	GetDetections(ctx context.Context, query DetectionQuery) ([]Detection, error)
}

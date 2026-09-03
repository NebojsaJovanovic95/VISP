package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

const maxRowLimit = 1000

type PostgresDB struct {
	db *sql.DB
}

func NewPostgresDB(db *sql.DB) *PostgresDB {
	return &PostgresDB{
		db: db,
	}
}

// SCHEMA INITIALIZATION

func (p *PostgresDB) InitSchema(ctx context.Context) error {
	const createFrameTable = `
	CREATE TABLE IF NOT EXISTS frame (
		id BIGSERIAL PRIMARY KEY,
		image_path TEXT NOT NULL,
		camera_id TEXT,
		model_name TEXT,
		created_at BIGINT NOT NULL
	);
	`

	const createDetectionTable = `
	CREATE TABLE IF NOT EXISTS detection (
		id BIGSERIAL PRIMARY KEY,
		frame_id BIGINT NOT NULL
			REFERENCES frame(id)
			ON DELETE CASCADE,
		class_name TEXT NOT NULL,
		confidence REAL NOT NULL,
		bbox JSONB NOT NULL,
		attributes JSONB
	);
	`

	indexes := []string{
		`
		CREATE INDEX IF NOT EXISTS idx_frames_ts
		ON frame (created_at DESC);
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_frames_camera
		ON frame (camera_id);
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_detections_class
		ON detection (class_name);
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_detections_frame
		ON detection (frame_id);
		`,
	}

	if _, err := p.db.ExecContext(ctx, createFrameTable); err != nil {
		return fmt.Errorf("create frame table: %w", err)
	}

	if _, err := p.db.ExecContext(ctx, createDetectionTable); err != nil {
		return fmt.Errorf("create detection table: %w", err)
	}

	for _, index := range indexes {
		if _, err := p.db.ExecContext(ctx, index); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// END OF SCHEMA INITIALIZATION

func (p *PostgresDB) InsertFrame(
	ctx context.Context,
	frame Frame,
) (int64, error) {

	const query = `
		INSERT INTO frame (
			image_path,
			camera_id,
			model_name,
			created_at
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id int64

	err := p.db.QueryRowContext(
		ctx,
		query,
		frame.ImagePath,
		frame.CameraID,
		frame.ModelName,
		frame.CreatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("insert frame: %w", err)
	}

	return id, nil
}

func (p *PostgresDB) InsertDetection(
	ctx context.Context,
	detection Detection,
) (int64, error) {

	bboxJSON, err := json.Marshal(detection.BBox)
	if err != nil {
		return 0, fmt.Errorf("marshal bbox: %w", err)
	}

	attributesJSON, err := json.Marshal(
		detection.Attributes,
	)
	if err != nil {
		return 0, fmt.Errorf("marshal attributes: %w", err)
	}

	const query = `
		INSERT INTO detection (
			frame_id,
			class_name,
			confidence,
			bbox,
			attributes,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int64

	err = p.db.QueryRowContext(
		ctx,
		query,
		detection.FrameID,
		detection.ClassName,
		detection.Confidence,
		bboxJSON,
		attributesJSON,
		detection.CreatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("insert detection: %w", err)
	}

	return id, nil
}

func (p *PostgresDB) GetFrameByID(
	ctx context.Context,
	id int64,
) (*FrameWithDetections, error) {

	const frameQuery = `
		SELECT
			id,
			image_path,
			camera_id,
			model_name,
			created_at
		FROM frame
		WHERE id = $1
	`

	var frame Frame

	err := p.db.QueryRowContext(
		ctx,
		frameQuery,
		id,
	).Scan(
		&frame.ID,
		&frame.ImagePath,
		&frame.CameraID,
		&frame.ModelName,
		&frame.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"get frame %d: %w",
			id,
			err,
		)
	}

	const detectionQuery = `
		SELECT
			id,
			frame_id,
			class_name,
			confidence,
			bbox,
			attributes
		FROM detection
		WHERE frame_id = $1
		ORDER BY id ASC
	`

	rows, err := p.db.QueryContext(
		ctx,
		detectionQuery,
		id,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"get detections for frame %d: %w",
			id,
			err,
		)
	}

	defer rows.Close()

	detections := make([]Detection, 0)

	for rows.Next() {
		var detection Detection
		var bboxJSON []byte
		var attributesJSON []byte

		err := rows.Scan(
			&detection.ID,
			&detection.FrameID,
			&detection.ClassName,
			&detection.Confidence,
			&bboxJSON,
			&attributesJSON,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"scan detection: %w",
				err,
			)
		}

		if err := json.Unmarshal(
			bboxJSON,
			&detection.BBox,
		); err != nil {
			return nil, fmt.Errorf(
				"unmarshal bbox: %w",
				err,
			)
		}

		if len(attributesJSON) > 0 {
			if err := json.Unmarshal(
				attributesJSON,
				&detection.Attributes,
			); err != nil {
				return nil, fmt.Errorf(
					"unmarshal attributes: %w",
					err,
				)
			}
		}

		detections = append(detections, detection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate detections: %w",
			err,
		)
	}

	return &FrameWithDetections{
		Frame:      frame,
		Detections: detections,
	}, nil
}

func (p *PostgresDB) GetDetectionByID(
	ctx context.Context,
	id int64,
) (*Detection, error) {

	const query = `
		SELECT
			id,
			frame_id,
			class_name,
			confidence,
			bbox,
			attributes
		FROM detection
		WHERE id = $1
	`

	var detection Detection
	var bboxJSON []byte
	var attributesJSON []byte

	err := p.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&detection.ID,
		&detection.FrameID,
		&detection.ClassName,
		&detection.Confidence,
		&bboxJSON,
		&attributesJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"get detection %d: %w",
			id,
			err,
		)
	}

	if err := json.Unmarshal(
		bboxJSON,
		&detection.BBox,
	); err != nil {
		return nil, fmt.Errorf(
			"unmarshal bbox: %w",
			err,
		)
	}

	if len(attributesJSON) > 0 {
		if err := json.Unmarshal(
			attributesJSON,
			&detection.Attributes,
		); err != nil {
			return nil, fmt.Errorf(
				"unmarshal attributes: %w",
				err,
			)
		}
	}

	return &detection, nil
}

func (p *PostgresDB) GetFrames(
	ctx context.Context,
	q FrameQuery,
) ([]Frame, error) {

	query := `
		SELECT
			id,
			image_path,
			camera_id,
			model_name,
			created_at
		FROM frame
	`

	var conditions []string
	var args []any
	argNumber := 1

	if q.CameraID != "" {
		conditions = append(
			conditions,
			fmt.Sprintf("camera_id = $%d", argNumber),
		)

		args = append(args, q.CameraID)
		argNumber++
	}

	if q.ModelName != "" {
		conditions = append(
			conditions,
			fmt.Sprintf("model_name = $%d", argNumber),
		)

		args = append(args, q.ModelName)
		argNumber++
	}

	if q.CreatedAfter != 0 {
		conditions = append(
			conditions,
			fmt.Sprintf("created_at > $%d", argNumber),
		)

		args = append(args, q.CreatedAfter)
		argNumber++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(
			conditions,
			" AND ",
		)
	}

	query += " ORDER BY id DESC"

	if q.Limit > 0 {
		limit := q.Limit

		if limit > maxRowLimit {
			limit = maxRowLimit
		}

		query += fmt.Sprintf(
			" LIMIT $%d",
			argNumber,
		)

		args = append(args, limit)
		argNumber++
	}

	if q.Offset > 0 {
		query += fmt.Sprintf(
			" OFFSET $%d",
			argNumber,
		)

		args = append(args, q.Offset)
	}

	rows, err := p.db.QueryContext(
		ctx,
		query,
		args...,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"get frames: %w",
			err,
		)
	}

	defer rows.Close()

	frames := make([]Frame, 0)

	for rows.Next() {
		var frame Frame

		err := rows.Scan(
			&frame.ID,
			&frame.ImagePath,
			&frame.CameraID,
			&frame.ModelName,
			&frame.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"scan frame: %w",
				err,
			)
		}

		frames = append(frames, frame)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate frames: %w",
			err,
		)
	}

	return frames, nil
}

func (p *PostgresDB) GetDetections(
	ctx context.Context,
	q DetectionQuery,
) ([]Detection, error) {

	query := `
		SELECT
			id,
			frame_id,
			class_name,
			confidence,
			bbox,
			attributes
		FROM detection
	`

	var conditions []string
	var args []any
	argNumber := 1

	if q.FrameID != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("frame_id = $%d", argNumber),
		)

		args = append(args, *q.FrameID)
		argNumber++
	}

	if q.ClassName != "" {
		conditions = append(
			conditions,
			fmt.Sprintf("class_name = $%d", argNumber),
		)

		args = append(args, q.ClassName)
		argNumber++
	}

	if q.MinConfidence != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("confidence > $%d", argNumber),
		)

		args = append(args, *q.MinConfidence)
		argNumber++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(
			conditions,
			" AND ",
		)
	}

	query += " ORDER BY id DESC"

	if q.Limit > 0 {
		limit := q.Limit

		if limit > maxRowLimit {
			limit = maxRowLimit
		}

		query += fmt.Sprintf(
			" LIMIT $%d",
			argNumber,
		)

		args = append(args, limit)
		argNumber++
	}

	if q.Offset > 0 {
		query += fmt.Sprintf(
			" OFFSET $%d",
			argNumber,
		)

		args = append(args, q.Offset)
	}

	rows, err := p.db.QueryContext(
		ctx,
		query,
		args...,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"get detections: %w",
			err,
		)
	}

	defer rows.Close()

	detections := make([]Detection, 0)

	for rows.Next() {
		var detection Detection
		var bboxJSON []byte
		var attributesJSON []byte

		err := rows.Scan(
			&detection.ID,
			&detection.FrameID,
			&detection.ClassName,
			&detection.Confidence,
			&bboxJSON,
			&attributesJSON,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"scan detection: %w",
				err,
			)
		}

		if err := json.Unmarshal(
			bboxJSON,
			&detection.BBox,
		); err != nil {
			return nil, fmt.Errorf(
				"unmarshal bbox: %w",
				err,
			)
		}

		if len(attributesJSON) > 0 {
			if err := json.Unmarshal(
				attributesJSON,
				&detection.Attributes,
			); err != nil {
				return nil, fmt.Errorf(
					"unmarshal attributes: %w",
					err,
				)
			}
		}

		detections = append(
			detections,
			detection,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate detections: %w",
			err,
		)
	}

	return detections, nil
}

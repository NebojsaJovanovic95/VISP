package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLiteDB(db *sql.DB) *SQLiteDB {
	return &SQLiteDB{
		db: db,
	}
}

func (s *SQLiteDB) InitSchema(ctx context.Context) error {
	const frameTable = `
	CREATE TABLE IF NOT EXISTS frame (
		id INTEGER PRIMARY KEY AUTOINCREMENT,

		image_path TEXT NOT NULL,
		camera_id TEXT,
		model_name TEXT,

		created_at INTEGER NOT NULL,

		synced INTEGER NOT NULL DEFAULT 0,
		postgres_id INTEGER
	);
	`

	const detectionTable = `
	CREATE TABLE IF NOT EXISTS detection (
		id INTEGER PRIMARY KEY AUTOINCREMENT,

		frame_id INTEGER NOT NULL,

		class_name TEXT NOT NULL,
		confidence REAL NOT NULL,

		bbox TEXT NOT NULL,
		attributes TEXT,

		created_at INTEGER NOT NULL,

		FOREIGN KEY (frame_id)
			REFERENCES frame(id)
			ON DELETE CASCADE
	);
	`

	indexes := []string{
		`
		CREATE INDEX IF NOT EXISTS
		idx_frame_created_at
		ON frame(created_at DESC);
		`,

		`
		CREATE INDEX IF NOT EXISTS
		idx_frame_synced
		ON frame(synced);
		`,

		`
		CREATE INDEX IF NOT EXISTS
		idx_detection_frame
		ON detection(frame_id);
		`,

		`
		CREATE INDEX IF NOT EXISTS
		idx_detection_class
		ON detection(class_name);
		`,
	}

	if _, err := s.db.ExecContext(ctx, frameTable); err != nil {
		return fmt.Errorf("create frame table: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, detectionTable); err != nil {
		return fmt.Errorf("create detection table: %w", err)
	}

	for _, index := range indexes {
		if _, err := s.db.ExecContext(ctx, index); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

func OpenSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open(
		"sqlite",
		path,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"open sqlite: %w",
			err,
		)
	}

	return db, nil
}

func (s *SQLiteDB) InsertFrameWithDetections(
	ctx context.Context,
	frame Frame,
	detections []Detection,
) (int64, error) {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf(
			"begin transaction: %w",
			err,
		)
	}

	defer tx.Rollback()

	const frameQuery = `
		INSERT INTO frame (
			image_path,
			camera_id,
			model_name,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`

	result, err := tx.ExecContext(
		ctx,
		frameQuery,
		frame.ImagePath,
		frame.CameraID,
		frame.ModelName,
		frame.CreatedAt,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"insert frame: %w",
			err,
		)
	}

	frameID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf(
			"get frame id: %w",
			err,
		)
	}

	const detectionQuery = `
		INSERT INTO detection (
			frame_id,
			class_name,
			confidence,
			bbox,
			attributes,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	for _, detection := range detections {
		bbox, err := json.Marshal(
			detection.BBox,
		)

		if err != nil {
			return 0, fmt.Errorf(
				"marshal bbox: %w",
				err,
			)
		}

		attributes, err := json.Marshal(
			detection.Attributes,
		)

		if err != nil {
			return 0, fmt.Errorf(
				"marshal attributes: %w",
				err,
			)
		}

		_, err = tx.ExecContext(
			ctx,
			detectionQuery,
			frameID,
			detection.ClassName,
			detection.Confidence,
			bbox,
			attributes,
			detection.CreatedAt,
		)

		if err != nil {
			return 0, fmt.Errorf(
				"insert detection: %w",
				err,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf(
			"commit transaction: %w",
			err,
		)
	}

	return frameID, nil
}

func (s *SQLiteDB) GetUnsyncedFrames(
	ctx context.Context,
) ([]Frame, error) {

	const query = `
		SELECT
			id,
			image_path,
			camera_id,
			model_name,
			created_at
		FROM frame
		WHERE synced = 0
		ORDER BY id ASC
	`

	rows, err := s.db.QueryContext(
		ctx,
		query,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"query unsynced frames: %w",
			err,
		)
	}

	defer rows.Close()

	var frames []Frame

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
				"scan unsynced frame: %w",
				err,
			)
		}

		frames = append(frames, frame)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return frames, nil
}

func (s *SQLiteDB) GetDetectionsForFrame(
	ctx context.Context,
	frameID int64,
) ([]Detection, error) {

	const query = `
		SELECT
			id,
			frame_id,
			class_name,
			confidence,
			bbox,
			attributes,
			created_at
		FROM detection
		WHERE frame_id = ?
		ORDER BY id ASC
	`

	rows, err := s.db.QueryContext(
		ctx,
		query,
		frameID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"query detections: %w",
			err,
		)
	}

	defer rows.Close()

	var detections []Detection

	for rows.Next() {
		var d Detection
		var bbox []byte
		var attributes []byte

		err := rows.Scan(
			&d.ID,
			&d.FrameID,
			&d.ClassName,
			&d.Confidence,
			&bbox,
			&attributes,
			&d.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(
			bbox,
			&d.BBox,
		); err != nil {
			return nil, fmt.Errorf(
				"decode bbox: %w",
				err,
			)
		}

		if len(attributes) > 0 {
			if err := json.Unmarshal(
				attributes,
				&d.Attributes,
			); err != nil {
				return nil, fmt.Errorf(
					"decode attributes: %w",
					err,
				)
			}
		}

		detections = append(
			detections,
			d,
		)
	}

	return detections, rows.Err()
}

func (s *SQLiteDB) MarkSynced(
	ctx context.Context,
	localID int64,
	postgresID int64,
) error {

	const query = `
		UPDATE frame
		SET
			synced = 1,
			postgres_id = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(
		ctx,
		query,
		postgresID,
		localID,
	)

	if err != nil {
		return fmt.Errorf(
			"mark frame synced: %w",
			err,
		)
	}

	return nil
}

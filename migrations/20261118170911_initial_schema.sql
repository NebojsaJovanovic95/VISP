-- +goose Up

-- Cameras registered with VISP.
CREATE TABLE cameras (
  camera_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  camera_name TEXT NOT NULL,
  rtsp_url TEXT NOT NULL,
  location TEXT,
  status TEXT NOT NULL,
  acquisition_interval INTERVAL NOT NULL CHECK (acquisition_interval > INTERVAL '0 seconds')
);

-- Models managed by VISP.
CREATE TABLE models (
  model_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  model_name TEXT NOT NULL,
  model_reference TEXT NOT NULL,
  model_type TEXT NOT NULL,
  version TEXT NOT NULL,
  status TEXT NOT NULL
);

-- Defines how a model is applied.
CREATE TABLE subscriptions (
  subscription_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subscription_name TEXT NOT NULL,
  model_id UUID NOT NULL REFERENCES models(model_id) ON DELETE RESTRICT,
  inference_schedule JSONB NOT NULL,
  saving_policy JSONB NOT NULL,
  status TEXT NOT NULL,
  max_image_age INTERVAL
);

-- Assigns subscriptions to cameras.
CREATE TABLE camera_subscriptions (
  camera_subscription_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  camera_id UUID NOT NULL REFERENCES cameras(camera_id) ON DELETE RESTRICT,
  subscription_id UUID NOT NULL REFERENCES subscriptions(subscription_id) ON DELETE RESTRICT,
  status TEXT NOT NULL,

  CONSTRAINT uq_camera_subscription UNIQUE (camera_id, subscription_id)
);

-- Metadata for captured images. The physical files live in object storage.
CREATE TABLE images (
  image_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  image_reference TEXT NOT NULL UNIQUE,
  camera_id UUID NOT NULL REFERENCES cameras(camera_id) ON DELETE RESTRICT,
  camera_snapshot JSONB NOT NULL,
  captured_at TIMESTAMPTZ NOT NULL
);

-- Records of inference work submitted for processing.
CREATE TABLE inference_tasks (
  task_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  image_id UUID NOT NULL REFERENCES images(image_id) ON DELETE RESTRICT,
  camera_subscription_id UUID NOT NULL REFERENCES camera_subscriptions(camera_subscription_id) ON DELETE RESTRICT,
  model_id UUID NOT NULL REFERENCES models(model_id) ON DELETE RESTRICT,
  subscription_snapshot JSONB NOT NULL,
  model_snapshot JSONB NOT NULL,
  status TEXT NOT NULL,
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0),

  CONSTRAINT uq_task_image_subscription UNIQUE (image_id, camera_subscription_id)
);

-- Final outcome of an inference task.
CREATE TABLE inference_results (
  result_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id UUID NOT NULL UNIQUE REFERENCES inference_tasks(task_id) ON DELETE RESTRICT,
  status TEXT NOT NULL,
  completed_at TIMESTAMPTZ NOT NULL,
  processing_duration INTERVAL,
  detection_count INTEGER NOT NULL DEFAULT 0 CHECK (detection_count >= 0),
  result_metadata JSONB
);

-- Individual detections produced by an inference result.
CREATE TABLE detections (
  detection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  result_id UUID NOT NULL REFERENCES inference_results(result_id) ON DELETE RESTRICT,
  class_id INTEGER NOT NULL,
  class_name TEXT NOT NULL,
  confidence NUMERIC(6,5) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  x_min NUMERIC(8,7) NOT NULL CHECK (x_min >= 0 AND x_min <= 1),
  y_min NUMERIC(8,7) NOT NULL CHECK (y_min >= 0 AND y_min <= 1),
  width NUMERIC(8,7) NOT NULL CHECK (width > 0 AND width <= 1),
  height NUMERIC(8,7) NOT NULL CHECK (height > 0 AND height <= 1),

  CONSTRAINT chk_detection_bounds CHECK (
    x_min + width <= 1 AND y_min + height <= 1
  )
);

-- Indexes for common relationship and retrieval queries.
CREATE INDEX idx_subscriptions_model_id
ON subscriptions(model_id);

CREATE INDEX idx_camera_subscriptions_subscription_id
ON camera_subscriptions(subscription_id);

CREATE INDEX idx_images_camera_captured_at
ON images(camera_id, captured_at DESC);

CREATE INDEX idx_tasks_camera_subscription
ON inference_tasks(camera_subscription_id);

CREATE INDEX idx_tasks_model_id
ON inference_tasks(model_id);

CREATE INDEX idx_tasks_status_submitted_at
ON inference_tasks(status, submitted_at);

CREATE INDEX idx_detections_result_id
ON detections(result_id);


-- +goose Down

DROP TABLE IF EXISTS detections;
DROP TABLE IF EXISTS inference_results;
DROP TABLE IF EXISTS inference_tasks;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS camera_subscriptions;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS cameras;

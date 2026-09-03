package api

import (
	"fmt"
	"os"
)

func StartupEvent() {
	os.Mkdir("LOG_DIR", os.ModeAppend)
	// process queue
}

func Stream(UploadFile os.File) string {
	// receive images and enqueue them for background processing
	EnqueImage(UploadFile, UploadFile.Name())
	return fmt.Sprintf("Message: %s queued for detection", UploadFile.Name())
}

func HealthCheck() string {
	// returns 200 if api is running
	// have it return as dict
	return "status : healthy"
}

func Detect(UploadFile os.File) string {
	// open file
	// save the image to minio container
	// log event to the logger
	// create a uuid
	// make a payload with the created id, filename, image bites and minio path'
	// push payload to redis on model result queue
	// pop result off the queue by using the result key made from the que name and request id
	// load the result raw
	// get the detection data out of detection field
	// convert the results to json
	// insert camera id, image path, raw detection and model name to db and get the id that was generated
	// log the event to logger
	// return a Json response with message, detection id, image, path, detection as json
	return "json"
}

func GetAllDetections(frameId int, className string, confidence float64, limit int, offset int) string {
	// get detections using db.GetDetections
	// send it frameId, className, confidence, limit, offset
	// return json responce with:
	// count which is length of detection list
	// detections which is the detection list
	return "json"
}

func GetAllFrames(cameraId int, modelName string, afterTS int, limit int, offset int) string {
	// get frames using db.GetFrames
	// send it cameraId, modelName, afterTS, limit, offset
	// return json responce with:
	// count which is length of frames list
	// detections which is the frames list
	return "json"
}

func GetDetection(id int) string {
	// ...
	return "detection json"
}

func GetFrame(id int) string {
	// ...
	return "frame json"
}

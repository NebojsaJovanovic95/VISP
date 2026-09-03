package api

import "os"

func EnqueImage(UploadFile os.File, fileName string) {
	// add an image to redis queue as serialized tuple.
}

func ProcessQueu() {
	// log "starting redis yolo worke"
	// in an infinite loop:
	// pop the item from the queue
	// if its None continue to next
	// log got item from redis queue
	// load image bytes and file name from the serialized data
	// use tempFile objects function NamedTemporaryFile and use that temp file to write the image bytes and give the tmp path the name of the temp file
	// make a request id as a new uuid
	// make a payload using request id, file name and image bytes
	// push it to the redis model request queue
	// await the result
	// make result key by using redis model result queue and request id
	// use the result key to pop the result from the redis queue
	// load the raw results into results
	// delete the result from the queue as we have collected it (we can defer this)
	// get detection data from result["detection"]
	// open tmp path as f
	// save the file and filename to minio storage by using MinioStorage Objects SaveImage function, this will return a minio path
	// insert frame with detections to the db, use DB objects InsertFrameWithDetections
	// we insert camera id, image path, raw detections and model name
	// log that file was processed and uploaded
	// remove the tmp path
}

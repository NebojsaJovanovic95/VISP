.PHONY: build test coverage fmt vet

build:
	mkdir -p bin
	go build -o bin/visp-server ./cmd/visp-server
	go build -o bin/camera-acquisition ./cmd/camera-acquisition

test:
	go test ./...

coverage:
	go test -cover ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

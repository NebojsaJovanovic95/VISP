.PHONY: build test coverage fmt vet run-server

build:
	mkdir -p bin
	go build -o bin/visp-server ./cmd/visp-server
	go build -o bin/camera-acquisition ./cmd/camera-acquisition

run-server:
	go build -o bin/visp-server ./cmd/visp-server
	@set -a; . ./.env; set +a; ./bin/visp-server

test:
	go test ./...

coverage:
	go test -cover ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

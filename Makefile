.PHONY: build test coverage fmt vet run-server

build:
	mkdir -p bin
	go build -o bin/visp-server ./cmd/visp-server
	go build -o bin/camera-acquisition ./cmd/camera-acquisition

run-server:
	go build -o bin/visp-server ./cmd/visp-server
	@set -a; . ./.env; set +a; ./bin/visp-server

run-acquisition:
	go build -o bin/camera-acquisition ./cmd/camera-acquisition
	./bin/camera-acquisition

migrate-up:
	@set -a; . ./.env; set +a; goose -dir migrations postgres "postgres://$${POSTGRES_USER}:$${POSTGRES_PASSWORD}@$${POSTGRES_HOST}:$${POSTGRES_PORT}/$${POSTGRES_DB}?sslmode=$${POSTGRES_SSLMODE}" up

migrate-down:
	@set -a; . ./.env; set +a; goose -dir migrations postgres "postgres://$${POSTGRES_USER}:$${POSTGRES_PASSWORD}@$${POSTGRES_HOST}:$${POSTGRES_PORT}/$${POSTGRES_DB}?sslmode=$${POSTGRES_SSLMODE}" down
test:
	go test ./...

coverage:
	go test -cover ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

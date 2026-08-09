APP_NAME=cloud-efficiency-engine

.PHONY: test build run docker-build docker-run

test:
	go test ./...

build:
	go build -o bin/$(APP_NAME) ./cmd/api

run:
	go run ./cmd/api

docker-build:
	docker build -t $(APP_NAME):local .

docker-run:
	docker run --rm -p 8080:8080 $(APP_NAME):local
.PHONY: build run test

build:
	go build -o bin/image-previewer ./cmd/image-previewer

run:
	APP_PORT=8080 CACHE_SIZE=100 CACHE_DIR=./cache LOG_LEVEL=debug go run ./cmd/image-previewer/main.go --config=./config/config.yaml

test:
	go test ./...

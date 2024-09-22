.PHONY: build run test

build:
	go build -o bin/image-previewer ./cmd/image-previewer

run:
	go run ./cmd/image-previewer/main.go --config=./config/config.yaml

test:
	go test -race -count 100 ./internal/...

integration-test:
	go test ./test/...

lint:
	golangci-lint run --config=.golangci.yml ./...
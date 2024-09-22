run:
	docker compose up

build:
	go build -o bin/image-previewer ./cmd/image-previewer


test:
	go test -race -count 100 ./internal/...

integration-test:
	go test ./test/...

lint:
	golangci-lint run --config=.golangci.yml ./...
BINARY := docker-override

.PHONY: build test tidy

build:
	go build -o $(BINARY) ./cmd/docker-override

test:
	go test ./...

tidy:
	go mod tidy

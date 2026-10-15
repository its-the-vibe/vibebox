BINARY := gsec

.PHONY: build test tidy

build:
	go build -o $(BINARY) ./cmd/gsec

test:
	go test ./...

tidy:
	go mod tidy

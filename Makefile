.PHONY: build test lint clean

build:
	go build -o bin/goquery ./cmd/goquery

test:
	go test -v ./...

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -rf bin/

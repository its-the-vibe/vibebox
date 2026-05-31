BINARY := bin/goquery

.PHONY: build test lint clean

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/goquery

test:
	go test -v ./...

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -rf bin

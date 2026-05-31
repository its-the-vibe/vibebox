BINARY := bin/goquery

.PHONY: build test lint clean

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/goquery
	cp queries.json bin/queries.json

test:
	go test -v ./...

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -rf bin

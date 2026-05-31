BINARY := bin/goquery

.PHONY: build test lint clean

build:
	mkdir -p bin
	mkdir -p bin/queries
	go build -o $(BINARY) ./cmd/goquery
	cp queries.json bin/queries.json
	cp queries/*.sql bin/queries/

test:
	go test -v ./...

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -rf bin

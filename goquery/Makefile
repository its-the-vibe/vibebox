BINARY := goquery

.PHONY: build test lint clean

build:
	go build -o $(BINARY) ./cmd/goquery

test:
	go test -v ./...

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -rf $(BINARY)

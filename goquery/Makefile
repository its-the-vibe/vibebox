BINARY := goquery

.PHONY: build test tidy lint clean

build:
	go build -o $(BINARY) ./cmd/goquery

test:
	go test -v ./...

tidy:
	go mod tidy

lint:
	go fmt ./...
	go vet ./...

clean:
	rm -rf $(BINARY)

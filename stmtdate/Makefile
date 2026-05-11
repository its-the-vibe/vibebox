BINARY := stmtdate

.PHONY: build test tidy

build:
	go build -o $(BINARY) .

test:
	go test ./...

tidy:
	go mod tidy

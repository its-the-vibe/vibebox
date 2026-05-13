BINARY := stmtpng2tsv

.PHONY: build test tidy run

build:
	go build -o $(BINARY) .

test:
	go test ./...

tidy:
	go mod tidy

run:
	go run . $(ARGS)

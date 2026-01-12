.PHONY: all check fmt lint test build tidy

all: check

check: tidy fmt lint test build

tidy:
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run

test:
	go test -v ./...

build:
	go build -v ./...
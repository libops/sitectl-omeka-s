.PHONY: build deps lint test work install

BINARY_NAME=sitectl-omeka-s

deps: work
	go mod tidy

build:
	go build -o $(BINARY_NAME) .

install: build
	mv $(BINARY_NAME) /usr/local/bin

lint:
	go fmt ./...
	golangci-lint run

test: build
	go test -v -race ./...

work:
	./scripts/use-go-work.sh

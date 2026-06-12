.PHONY: build deps lint test work install integration-test

BINARY_NAME=sitectl-omeka-s
CREATE_DEFINITION?=default
CREATE_ARGS?=
SITECTL_CONTEXT?=integration-test

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

integration-test:
	SITECTL_CONTEXT="$(SITECTL_CONTEXT)" CREATE_DEFINITION="$(CREATE_DEFINITION)" CREATE_ARGS="$(CREATE_ARGS)" ./scripts/test-create.sh

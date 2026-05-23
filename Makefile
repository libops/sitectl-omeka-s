.PHONY: build deps test work install

BINARY_NAME=sitectl-omeka-s
INSTALL_DIR ?= $(or $(dir $(shell which $(BINARY_NAME) 2>/dev/null)),/usr/local/bin/)

deps: work
	go mod tidy

build:
	go build -o $(BINARY_NAME) .

install: work build
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)$(BINARY_NAME)

test: build
	go test ./...

work:
	./scripts/use-go-work.sh

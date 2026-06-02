.PHONY: build deps test work install

BINARY_NAME=sitectl-omeka-s

deps: work
	go mod tidy

build:
	go build -o $(BINARY_NAME) .

install: build
	mv $(BINARY_NAME) /usr/local/bin

test: build
	go test ./...

work:
	./scripts/use-go-work.sh

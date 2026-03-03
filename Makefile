BINARY=dist/server

.PHONY: build run clean

build:
	mkdir -p dist
	go build -o $(BINARY) ./cmd/server

run:
	go run ./cmd/server

clean:
	rm -rf dist

BINARY_NAME=kubereplay
BUILD_DIR=bin

.PHONY: build clean install

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/kubereplay

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

test:
	go test ./...

.DEFAULT_GOAL := build

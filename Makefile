.PHONY: build run dev clean test test-cover

BINARY_NAME=mangosteen
BUILD_DIR=build
CMD_DIR=cmd/server

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

dev:
	$(shell go env GOPATH)/bin/air

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v -count=1 ./...

test-cover:
	go test -v -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

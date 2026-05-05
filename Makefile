.PHONY: build run dev clean

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

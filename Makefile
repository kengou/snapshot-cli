# Makefile for snapshot-cli
BINARY_NAME=snapshot-cli
BUILD_PATH=./cmd/main.go

.PHONY: build clean run docker

build:
	go build -o $(BINARY_NAME) $(BUILD_PATH)

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

docker:
	docker build -t $(BINARY_NAME):latest .

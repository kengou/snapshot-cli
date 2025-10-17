# Makefile for snapshot-cli
.PHONY: build clean run docker

.PHONY: build
build: build-snapshot-cli

build-%:
	go build -o bin/$* ./cmd/main.go

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

docker:
	docker build -t $(BINARY_NAME):latest .

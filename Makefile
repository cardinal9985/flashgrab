.PHONY: build run clean test vet

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o flashgrab ./cmd/flashgrab

run: build
	./flashgrab

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f flashgrab

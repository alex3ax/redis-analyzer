APP_NAME := redis-analyzer
VERSION  := $(shell git describe --tags --always --dirty)
BUILD_DIR := build

PLATFORMS := linux/amd64 darwin/amd64 darwin/arm64
ARCHIVES  := tar zip

LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build release clean

all: build

build:
	@echo "🔧 Building for host system..."
	go build -ldflags="$(LDFLAGS)" -o $(APP_NAME) .
	@echo "✅ Output: ./$(APP_NAME)"

release: clean
	@echo "🚀 Building release binaries..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d/ -f1); \
		ARCH=$$(echo $$platform | cut -d/ -f2); \
		OUT_DIR=$(BUILD_DIR)/$$OS-$$ARCH; \
		mkdir -p $$OUT_DIR; \
		GOOS=$$OS GOARCH=$$ARCH go build -ldflags="$(LDFLAGS)" -o $$OUT_DIR/$(APP_NAME) .; \
		for archive in $(ARCHIVES); do \
			case $$archive in \
				tar) tar -czf $(BUILD_DIR)/$(APP_NAME)-$$OS-$$ARCH.tar.gz -

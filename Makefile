APP_NAME := redis-analyzer
VERSION  := $(shell git describe --tags --always --dirty)
BUILD_DIR := build

PLATFORMS := linux/amd64 darwin/amd64 darwin/arm64
ARCHIVES  := tar

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
		OUT_BIN=$$OUT_DIR/$(APP_NAME); \
		mkdir -p $$OUT_DIR; \
		GOOS=$$OS GOARCH=$$ARCH CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $$OUT_BIN .; \
		for archive in $(ARCHIVES); do \
			case $$archive in \
				tar) COPYFILE_DISABLE=1 tar --format=ustar -czf $(BUILD_DIR)/$(APP_NAME)-$$OS-$$ARCH.tar.gz -C $$OUT_DIR $(APP_NAME) ;; \
				zip) zip -j $(BUILD_DIR)/$(APP_NAME)-$$OS-$$ARCH.zip $$OUT_BIN ;; \
			esac; \
		done; \
	done
	@echo "✅ Done. Binaries and archives are in $(BUILD_DIR)/"

clean:
	rm -rf $(BUILD_DIR) $(APP_NAME)

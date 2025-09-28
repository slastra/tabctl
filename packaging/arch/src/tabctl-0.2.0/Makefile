BINARY_NAME=tabctl
VERSION=$(shell git describe --tags --always --dirty)
BUILD_DIR=build
DIST_DIR=dist

# Go build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GCFLAGS=-gcflags="all=-trimpath=$(PWD)"
ASMFLAGS=-asmflags="all=-trimpath=$(PWD)"

# Platforms to build for
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: help build install clean test lint fmt deps dev release extensions

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

deps: ## Install Go dependencies
	go mod download
	go mod tidy

build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tabctl

install: build ## Install the binary to $GOPATH/bin
	go install $(LDFLAGS) ./cmd/tabctl

dev: ## Run in development mode (with hot reload if available)
	go run ./cmd/tabctl

test: ## Run tests
	go test -v ./...

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	goimports -w .

##@ Build

build-all: ## Build for all platforms
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTPUT_NAME=$(BINARY_NAME)-$$OS-$$ARCH; \
		if [ $$OS = "windows" ]; then OUTPUT_NAME=$$OUTPUT_NAME.exe; fi; \
		echo "Building for $$OS/$$ARCH..."; \
		GOOS=$$OS GOARCH=$$ARCH go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BUILD_DIR)/$$OUTPUT_NAME ./cmd/tabctl; \
	done

##@ Release

package: build-all ## Create release packages
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		BINARY_NAME_PLATFORM=$(BINARY_NAME)-$$OS-$$ARCH; \
		if [ $$OS = "windows" ]; then BINARY_NAME_PLATFORM=$$BINARY_NAME_PLATFORM.exe; fi; \
		ARCHIVE_NAME=$(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH; \
		if [ $$OS = "windows" ]; then \
			zip -j $(DIST_DIR)/$$ARCHIVE_NAME.zip $(BUILD_DIR)/$$BINARY_NAME_PLATFORM; \
		else \
			tar -czf $(DIST_DIR)/$$ARCHIVE_NAME.tar.gz -C $(BUILD_DIR) $$BINARY_NAME_PLATFORM; \
		fi; \
		echo "Created package: $$ARCHIVE_NAME"; \
	done

release: clean deps test lint package extensions ## Create a full release

##@ Extensions

extensions: ## Package browser extensions
	@mkdir -p $(DIST_DIR)
	@echo "Packaging Firefox extension..."
	@cd extensions/firefox && zip -r ../../$(DIST_DIR)/tabctl-firefox-extension.zip .
	@echo "Packaging Chrome extension..."
	@cd extensions/chrome && zip -r ../../$(DIST_DIR)/tabctl-chrome-extension.zip .

##@ Cleanup

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR) $(DIST_DIR)

##@ Installation

install-mediator: build ## Install native messaging mediator
	./$(BUILD_DIR)/$(BINARY_NAME) install

uninstall-mediator: ## Uninstall native messaging mediator
	@echo "Removing native messaging hosts..."
	@rm -f ~/.mozilla/native-messaging-hosts/tabctl_mediator.json
	@rm -f ~/.config/chromium/NativeMessagingHosts/tabctl_mediator.json
	@rm -f ~/.config/google-chrome/NativeMessagingHosts/tabctl_mediator.json
	@rm -f ~/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts/tabctl_mediator.json

##@ Tools

check-deps: ## Check if required tools are installed
	@which go > /dev/null || (echo "Go is not installed" && exit 1)
	@which git > /dev/null || (echo "Git is not installed" && exit 1)
	@echo "All required tools are installed"

version: ## Show version information
	@echo "Version: $(VERSION)"
	@go version
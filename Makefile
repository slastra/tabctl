BINARY_NAME=tabctl
VERSION=$(shell git describe --tags --always --dirty)
BUILD_DIR=build
DIST_DIR=dist

# Go build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GCFLAGS=-gcflags="all=-trimpath=$(PWD)"
ASMFLAGS=-asmflags="all=-trimpath=$(PWD)"
GO_BUILD=go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS)

# Platforms to build for
PLATFORMS=linux/amd64 linux/arm64

# Chrome extension signing key
CHROME_KEY=extensions/chrome-private-key.pem

.PHONY: help build install clean test lint fmt deps dev release extensions build-extensions

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

deps: ## Install Go dependencies
	go mod download
	go mod tidy

build: ## Build tabctl and tabctl-mediator
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tabctl
	$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-mediator ./cmd/tabctl-mediator

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

build-all: ## Build for all supported Linux architectures
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		echo "Building for linux/$$ARCH..."; \
		GOOS=linux GOARCH=$$ARCH $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-$$ARCH ./cmd/tabctl; \
		GOOS=linux GOARCH=$$ARCH $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-mediator-linux-$$ARCH ./cmd/tabctl-mediator; \
	done

##@ Release

package: build-all ## Create release tarballs for each Linux arch
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		ARCHIVE_NAME=$(BINARY_NAME)-$(VERSION)-linux-$$ARCH; \
		tar -czf $(DIST_DIR)/$$ARCHIVE_NAME.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-$$ARCH $(BINARY_NAME)-mediator-linux-$$ARCH; \
		echo "Created package: $$ARCHIVE_NAME"; \
	done

release: clean deps test lint package extensions ## Create a full release

##@ Extensions

build-extensions: ## Build browser extensions from shared source
	@./scripts/build-extensions.sh

extensions: build-extensions ## Package browser extensions for release
	@mkdir -p $(DIST_DIR)
	@echo "Packaging Firefox extension (zip)..."
	@cd extensions/firefox && zip -r ../../$(DIST_DIR)/tabctl-firefox-extension.zip . -x '.*'
	@echo "Packaging Chrome extension (zip)..."
	@cd extensions/chrome && zip -r ../../$(DIST_DIR)/tabctl-chrome-extension.zip . -x '.*'
	@echo "Packaging Chrome extension (crx)..."
	@CHROMIUM=$$(which brave || which google-chrome || which chromium 2>/dev/null); \
	if [ -z "$$CHROMIUM" ]; then \
		echo "Error: No Chromium-based browser found for CRX packaging"; \
		exit 1; \
	fi; \
	if [ ! -f $(CHROME_KEY) ]; then \
		echo "Error: Chrome signing key not found at $(CHROME_KEY)"; \
		exit 1; \
	fi; \
	$$CHROMIUM --pack-extension=$$(pwd)/extensions/chrome --pack-extension-key=$$(pwd)/$(CHROME_KEY) 2>/dev/null; \
	mv extensions/chrome.crx $(DIST_DIR)/tabctl-chrome-extension.crx
	@echo "Extensions packaged in $(DIST_DIR)/"

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
	@rm -f ~/.config/net.imput.helium/NativeMessagingHosts/tabctl_mediator.json

##@ Tools

check-deps: ## Check if required tools are installed
	@which go > /dev/null || (echo "Go is not installed" && exit 1)
	@which git > /dev/null || (echo "Git is not installed" && exit 1)
	@echo "All required tools are installed"

version: ## Show version information
	@echo "Version: $(VERSION)"
	@go version
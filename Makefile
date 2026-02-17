APP_NAME := gitcrn
VERSION ?= dev
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test build-linux build-windows release-assets install update clean clean-go-cache clean-all

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/gitcrn

test:
	go test ./...

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-linux-amd64 ./cmd/gitcrn

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-windows-amd64.exe ./cmd/gitcrn

release-assets:
	./scripts/release.sh --version $(VERSION)

install:
	./scripts/install.sh

update:
	./scripts/update.sh

clean:
	rm -rf bin dist

clean-go-cache:
	go clean -cache -modcache

clean-all: clean clean-go-cache

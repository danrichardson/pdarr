BINARY     := sqzarr
CMD        := ./cmd/sqzarr
DIST       := dist
GOFLAGS    := -trimpath -ldflags="-s -w"

.PHONY: all build test test-integration lint clean release frontend

all: frontend build

build:
	go build $(GOFLAGS) -o $(BINARY) $(CMD)

# Build for Linux amd64 (primary target: Proxmox LXC)
build-linux:
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 $(CMD)

# Build for macOS arm64 (M-series Mac Mini)
build-darwin:
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64 $(CMD)

# Unit tests only (no ffmpeg required)
test:
	go test ./...

# Integration tests (requires ffmpeg on PATH)
test-integration:
	go test -tags integration -v ./...

lint:
	golangci-lint run ./...

frontend:
	cd frontend && npm ci && npm run build

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)/

release: clean
	mkdir -p $(DIST)
	$(MAKE) build-linux
	$(MAKE) build-darwin
	@echo "Binaries in $(DIST)/"

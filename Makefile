PORT ?= 7070
DEMO ?= indie-dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build build-desktop bundle-desktop generate-icon dev lint demo-indie demo-ops demo-agency demo-sre demo-family demo-prompt-eng release-port build-desktop-universal bundle-desktop-release release-desktop

# Build CLI binary (no CGO, single arch)
build:
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o liveboard ./cmd/liveboard

# Compile desktop binary only (fast recompile for dev — assumes bundle structure already exists)
build-desktop:
	@rm -f LiveBoard.app/Contents/MacOS/liveboard-desktop
	CGO_ENABLED=1 CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags production -ldflags '$(LDFLAGS)' -o LiveBoard.app/Contents/MacOS/liveboard-desktop ./cmd/liveboard-desktop

# Regenerate icon.icns from SVG source (requires rsvg-convert and iconutil)
generate-icon:
	@mkdir -p LiveBoard.iconset
	@rsvg-convert -w 16 -h 16 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_16x16.png
	@rsvg-convert -w 32 -h 32 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_16x16@2x.png
	@rsvg-convert -w 32 -h 32 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_32x32.png
	@rsvg-convert -w 64 -h 64 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_32x32@2x.png
	@rsvg-convert -w 128 -h 128 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_128x128.png
	@rsvg-convert -w 256 -h 256 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_128x128@2x.png
	@rsvg-convert -w 256 -h 256 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_256x256.png
	@rsvg-convert -w 512 -h 512 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_256x256@2x.png
	@rsvg-convert -w 512 -h 512 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_512x512.png
	@rsvg-convert -w 1024 -h 1024 web/img/liveboard-icon-macos.svg -o LiveBoard.iconset/icon_512x512@2x.png
	@iconutil -c icns LiveBoard.iconset -o cmd/liveboard-desktop/icon.icns
	@rm -rf LiveBoard.iconset
	@echo "Generated icon.icns from SVG"

# Build a complete .app bundle for local use (binary + Info.plist + icon)
bundle-desktop: build-desktop
	@mkdir -p LiveBoard.app/Contents/Resources
	@cp cmd/liveboard-desktop/Info.plist LiveBoard.app/Contents/
	@cp cmd/liveboard-desktop/icon.icns LiveBoard.app/Contents/Resources/
	@echo "Built LiveBoard.app"

# Build universal (arm64 + amd64) binary via lipo
build-desktop-universal:
	@mkdir -p dist
	CGO_ENABLED=1 CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		GOARCH=arm64 go build -tags production -ldflags '$(LDFLAGS)' -o dist/liveboard-desktop-arm64 ./cmd/liveboard-desktop
	CGO_ENABLED=1 CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		GOARCH=amd64 go build -tags production -ldflags '$(LDFLAGS)' -o dist/liveboard-desktop-amd64 ./cmd/liveboard-desktop
	lipo -create -output dist/liveboard-desktop dist/liveboard-desktop-arm64 dist/liveboard-desktop-amd64
	@rm -f dist/liveboard-desktop-arm64 dist/liveboard-desktop-amd64
	@echo "Built universal binary: dist/liveboard-desktop"

# Assemble release .app bundle with universal binary, stamped version, and zip archive
bundle-desktop-release: build-desktop-universal
	@mkdir -p LiveBoard.app/Contents/MacOS LiveBoard.app/Contents/Resources
	@cp dist/liveboard-desktop LiveBoard.app/Contents/MacOS/liveboard-desktop
	@cp cmd/liveboard-desktop/Info.plist LiveBoard.app/Contents/
	@plutil -replace CFBundleVersion -string "$(VERSION)" LiveBoard.app/Contents/Info.plist
	@plutil -replace CFBundleShortVersionString -string "$(VERSION)" LiveBoard.app/Contents/Info.plist
	@cp cmd/liveboard-desktop/icon.icns LiveBoard.app/Contents/Resources/
	@ditto -c -k --keepParent LiveBoard.app "LiveBoard-$(VERSION)-macos-universal.zip"
	@echo "Built LiveBoard-$(VERSION)-macos-universal.zip"

# Upload release zip to GitHub and update Homebrew cask
release-desktop: bundle-desktop-release
	@TAG=$$(git describe --tags --abbrev=0); \
	gh release upload "$$TAG" "LiveBoard-$(VERSION)-macos-universal.zip" --clobber
	bash scripts/update-desktop-cask.sh "$(VERSION)"

# Kill any process occupying the dev server port
release-port:
	-lsof -ti :$(PORT) | xargs kill -9 2>/dev/null

# Run golangci-lint
lint:
	golangci-lint run ./...

# Start dev server with live reload (uses air if available)
dev: release-port
	@if command -v air >/dev/null 2>&1; then \
		NO_CACHE=1 air -- serve --dir=demo/$(DEMO)/ --port $(PORT); \
	else \
		echo "Tip: install 'air' for live reload: go install github.com/air-verse/air@latest"; \
		NO_CACHE=1 go run ./cmd/liveboard/... serve --dir=demo/$(DEMO)/ --port $(PORT); \
	fi

demo-indie: DEMO=indie-dev
demo-indie: dev

demo-ops: DEMO=ops-infra
demo-ops: dev

demo-agency: DEMO=agency
demo-agency: dev

demo-sre: DEMO=sre
demo-sre: dev

demo-family: DEMO=family
demo-family: dev

demo-prompt-eng: DEMO=prompt-eng
demo-prompt-eng: dev

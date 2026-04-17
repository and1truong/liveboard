PORT ?= 7070
DEMO ?= indie-dev
DEMOS     := indie-dev sre ops-infra agency family prompt-eng student-y7 tutorial
DEMO_ARG  := $(filter $(DEMOS), $(MAKECMDGOALS))
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build build-desktop bundle-desktop generate-icon dev lint demo $(DEMOS) release-port build-desktop-universal bundle-desktop-release release-desktop ipad-framework ipad-project ipad shell renderer frontend

# Build CLI binary (no CGO, single arch)
build: frontend
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o liveboard ./cmd/liveboard

# Compile desktop binary only (fast recompile for dev — assumes bundle structure already exists)
build-desktop:
	@rm -f LiveBoard.app/Contents/MacOS/liveboard-desktop
	CGO_ENABLED=1 CGO_CFLAGS="-Wno-deprecated-declarations" CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags production -ldflags '$(LDFLAGS)' -o LiveBoard.app/Contents/MacOS/liveboard-desktop ./cmd/liveboard-desktop

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
bundle-desktop: frontend build-desktop
	@mkdir -p LiveBoard.app/Contents/Resources
	@cp cmd/liveboard-desktop/Info.plist LiveBoard.app/Contents/
	@cp cmd/liveboard-desktop/icon.icns LiveBoard.app/Contents/Resources/
	@echo "Built LiveBoard.app"

# Build universal (arm64 + amd64) binary via lipo
build-desktop-universal:
	@mkdir -p dist
	CGO_ENABLED=1 CGO_CFLAGS="-Wno-deprecated-declarations" CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
		GOARCH=arm64 go build -tags production -ldflags '$(LDFLAGS)' -o dist/liveboard-desktop-arm64 ./cmd/liveboard-desktop
	CGO_ENABLED=1 CGO_CFLAGS="-Wno-deprecated-declarations" CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
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

# Build shell + stub bundles via Vite (multi-page build)
shell:
	cd web/shared && bun install --frozen-lockfile
	cd web/shell && bun install --frozen-lockfile
	cd web/shell && bunx --bun vite build

.PHONY: renderer
renderer:
	cd web/renderer/default && bun install --frozen-lockfile
	cd web/renderer/default && bunx --bun vite build
	@$(MAKE) bundle-check

.PHONY: bundle-check
bundle-check:
	bash scripts/check-bundle-size.sh

.PHONY: frontend
frontend: shell renderer

.PHONY: renderer-dev
renderer-dev:
	cd web/renderer/default && bun install --frozen-lockfile
	cd web/renderer/default && bunx --bun vite build --mode development --minify=false

# Dev server for /app/ (shell + renderer) — HMR, unminified, dev React.
# Shell Vite serves http://localhost:7070/app/ and proxies
# /app/renderer/default/* to renderer Vite at :5173.
# No Go involved — /app/ is browser-only (LocalAdapter + localStorage).
.PHONY: dev-app
dev-app:
	cd web/shell && bun install --frozen-lockfile
	cd web/renderer/default && bun install --frozen-lockfile
	@echo "Shell:    http://localhost:7070/app/"
	@echo "Stub:     http://localhost:7070/app/renderer-stub/"
	@echo "Renderer: proxied from :5173"
	@( cd web/renderer/default && bunx --bun vite --port 5173 --strictPort ) & \
	RENDERER_PID=$$!; \
	trap "kill $$RENDERER_PID 2>/dev/null" EXIT; \
	cd web/shell && bunx --bun vite

# Dev server for adapter-test with Vite HMR.
# Go serves /api/v1 on :7070 and reverse-proxies /app/* to shell Vite (:5180)
# and /app/renderer/default/* to renderer Vite (:5173). HMR WS is proxied
# through Go so TS/CSS edits hot-reload without rebuild.
.PHONY: adapter-test
adapter-test:
	-lsof -ti :7070 | xargs kill -9 2>/dev/null
	-lsof -ti :5173 | xargs kill -9 2>/dev/null
	-lsof -ti :5180 | xargs kill -9 2>/dev/null
	cd web/shared && bun install --frozen-lockfile
	cd web/shell && bun install --frozen-lockfile
	cd web/renderer/default && bun install --frozen-lockfile
	@echo "Go:       http://localhost:7070/app/"
	@echo "Shell:    :5180 (proxied via Go)"
	@echo "Renderer: :5173 (proxied via Go)"
	@( cd web/renderer/default && bunx --bun vite --port 5173 --strictPort ) & \
	RENDERER_PID=$$!; \
	( cd web/shell && bunx --bun vite --port 5180 --strictPort ) & \
	SHELL_PID=$$!; \
	trap "kill $$RENDERER_PID $$SHELL_PID 2>/dev/null" EXIT; \
	LIVEBOARD_SHELL_DEV_URL=http://localhost:5180 \
	LIVEBOARD_RENDERER_DEV_URL=http://localhost:5173 \
	go run ./cmd/liveboard serve --dir ./demo/indie-dev --port 7070

# Kill any process occupying the dev server port
release-port:
	-lsof -ti :$(PORT) | xargs kill -9 2>/dev/null

# Run golangci-lint
lint:
	golangci-lint run ./...

# Start dev server with live reload (uses air if available).
# Requires `make frontend` once to build the embedded shell/renderer bundles.
dev: release-port
	@if command -v air >/dev/null 2>&1; then \
		NO_CACHE=1 air -- serve --dir=demo/$(DEMO)/ --port $(PORT); \
	else \
		echo "Tip: install 'air' for live reload: go install github.com/air-verse/air@latest"; \
		NO_CACHE=1 go run ./cmd/liveboard/... serve --dir=demo/$(DEMO)/ --port $(PORT); \
	fi

$(DEMOS):
	@:

demo: release-port
	@bash -c '\
		demos=(indie-dev sre ops-infra agency family prompt-eng student-y7 tutorial); \
		selected="$(DEMO_ARG)"; \
		if [ -z "$$selected" ]; then \
			echo "Available demos:"; \
			for i in "$${!demos[@]}"; do \
				d="$${demos[$$i]}"; \
				label=$$(grep -o "\"site-name\"[^,}]*" demo/$$d/settings.json 2>/dev/null | sed "s/.*: *\"//;s/\".*//"); \
				printf "  %d) %-16s %s\n" "$$((i+1))" "$$d" "$$label"; \
			done; \
			printf "Select [1-$${#demos[@]}] or name: "; \
			read -r choice; \
			if [[ "$$choice" =~ ^[0-9]+$$ ]] && [ "$$choice" -ge 1 ] && [ "$$choice" -le "$${#demos[@]}" ]; then \
				selected="$${demos[$$((choice-1))]}"; \
			else \
				selected="$$choice"; \
			fi; \
		fi; \
		echo "Starting demo: $$selected (port $(PORT))"; \
		if command -v air >/dev/null 2>&1; then \
			NO_CACHE=1 air -- serve --dir=demo/$$selected/ --port $(PORT); \
		else \
			echo "Tip: install air for live reload: go install github.com/air-verse/air@latest"; \
			NO_CACHE=1 go run ./cmd/liveboard/... serve --dir=demo/$$selected/ --port $(PORT); \
		fi \
	'

# Build Go xcframework for iPad (requires gomobile: go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init)
ipad-framework: frontend
	gomobile bind -target=ios -o ipad/Gobridge.xcframework ./mobile/gobridge
	@echo "Built ipad/Gobridge.xcframework"

# Generate Xcode project for iPad app (requires xcodegen: brew install xcodegen)
ipad-project: ipad-framework
	cd ipad && xcodegen generate
	@echo "Generated ipad/LiveBoard.xcodeproj"

SIMULATOR_DEST ?= generic/platform=iOS Simulator

# Build iPad app for simulator
ipad: ipad-project
	cd ipad && xcodebuild -project LiveBoard.xcodeproj \
		-scheme LiveBoard \
		-destination '$(SIMULATOR_DEST)' \
		-configuration Debug \
		build


.PHONY: npm-build
npm-build:
	cd web/shared && bun run build
	cd web/shell && bun run build:npm
	cd web/renderer/default && bun run build:npm

.PHONY: npm-publish
npm-publish: npm-build
	cd web/shared && bun publish
	cd web/shell && bun publish
	cd web/renderer/default && bun publish

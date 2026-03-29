ROOT := $(shell cd .. && pwd)
GOBRIDGE_PKG := github.com/and1truong/liveboard/mobile/gobridge
SIMULATOR_DEST ?= generic/platform=iOS Simulator

.PHONY: framework project build clean

# Build the Go xcframework for iOS using gomobile.
# Requires: go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init
framework:
	$(MAKE) -C $(ROOT) css
	cd $(ROOT) && gomobile bind -target=ios -o ipad/Gobridge.xcframework $(GOBRIDGE_PKG)
	@echo "Built Gobridge.xcframework"

# Generate Xcode project from project.yml (requires xcodegen).
# Install: brew install xcodegen
project: framework
	xcodegen generate
	@echo "Generated LiveBoard.xcodeproj — open it in Xcode"

# Build the app via xcodebuild (simulator, debug).
build: project
	xcodebuild -project LiveBoard.xcodeproj \
		-scheme LiveBoard \
		-destination '$(SIMULATOR_DEST)' \
		-configuration Debug \
		build

# Remove build artifacts.
clean:
	rm -rf Gobridge.xcframework LiveBoard.xcodeproj build DerivedData

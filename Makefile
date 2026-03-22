PORT ?= 7070
DEMO ?= indie-dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build dev lint demo-indie demo-ops demo-agency demo-sre demo-family demo-prompt-eng release-port

build:
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o liveboard ./cmd/liveboard

release-port:
	-lsof -ti :$(PORT) | xargs kill -9 2>/dev/null

lint:
	golangci-lint run ./...

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

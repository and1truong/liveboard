PORT ?= 7070

.PHONY: dev release-port

release-port:
	-lsof -ti :$(PORT) | xargs kill -9 2>/dev/null

dev: release-port
	@if command -v air >/dev/null 2>&1; then \
		air -- serve --dir=demo/ --port $(PORT); \
	else \
		echo "Tip: install 'air' for live reload: go install github.com/air-verse/air@latest"; \
		go run ./cmd/liveboard/... serve --dir=demo/ --port $(PORT); \
	fi

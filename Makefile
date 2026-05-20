.PHONY: all fmt vet lint test cover tidy clean help

GO ?= go
PKGS ?= ./...

all: fmt vet test

help:
	@echo "Available targets:"
	@echo "  fmt    - gofmt all sources"
	@echo "  vet    - go vet $(PKGS)"
	@echo "  lint   - run golangci-lint (requires golangci-lint installed)"
	@echo "  test   - run unit tests"
	@echo "  cover  - run tests with coverage report (coverage.txt + coverage.html)"
	@echo "  tidy   - go mod tidy"
	@echo "  clean  - remove build artifacts"

fmt:
	$(GO) fmt $(PKGS)

vet:
	$(GO) vet $(PKGS)

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not installed; see https://golangci-lint.run/usage/install/"; \
		exit 1; \
	}
	golangci-lint run $(PKGS)

test:
	$(GO) test -race -count=1 $(PKGS)

cover:
	$(GO) test -race -count=1 -coverprofile=coverage.txt -covermode=atomic $(PKGS)
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

tidy:
	$(GO) mod tidy

clean:
	rm -f coverage.txt coverage.html

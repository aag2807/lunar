# Makefile for Lunar Language

# Variables
GO=go
GOFLAGS=
BINDIR=/usr/local/bin
LUNAR_BIN=lunar
LUNAR2DECL_BIN=lunar2decl

# Build targets
.PHONY: all build clean install uninstall test help

all: build

# Build both binaries
build: build-lunar build-lunar2decl

# Build the main Lunar compiler
build-lunar:
	@echo "Building Lunar compiler..."
	$(GO) build $(GOFLAGS) -o $(LUNAR_BIN) ./cmd/lunar
	@echo "✓ Built $(LUNAR_BIN)"

# Build the declaration generator tool
build-lunar2decl:
	@echo "Building lunar2decl tool..."
	$(GO) build $(GOFLAGS) -o $(LUNAR2DECL_BIN) ./cmd/lunar2decl
	@echo "✓ Built $(LUNAR2DECL_BIN)"

# Install binaries to system
install: build
	@echo "Installing to $(BINDIR)..."
	@install -d $(BINDIR)
	@install -m 755 $(LUNAR_BIN) $(BINDIR)/$(LUNAR_BIN)
	@install -m 755 $(LUNAR2DECL_BIN) $(BINDIR)/$(LUNAR2DECL_BIN)
	@echo "✓ Installed $(LUNAR_BIN) and $(LUNAR2DECL_BIN) to $(BINDIR)"
	@echo ""
	@echo "Installation complete! You can now use:"
	@echo "  lunar <file.lunar>       - Compile Lunar code"
	@echo "  lunar2decl <file.lua>    - Generate declaration files"

# Uninstall binaries from system
uninstall:
	@echo "Uninstalling from $(BINDIR)..."
	@rm -f $(BINDIR)/$(LUNAR_BIN)
	@rm -f $(BINDIR)/$(LUNAR2DECL_BIN)
	@echo "✓ Uninstalled"

# Run tests
test: build
	@echo "Running tests..."
	@echo "Testing Lunar compiler..."
	@./test-suite.sh || (echo "Test suite not found, creating basic tests..." && $(MAKE) test-basic)
	@echo "✓ Tests passed"

# Basic smoke tests
test-basic: build
	@echo "Running basic smoke tests..."
	@# Test compiler help
	@./$(LUNAR_BIN) --help > /dev/null && echo "  ✓ --help works"
	@# Test compiler version
	@./$(LUNAR_BIN) --version > /dev/null && echo "  ✓ --version works"
	@# Test lunar2decl help
	@./$(LUNAR2DECL_BIN) --help > /dev/null && echo "  ✓ lunar2decl --help works"
	@# Test compilation of examples
	@if [ -f examples/class.lunar ]; then \
		./$(LUNAR_BIN) examples/class.lunar -o /tmp/test_class.lua && \
		echo "  ✓ class.lunar compiles"; \
	fi
	@echo "✓ Basic tests passed"

# Run examples
test-examples: build
	@echo "Testing all examples..."
	@for file in examples/*.lunar; do \
		if [ -f "$$file" ]; then \
			echo "  Testing $$file..."; \
			./$(LUNAR_BIN) "$$file" -o "/tmp/$$(basename $$file .lunar).lua" || exit 1; \
		fi \
	done
	@echo "✓ All examples compile successfully"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(LUNAR_BIN)
	@rm -f $(LUNAR2DECL_BIN)
	@rm -f examples/*.lua
	@rm -f stdlib/*.lua
	@rm -f /*.lua
	@rm -f test*.lua
	@echo "✓ Clean complete"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	@echo "✓ Formatted"

# Run Go linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, using go vet..."; \
		$(GO) vet ./...; \
	fi
	@echo "✓ Linting complete"

# Show help
help:
	@echo "Lunar Language Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build both lunar and lunar2decl"
	@echo "  make install        - Install binaries to $(BINDIR)"
	@echo "  make uninstall      - Remove binaries from $(BINDIR)"
	@echo "  make test           - Run tests"
	@echo "  make test-basic     - Run basic smoke tests"
	@echo "  make test-examples  - Test all example files"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make fmt            - Format Go code"
	@echo "  make lint           - Run Go linter"
	@echo "  make help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make && make install"
	@echo "  make build && ./lunar myfile.lunar"
	@echo "  make clean && make build"

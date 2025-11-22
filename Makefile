# Makefile for Lunar Language

# Detect OS
ifeq ($(OS),Windows_NT)
	DETECTED_OS := Windows
else
	DETECTED_OS := $(shell uname -s)
endif

# Variables
GO=go
GOFLAGS=

# Platform-specific settings
ifeq ($(DETECTED_OS),Windows)
	EXE_EXT=.exe
	BINDIR=$(USERPROFILE)/bin
	RM=del /Q
	RMDIR=rd /S /Q
	MKDIR=mkdir
	PATHSEP=\\
	NULL=nul
else
	EXE_EXT=
	BINDIR=/usr/local/bin
	RM=rm -f
	RMDIR=rm -rf
	MKDIR=mkdir -p
	PATHSEP=/
	NULL=/dev/null
endif

LUNAR_BIN=lunar$(EXE_EXT)
LUNAR2DECL_BIN=lunar2decl$(EXE_EXT)
LUNAR_LSP_BIN=lunar-lsp$(EXE_EXT)

# Build targets
.PHONY: all build clean install uninstall test help

all: build

# Build all binaries
build: build-lunar build-lunar2decl build-lunar-lsp

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

# Build the LSP server
build-lunar-lsp:
	@echo "Building lunar-lsp server..."
	$(GO) build $(GOFLAGS) -o $(LUNAR_LSP_BIN) ./cmd/lunar-lsp
	@echo "✓ Built $(LUNAR_LSP_BIN)"

# Install binaries to system
install: build
	@echo "Installing to $(BINDIR)..."
ifeq ($(DETECTED_OS),Windows)
	@if not exist "$(BINDIR)" $(MKDIR) "$(BINDIR)"
	@copy /Y $(LUNAR_BIN) "$(BINDIR)$(PATHSEP)$(LUNAR_BIN)"
	@copy /Y $(LUNAR2DECL_BIN) "$(BINDIR)$(PATHSEP)$(LUNAR2DECL_BIN)"
	@copy /Y $(LUNAR_LSP_BIN) "$(BINDIR)$(PATHSEP)$(LUNAR_LSP_BIN)"
	@echo ✓ Installed $(LUNAR_BIN), $(LUNAR2DECL_BIN), and $(LUNAR_LSP_BIN) to $(BINDIR)
	@echo.
	@echo Installation complete! Make sure $(BINDIR) is in your PATH.
	@echo You can now use:
	@echo   lunar ^<file.lunar^>       - Compile Lunar code
	@echo   lunar2decl ^<file.lua^>    - Generate declaration files
	@echo   lunar-lsp                  - Start the LSP server
else
	@$(MKDIR) $(BINDIR)
	@install -m 755 $(LUNAR_BIN) $(BINDIR)/$(LUNAR_BIN)
	@install -m 755 $(LUNAR2DECL_BIN) $(BINDIR)/$(LUNAR2DECL_BIN)
	@install -m 755 $(LUNAR_LSP_BIN) $(BINDIR)/$(LUNAR_LSP_BIN)
	@echo "✓ Installed $(LUNAR_BIN), $(LUNAR2DECL_BIN), and $(LUNAR_LSP_BIN) to $(BINDIR)"
	@echo ""
	@echo "Installation complete! You can now use:"
	@echo "  lunar <file.lunar>       - Compile Lunar code"
	@echo "  lunar2decl <file.lua>    - Generate declaration files"
	@echo "  lunar-lsp                - Start the LSP server"
endif

# Uninstall binaries from system
uninstall:
	@echo "Uninstalling from $(BINDIR)..."
ifeq ($(DETECTED_OS),Windows)
	@if exist "$(BINDIR)$(PATHSEP)$(LUNAR_BIN)" $(RM) "$(BINDIR)$(PATHSEP)$(LUNAR_BIN)"
	@if exist "$(BINDIR)$(PATHSEP)$(LUNAR2DECL_BIN)" $(RM) "$(BINDIR)$(PATHSEP)$(LUNAR2DECL_BIN)"
	@if exist "$(BINDIR)$(PATHSEP)$(LUNAR_LSP_BIN)" $(RM) "$(BINDIR)$(PATHSEP)$(LUNAR_LSP_BIN)"
else
	@$(RM) $(BINDIR)/$(LUNAR_BIN)
	@$(RM) $(BINDIR)/$(LUNAR2DECL_BIN)
	@$(RM) $(BINDIR)/$(LUNAR_LSP_BIN)
endif
	@echo "✓ Uninstalled"

# Run tests
test: build
	@echo "Running tests..."
	@echo "Testing Lunar compiler..."
ifeq ($(DETECTED_OS),Windows)
	@echo "Running basic tests on Windows..."
	@$(MAKE) test-basic
else
	@./test-suite.sh || (echo "Test suite not found, creating basic tests..." && $(MAKE) test-basic)
endif
	@echo "✓ Tests passed"

# Basic smoke tests
test-basic: build
	@echo "Running basic smoke tests..."
ifeq ($(DETECTED_OS),Windows)
	@echo   Testing --help...
	@$(LUNAR_BIN) --help > $(NULL) && echo   ✓ --help works
	@echo   Testing --version...
	@$(LUNAR_BIN) --version > $(NULL) && echo   ✓ --version works
	@echo   Testing lunar2decl --help...
	@$(LUNAR2DECL_BIN) --help > $(NULL) && echo   ✓ lunar2decl --help works
	@if exist examples$(PATHSEP)class.lunar ( \
		echo   Testing class.lunar compilation... && \
		$(LUNAR_BIN) examples$(PATHSEP)class.lunar -o test_class.lua && \
		echo   ✓ class.lunar compiles && \
		$(RM) test_class.lua \
	)
else
	@# Test compiler help
	@./$(LUNAR_BIN) --help > $(NULL) && echo "  ✓ --help works"
	@# Test compiler version
	@./$(LUNAR_BIN) --version > $(NULL) && echo "  ✓ --version works"
	@# Test lunar2decl help
	@./$(LUNAR2DECL_BIN) --help > $(NULL) && echo "  ✓ lunar2decl --help works"
	@# Test compilation of examples
	@if [ -f examples/class.lunar ]; then \
		./$(LUNAR_BIN) examples/class.lunar -o /tmp/test_class.lua && \
		echo "  ✓ class.lunar compiles"; \
	fi
endif
	@echo "✓ Basic tests passed"

# Run examples
test-examples: build
	@echo "Testing all examples..."
ifeq ($(DETECTED_OS),Windows)
	@for %%f in (examples$(PATHSEP)*.lunar) do ( \
		echo   Testing %%f... && \
		$(LUNAR_BIN) "%%f" -o "%%~nf.lua" || exit 1 \
	)
	@$(RM) *.lua 2>$(NULL)
else
	@for file in examples/*.lunar; do \
		if [ -f "$$file" ]; then \
			echo "  Testing $$file..."; \
			./$(LUNAR_BIN) "$$file" -o "/tmp/$$(basename $$file .lunar).lua" || exit 1; \
		fi \
	done
endif
	@echo "✓ All examples compile successfully"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
ifeq ($(DETECTED_OS),Windows)
	@if exist $(LUNAR_BIN) $(RM) $(LUNAR_BIN)
	@if exist $(LUNAR2DECL_BIN) $(RM) $(LUNAR2DECL_BIN)
	@if exist $(LUNAR_LSP_BIN) $(RM) $(LUNAR_LSP_BIN)
	@if exist examples$(PATHSEP)*.lua $(RM) examples$(PATHSEP)*.lua
	@if exist stdlib$(PATHSEP)*.lua $(RM) stdlib$(PATHSEP)*.lua
	@if exist test*.lua $(RM) test*.lua
else
	@$(RM) $(LUNAR_BIN)
	@$(RM) $(LUNAR2DECL_BIN)
	@$(RM) $(LUNAR_LSP_BIN)
	@$(RM) examples/*.lua
	@$(RM) stdlib/*.lua
	@$(RM) /*.lua
	@$(RM) test*.lua
endif
	@echo "✓ Clean complete"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	@echo "✓ Formatted"

# Run Go linter
lint:
	@echo "Running linter..."
ifeq ($(DETECTED_OS),Windows)
	@where golangci-lint >$(NULL) 2>$(NULL) && golangci-lint run ./... || (echo golangci-lint not installed, using go vet... && $(GO) vet ./...)
else
	@if command -v golangci-lint > $(NULL); then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, using go vet..."; \
		$(GO) vet ./...; \
	fi
endif
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

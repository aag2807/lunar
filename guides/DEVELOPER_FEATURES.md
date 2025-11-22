# Lunar Developer Features

## Test Runner Features ✅

### Fully Implemented

- **✅ Colored Output** - Pass/fail tests shown in green/red with visual indicators (✓/✗)
  - Located in: `vendor/testing/testing.lua`
  - Automatic terminal color detection
  - Usage: Enabled by default in all tests

- **✅ Test Filtering by Pattern** - Run only tests matching a specific pattern
  - Usage: `lunar --test . --filter=math`
  - Filters by file path (case-insensitive)
  - Shows which tests were filtered

- **✅ Watch Mode** - Automatically re-run tests when files change
  - Usage: `lunar --test-watch .`
  - Polls file system every second
  - Clears screen and shows which files changed
  - Can be combined with filtering: `lunar --test-watch . --filter=unit`

- **✅ Test Discovery** - Automatic discovery of test files
  - Patterns: `*_test.lunar` or `*.test.lunar`
  - Skips vendor and hidden directories
  - Recursive directory scanning

- **✅ Test Lifecycle Hooks** - beforeEach, afterEach, beforeAll, afterAll
  - Defined in vendor/testing/testing.lua
  - Full Jest-like testing API

### Partially Implemented

- **⚠️ Coverage Reporting** - Infrastructure added, but requires code instrumentation
  - Coverage module created: `vendor/testing/coverage.lua`
  - Flag added: `lunar --test . --coverage`
  - **What's missing:** Source code instrumentation to track line execution
  - **To complete:** Would require either:
    - Pre-compilation instrumentation pass
    - Lua-level debugger hooks
    - Integration with luacov or similar tools

## REPL Features ✅

### Fully Implemented

- **✅ Command History** - Persistent history across sessions
  - Saved to: `~/.lunar_history`
  - Last 1000 commands saved
  - View with: `history` command
  - Automatic on REPL startup/shutdown

- **✅ Multi-line Input** - Automatic detection of incomplete blocks
  - Detects: function, if, while, for, do, class, interface, enum, namespace
  - Continues until 'end' is reached
  - Shows `...` prompt for continuation
  - Tracks nesting level

- **✅ Better Error Messages** - Clear, detailed error reporting
  - Parse errors: Show line and column number
  - Type errors: Show line number and message
  - Context-aware error messages

- **✅ Symbol Listing** - View all available symbols in current context
  - Command: `symbols`
  - Shows: Variables, Functions, Classes, Interfaces, Enums, Type Aliases
  - Displays function signatures with parameter and return types
  - Grouped by symbol type
  - Total count displayed

- **✅ Context Management** - Commands for managing REPL state
  - `clear` - Reset accumulated context
  - `context` - Show number of accumulated statements
  - `help` - Show all available commands

### Not Implemented

- **❌ LSP Auto-completion** - Tab completion using Language Server
  - **Why not:** Requires readline library (not in Go stdlib)
  - **Alternatives:**
    - Use `symbols` command to see available symbols
    - External editors (VS Code, Neovim) have full LSP auto-completion
  - **To implement:** Would need:
    - Third-party readline library (e.g., github.com/chzyer/readline)
    - Integration with LSP server
    - Tab-completion callback system
    - Completion filtering and ranking

## Quick Reference

### Test Commands
```bash
# Run all tests
lunar --test .

# Run specific tests
lunar --test ./tests

# Filter tests by pattern
lunar --test . --filter=unit

# Watch mode
lunar --test-watch .

# Watch with filter
lunar --test-watch . --filter=integration

# With coverage (framework in place, needs instrumentation)
lunar --test . --coverage
```

### REPL Commands
```bash
# Start REPL
lunar --repl

# Inside REPL:
help      # Show help
symbols   # List all available symbols
history   # Show command history
clear     # Clear context
context   # Show context size
exit      # Exit REPL
```

### REPL Example Session
```lunar
>>> local x: number = 42
>>> function add(a: number, b: number): number
...   return a + b
... end
>>> symbols

Available symbols:

  Variables:
    - x

  Functions:
    - add(a: number, b: number): number

Total: 2 symbols
```

## Implementation Notes

### Test Runner Architecture
- Test files discovered via file system walk
- Tests bundled together with vendor/testing module
- Executed in Lua interpreter
- Results formatted and displayed with colors

### REPL Architecture
- Statement-based parsing and accumulation
- Type checking against accumulated context
- Lua code generation for each statement
- Persistent history via file storage
- AST traversal for symbol extraction

### Future Enhancements

**Coverage Reporting:**
- Add source map support for coverage tracking
- Implement line instrumentation pass
- Generate HTML coverage reports
- Track branch coverage

**LSP Auto-completion:**
- Add readline library dependency
- Implement tab-completion callback
- Integrate with LSP completion endpoint
- Add fuzzy matching for suggestions
- Support snippet completion

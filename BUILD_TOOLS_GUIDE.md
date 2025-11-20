# Lunar Build Tools Guide

This guide documents the build tools and workflow improvements in Lunar, including watch mode enhancements, build profiles, and optimization features.

---

## Table of Contents

1. [Build Profiles](#build-profiles)
2. [Watch Mode Enhancements](#watch-mode-enhancements)
3. [Source Map Improvements](#source-map-improvements)
4. [Optimization Options](#optimization-options)
5. [Performance Tips](#performance-tips)
6. [Examples](#examples)

---

## Build Profiles

### Overview

Build profiles provide predefined sets of compiler options optimized for different development scenarios.

### Available Profiles

#### Development Mode (`--mode dev`)

Optimized for fast development cycles and debugging:

```bash
lunar main.lunar --mode dev
```

**Automatically enables:**
- ✅ Source maps generation (`--source-map`)
- ✅ Incremental compilation cache
- ❌ Optimizations disabled (faster builds)
- ❌ Minification disabled (readable output)

**Use when:**
- Actively developing
- Need fast rebuild times
- Want source maps for debugging
- Need readable compiled output

#### Production Mode (`--mode production`)

Optimized for deployment:

```bash
lunar main.lunar --mode production
```

**Automatically enables:**
- ✅ AST optimizations (`--optimize`)
- ✅ Minification (`--minify`)
- ✅ Incremental compilation cache
- ❌ Source maps disabled (smaller output)

**Use when:**
- Building for deployment
- Want optimized, minified code
- File size matters
- Don't need debugging symbols

### Custom Builds

You can override profile defaults:

```bash
# Production build WITH source maps
lunar main.lunar --mode production --source-map

# Development build WITH optimization
lunar main.lunar --mode dev --optimize
```

---

## Watch Mode Enhancements

### Overview

Watch mode automatically recompiles your code when files change, now with improved debouncing and console management.

### Basic Watch Mode

```bash
lunar main.lunar --watch
```

### New Features

#### 1. Smart Debouncing ✨

**Problem:** Multiple rapid file saves cause multiple unnecessary recompilations.

**Solution:** Lunar now waits 200ms after the last file change before recompiling.

```
Before (old behavior):
File saved → Compile
File saved again → Compile
File saved again → Compile (wasteful!)

After (new behavior):
File saved → Wait 200ms
File saved again → Reset timer, wait 200ms
No more changes → Compile once (efficient!)
```

**Benefits:**
- Fewer unnecessary recompiles
- Less CPU usage
- Faster perceived response time
- Works great with auto-save editors

#### 2. Clear Console Option

Clear the terminal before each rebuild for a clean view:

```bash
lunar main.lunar --watch --watch-clear
```

**Before:**
```
[12:00:01] File changed, recompiling...
[12:00:01] Successfully compiled main.lunar -> main.lua
[12:00:05] File changed, recompiling...
[12:00:05] Successfully compiled main.lunar -> main.lua
[12:00:10] File changed, recompiling...
[12:00:10] Successfully compiled main.lunar -> main.lua
```

**After (`--watch-clear`):**
```
[Clear screen]

[12:00:10] File changed, recompiling...
[12:00:10] Successfully compiled main.lunar -> main.lua
```

#### 3. Configurable Watch Interval

Adjust polling frequency (default: 500ms):

```bash
# Check for changes every 250ms (more responsive)
lunar main.lunar --watch --watch-interval 250

# Check every 1000ms (less CPU usage)
lunar main.lunar --watch --watch-interval 1000
```

### Watch Mode with Build Profiles

Combine watch mode with build profiles:

```bash
# Development watch mode
lunar main.lunar --mode dev --watch --watch-clear

# Production optimization in watch mode
lunar main.lunar --mode production --watch
```

### Bundle Watch Mode

Watch mode automatically tracks dependencies:

```bash
lunar main.lunar --bundle --watch --run
```

**Features:**
- Watches entry file AND all dependencies
- Automatically updates watch list when imports change
- Same smart debouncing as regular watch mode
- Optionally runs the bundled output with `--run`

---

## Source Map Improvements

### Overview

Source maps help you debug compiled Lua by mapping back to original Lunar source code.

### Generating Source Maps

```bash
# Generate .lua.map file
lunar main.lunar --source-map
```

**Output:**
- `main.lua` - Compiled Lua code
- `main.lua.map` - Source map file
- Lua file includes: `--# sourceMappingURL=main.lua.map`

### Source Map Features

#### Position Mapping

Every statement in the generated Lua maps back to its original Lunar location:

```lunar
// main.lunar:10
function greet(name: string): void
    print(`Hello, ${name}!`)
end
```

**Generated with mapping:**
```lua
-- Maps to main.lunar:10
function greet(name)
    print("Hello, " .. tostring(name) .. "!")
end
--# sourceMappingURL=main.lua.map
```

#### Cached with Compilation

Source maps are stored in the incremental compilation cache:

```bash
# First compile: generates source map and caches it
lunar main.lunar --source-map

# Second compile: retrieves from cache (includes source map)
lunar main.lunar --source-map  # ✓ Using cached
```

### Development Workflow

**Recommended setup:**

```bash
# Development: source maps enabled
lunar main.lunar --mode dev --watch --watch-clear

# Production: no source maps
lunar main.lunar --mode production
```

### Debugger Integration

Source maps work with debuggers that support Source Map v3 specification:

1. **VS Code Lua Debugger**: Automatically detects `.lua.map` files
2. **ZeroBrane Studio**: Configure to use source maps
3. **Custom Tools**: Parse `main.lua.map` JSON for position lookups

---

## Optimization Options

### Overview

Lunar provides multiple optimization passes to improve code quality and reduce file size.

### AST Optimizations (`--optimize`)

Performs compile-time optimizations on the Abstract Syntax Tree:

```bash
lunar main.lunar --optimize
```

**Optimizations performed:**

#### 1. Constant Folding

Evaluates constant expressions at compile time:

```lunar
// Before optimization
local result: number = 10 + 20 * 2

// After optimization (in AST)
local result: number = 50
```

#### 2. Dead Code Elimination

Removes unreachable code:

```lunar
// Before optimization
function example(): number
    return 42
    print("This will never run")  // Removed!
    local x = 10                    // Removed!
end

// After optimization
function example(): number
    return 42
end
```

### Minification (`--minify`)

Reduces file size by removing unnecessary characters:

```bash
lunar main.lunar --minify
```

**Transformations:**
- Removes comments
- Removes extra whitespace
- Compresses multi-line code

```lunar
// Before minification (generated Lua)
-- User class definition
local User = {}
User.__index = User

function User.new(name, age)
    local self = setmetatable({}, User)
    self.name = name
    self.age = age
    return self
end

// After minification
local User={} User.__index=User function User.new(name,age) local self=setmetatable({},User) self.name=name self.age=age return self end
```

**File size reduction:** Typically 10-30% smaller

### Incremental Compilation Cache

Speeds up recompilation by caching results:

```bash
# Default: cache enabled
lunar main.lunar

# Disable cache
lunar main.lunar --no-cache

# Clear cache
lunar --clear-cache

# View cache statistics
lunar --cache-stats
```

**Cache features:**
- SHA-256 content hashing for invalidation
- Tracks file dependencies
- Stores compiled code AND source maps
- Automatic cleanup of stale entries

**Performance:**
- Cache hit: ~10ms
- Cache miss: ~50-100ms (full compilation)

---

## Performance Tips

### 1. Use Development Mode for Rapid Iteration

```bash
lunar main.lunar --mode dev --watch --watch-clear
```

- Fast builds (no optimization)
- Source maps for debugging
- Clear console for clean output

### 2. Enable Cache for Repeated Builds

The cache is enabled by default and provides significant speedups:

```bash
# First build: 100ms
lunar main.lunar

# Second build (no changes): 10ms
lunar main.lunar
```

### 3. Optimize Watch Interval

Balance responsiveness vs CPU usage:

```bash
# More responsive (but more CPU)
lunar main.lunar --watch --watch-interval 250

# Less CPU usage (but slower to detect changes)
lunar main.lunar --watch --watch-interval 1000
```

### 4. Use Production Mode for Deployment Builds

```bash
# One-time production build
lunar main.lunar --mode production
```

- Optimized code
- Minified output
- No debug overhead

### 5. Bundle for Faster Load Times

```bash
lunar main.lunar --bundle --mode production
```

- Single file output
- Fewer file I/O operations
- Tree shaking removes unused exports

---

## Examples

### Example 1: Development Workflow

```bash
# Start developing
lunar main.lunar --mode dev --watch --watch-clear

# Output:
# Build mode: Development (fast builds, source maps enabled)
# Watching main.lunar for changes (Ctrl+C to stop)
# [12:00:00] Successfully compiled main.lunar -> main.lua
#
# [12:00:05] File changed, recompiling...
# [12:00:05] Successfully compiled main.lunar -> main.lua
```

### Example 2: Production Build

```bash
# Build for production
lunar main.lunar --mode production --bundle

# Output:
# Build mode: Production (optimized, minified)
# Successfully bundled main.lunar -> main.lua
# Bundle size: 15.2 KB (minified)
```

### Example 3: Testing with Auto-Rerun

```bash
# Watch and automatically run tests
lunar test/all_tests.lunar --mode dev --watch --run

# Output:
# Build mode: Development (fast builds, source maps enabled)
# Watching test/all_tests.lunar for changes (Ctrl+C to stop)
# [12:00:00] Successfully compiled test/all_tests.lunar -> test/all_tests.lua
# [12:00:00] Running test/all_tests.lua...
# ✓ All tests passed (45/45)
#
# [12:00:10] File changed, recompiling...
# [12:00:10] Successfully compiled test/all_tests.lunar -> test/all_tests.lua
# [12:00:10] Running test/all_tests.lua...
# ✓ All tests passed (45/45)
```

### Example 4: Custom Configuration

```bash
# Custom: optimized but keep source maps
lunar main.lunar --optimize --minify --source-map

# Or use production mode and override
lunar main.lunar --mode production --source-map
```

### Example 5: Bundle with Watch and Run

```bash
# Development server workflow
lunar server/main.lunar --mode dev --bundle --watch --run --watch-clear

# Output gets cleared on each change, then shows:
# [12:00:15] File changed, rebundling...
# [12:00:15] Successfully bundled server/main.lunar -> server/main.lua
# [12:00:15] Running server/main.lua...
# Server listening on port 8080...
```

---

## Command Reference

### Build Mode Flags

| Flag | Description | Values |
|------|-------------|--------|
| `--mode` | Set build profile | `dev`, `production` |

### Watch Mode Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--watch` | Enable watch mode | off |
| `--watch-interval` | Polling interval (ms) | 500 |
| `--watch-clear` | Clear console on rebuild | off |

### Optimization Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--optimize` | Enable AST optimizations | off |
| `--minify` | Minify output | off |
| `--source-map` | Generate source maps | off |
| `--no-cache` | Disable cache | off (cache enabled) |

### Cache Management

| Flag | Description |
|------|-------------|
| `--clear-cache` | Clear cache and exit |
| `--cache-stats` | Show cache statistics |

---

## Troubleshooting

### Watch Mode Not Detecting Changes

**Issue:** Files change but watch mode doesn't recompile.

**Solutions:**
1. Check watch interval: `--watch-interval 250` for faster detection
2. Ensure file permissions allow reading
3. Some editors create temp files - save directly to the file

### Cache Not Working

**Issue:** Cache always misses even for unchanged files.

**Solutions:**
1. Check cache stats: `lunar --cache-stats`
2. Clear and rebuild: `lunar --clear-cache && lunar main.lunar`
3. Ensure file paths are consistent (absolute vs relative)

### Slow Watch Mode

**Issue:** Watch mode causes high CPU usage.

**Solutions:**
1. Increase watch interval: `--watch-interval 1000`
2. Smart debouncing (200ms) already helps reduce unnecessary builds
3. Use `--no-cache` only when debugging cache issues

### Clear Console Not Working

**Issue:** `--watch-clear` doesn't clear the terminal.

**Solutions:**
1. Ensure terminal supports ANSI escape codes
2. Works on: Linux, macOS, Windows 10+
3. Some older terminals may not support clearing

---

## Best Practices

### 1. Development

```bash
lunar main.lunar --mode dev --watch --watch-clear
```

- Fast iteration
- Readable output
- Source maps enabled
- Clean console

### 2. CI/CD Builds

```bash
lunar main.lunar --mode production --no-cache
```

- Fresh, reproducible builds
- Optimized output
- No cache dependencies

### 3. Local Production Testing

```bash
lunar main.lunar --mode production --source-map
```

- Optimized like production
- Source maps for debugging
- Test performance-critical code

### 4. Large Projects

```bash
lunar main.lunar --bundle --mode dev --watch
```

- Bundle dependencies
- Fast rebuilds with cache
- Watch all dependencies automatically

---

## Future Enhancements

Planned improvements for future versions:

- [ ] **Build Scripts**: Custom pre/post-build hooks via `lunar.config.json`
- [ ] **Parallel Compilation**: Multi-threaded builds for large projects
- [ ] **Differential Updates**: Only recompile changed modules in bundles
- [ ] **Build Profiles Config**: Define custom profiles in config file
- [ ] **Stack Trace Mapping**: Automatically map Lua errors to Lunar source
- [ ] **Watch Ignore Patterns**: Exclude files from watch mode
- [ ] **Build Metrics**: Detailed timing and size reports

---

## Version History

### v1.2 - Build Tools Update
- ✅ Build profiles (`--mode dev/production`)
- ✅ Smart debouncing in watch mode
- ✅ Clear console option (`--watch-clear`)
- ✅ Improved incremental compilation
- ✅ Source map caching
- ✅ Documentation and examples

### v1.1 - String and Table Enhancements
- Enhanced escape sequences
- Lua-style long strings
- Improved spread operator

### v1.0 - Initial Release
- Basic compilation
- Type checking
- Source maps
- Watch mode
- Bundling

---

## References

- **Source Map Specification**: [Source Map Revision 3](https://sourcemaps.info/spec.html)
- **Lua Optimization**: [Lua Performance Tips](https://www.lua.org/gems/sample.pdf)
- **Incremental Compilation**: Based on content hashing (SHA-256)

---

**Last Updated**: 2025-01-20
**Version**: 1.2
**Author**: Lunar Development Team

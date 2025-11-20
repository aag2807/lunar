# LuaJIT Compilation Hints

Lunar provides powerful JIT compilation hints to optimize your code when running with LuaJIT. These hints help the compiler make better decisions about inlining, loop unrolling, and hot path detection.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Annotation Reference](#annotation-reference)
- [Hot Path Detection](#hot-path-detection)
- [Profiling](#profiling)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

JIT compilation hints are special annotations that guide the LuaJIT compiler to optimize specific code paths. Lunar's JIT hint system provides:

- **Manual annotations** - Direct control over JIT compilation
- **Automatic detection** - Profile-guided optimization
- **Hot path analysis** - Identify frequently executed code
- **Performance profiling** - Track execution time and call counts

## Quick Start

### Enable JIT Hints

```lunar
-- Enable JIT compilation for a function
-- @jit.on
function calculatePi(iterations: number): number
    local sum: number = 0
    for i = 1, iterations do
        sum = sum + (1.0 / (2 * i - 1)) * (i % 2 == 1 ? 1 : -1)
    end
    return sum * 4
end
```

### Mark Hot Paths

```lunar
-- @hotpath
function processImage(pixels: array<number>): array<number>
    local result: array<number> = {}
    -- Image processing code that runs frequently
    for i = 1, #pixels do
        result[i] = pixels[i] * 1.5
    end
    return result
end
```

## Annotation Reference

### @jit.on

Enable JIT compilation for a function or block.

```lunar
-- @jit.on
function fastFunction(): void
    -- This function will be JIT compiled
end
```

### @jit.off

Disable JIT compilation (useful for infrequently called code).

```lunar
-- @jit.off
function initializeConfig(): table<string, any>
    -- Initialization code doesn't benefit from JIT
    return {config = "values"}
end
```

### @hotpath

Mark a function as a hot path (frequently executed).

```lunar
-- @hotpath
function innerLoop(data: array<number>): number
    local sum: number = 0
    for _, value in ipairs(data) do
        sum = sum + value
    end
    return sum
end
```

### @inline

Suggest function inlining.

```lunar
-- @inline
function square(x: number): number
    return x * x
end
```

### @noinline

Prevent function inlining (reduces code size).

```lunar
-- @noinline
function complexCalculation(x: number, y: number): number
    -- Large function that shouldn't be inlined
    return x * y + x / y - x % y
end
```

### @unroll

Suggest loop unrolling for small, fixed iterations.

```lunar
function processVector(vec: array<number>): void
    -- @unroll
    for i = 1, 4 do
        vec[i] = vec[i] * 2
    end
end
```

### @optimize

Request aggressive optimization for a function.

```lunar
-- @optimize level=3
function criticalPath(data: array<number>): number
    -- Performance-critical code
    local result: number = 0
    for _, value in ipairs(data) do
        result = result + value * value
    end
    return result
end
```

### @jit.flush

Flush JIT cache after function execution.

```lunar
-- @jit.flush
function dynamicCodeLoader(): void
    -- Code that generates new functions
end
```

## Hot Path Detection

Lunar can automatically detect hot paths by profiling your application.

### Enable Profiling

```bash
lunar --jit-profile myapp.lunar
```

### Profiling Output

```
=== JIT Profiling Report ===
calculatePi: 2.456000 seconds (1000 calls, 0.002456 sec/call)
processImage: 5.123000 seconds (5000 calls, 0.001025 sec/call)
innerLoop: 8.234000 seconds (10000 calls, 0.000823 sec/call)
============================
```

### Apply Automatic Optimizations

```bash
# Generate optimized version with JIT hints
lunar --jit-optimize --profile-data=profile.json myapp.lunar
```

## Profiling

### Instrument Code for Profiling

```lunar
-- Enable profiling instrumentation
-- @jit-profile

function businessLogic(data: array<any>): void
    -- Your code here
    processData(data)
    validateResults()
    saveToDatabase()
end
```

### Profiling Configuration

```json
{
  "jit": {
    "profiling": {
      "enabled": true,
      "outputFile": "jit_profile.json",
      "threshold": 1000,
      "includeCallCounts": true,
      "includeTimings": true
    }
  }
}
```

### Reading Profile Data

```bash
# View profiling report
lunar --jit-report profile.json

# Hot paths (>1000 executions)
Function                 Calls      Time (ms)   Avg (ms)
processImage             5000       5123        1.025
innerLoop                10000      8234        0.823
calculateSum             2500       1234        0.494
```

## Best Practices

### 1. Profile Before Optimizing

```lunar
-- ❌ Don't blindly add annotations
-- @jit.on
function maybeSlowFunction(): void
    -- ...
end

-- ✓ Profile first, then optimize hot paths
-- Run profiler: lunar --jit-profile app.lunar
-- Review results, then add hints to actual hot paths
```

### 2. Use Annotations Sparingly

```lunar
-- ❌ Too many annotations
-- @jit.on
-- @hotpath
-- @inline
-- @optimize level=3
function simpleGetter(): number
    return self.value
end

-- ✓ Simple code doesn't need hints
function simpleGetter(): number
    return self.value
end
```

### 3. Mark True Hot Paths

```lunar
-- ✓ Good use of @hotpath
-- @hotpath
function renderFrame(objects: array<GameObject>): void
    -- Called 60 times per second
    for _, obj in ipairs(objects) do
        obj:update()
        obj:render()
    end
end
```

### 4. Disable JIT for Init Code

```lunar
-- ✓ Disable JIT for one-time initialization
-- @jit.off
function loadGameAssets(): void
    -- Called once at startup
    loadTextures()
    loadSounds()
    loadLevels()
end
```

### 5. Optimize Numerical Code

```lunar
-- @optimize level=3
function matrixMultiply(a: array<array<number>>, b: array<array<number>>): array<array<number>>
    -- Numerical operations benefit greatly from JIT
    local result: array<array<number>> = {}
    for i = 1, #a do
        result[i] = {}
        for j = 1, #b[1] do
            result[i][j] = 0
            for k = 1, #b do
                result[i][j] = result[i][j] + a[i][k] * b[k][j]
            end
        end
    end
    return result
end
```

## Examples

### Example 1: Game Loop Optimization

```lunar
class Game
    -- @hotpath - Runs 60 times per second
    function update(dt: number): void
        self:updatePhysics(dt)
        self:updateAI(dt)
        self:updateAnimations(dt)
    end

    -- @hotpath
    function render(): void
        self:renderBackground()
        self:renderEntities()
        self:renderUI()
    end

    -- @jit.off - Called rarely
    function loadLevel(levelName: string): void
        local data = loadLevelData(levelName)
        self:initializeLevel(data)
    end
end
```

### Example 2: Data Processing Pipeline

```lunar
-- @optimize level=3
function processDataBatch(batch: array<Record>): array<ProcessedRecord>
    local results: array<ProcessedRecord> = {}

    -- @hotpath - Inner loop executed millions of times
    for i, record in ipairs(batch) do
        results[i] = {
            id = record.id,
            value = transformValue(record.value),
            timestamp = os.time()
        }
    end

    return results
end

-- @inline - Small helper function
function transformValue(value: number): number
    return value * 1.5 + 10
end
```

### Example 3: Image Processing

```lunar
-- @optimize level=3
function applyFilter(image: Image, filter: Filter): Image
    local width: number = image.width
    local height: number = image.height
    local result: Image = createImage(width, height)

    -- @hotpath
    -- @unroll - Process 4 pixels at a time
    for y = 1, height do
        for x = 1, width do
            local pixel: Pixel = image:getPixel(x, y)
            result:setPixel(x, y, filter:apply(pixel))
        end
    end

    return result
end
```

### Example 4: With Profiling

```lunar
-- main.lunar

-- Enable automatic profiling
-- @jit-profile

function main(): void
    local data: array<number> = generateTestData(1000000)

    -- This will be profiled
    local result1 = algorithm1(data)
    local result2 = algorithm2(data)
    local result3 = algorithm3(data)

    print("Results:", result1, result2, result3)
end

function algorithm1(data: array<number>): number
    local sum: number = 0
    for _, value in ipairs(data) do
        sum = sum + value
    end
    return sum
end

function algorithm2(data: array<number>): number
    local sum: number = 0
    for i = 1, #data do
        sum = sum + data[i] * data[i]
    end
    return sum
end

function algorithm3(data: array<number>): number
    local sum: number = 0
    for i = 1, #data, 2 do
        sum = sum + data[i]
    end
    return sum
end

main()
```

Run with profiling:

```bash
# Profile the code
lunar --jit-profile main.lunar

# View results
lunar --jit-report jit_profile.json

# Apply automatic optimization hints
lunar --jit-optimize --profile-data=jit_profile.json main.lunar
```

## Advanced Configuration

### LuaJIT Options

You can configure LuaJIT parameters in your build:

```json
{
  "jit": {
    "enabled": true,
    "options": {
      "maxtrace": 10000,
      "maxrecord": 40000,
      "maxirconst": 10000,
      "loopunroll": 40,
      "instunroll": 4,
      "callunroll": 3,
      "recunroll": 2
    }
  }
}
```

### Trace Analysis

```bash
# Enable trace dumping
lunar --jit-trace myapp.lunar

# Output: jit_traces.txt
[TRACE   1] myapp.lunar:15 loop
[TRACE   2] myapp.lunar:23 return
[TRACE   3 abort] myapp.lunar:45 NYI: unsupported instruction
```

## Performance Tips

1. **Numerical code benefits most** - JIT excels at number-heavy computations
2. **Avoid mixed types** - Keep operations type-stable
3. **Minimize table allocations** - Reuse tables when possible
4. **Use local variables** - Faster than globals
5. **Profile before optimizing** - Let data guide decisions

## Troubleshooting

### Trace Aborts

If you see "trace abort" messages:

```lunar
-- ❌ Causes trace abort (mixing types)
function badFunction(x: any): any
    if type(x) == "number" then
        return x * 2
    else
        return tostring(x)
    end
end

-- ✓ Type-stable function
function goodFunction(x: number): number
    return x * 2
end
```

### Performance Regression

If JIT makes code slower:

```lunar
-- @jit.off - Disable JIT for this function
function stringHeavyFunction(strings: array<string>): string
    -- String operations may not benefit from JIT
    return table.concat(strings, ", ")
end
```

## Further Reading

- [LuaJIT Performance Guide](http://wiki.luajit.org/Numerical-Computing-Performance-Guide)
- [LuaJIT NYI (Not Yet Implemented)](http://wiki.luajit.org/NYI)
- [Lunar Performance Guide](./PERFORMANCE.md)

## Summary

JIT compilation hints are a powerful tool for optimizing Lunar code. Key takeaways:

- ✓ Profile first, optimize hot paths
- ✓ Use annotations sparingly
- ✓ Numerical code benefits most
- ✓ Let profiling data guide decisions
- ✓ Disable JIT for init/string-heavy code

Happy optimizing! 🚀

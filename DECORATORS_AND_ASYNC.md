# Decorators and Async/Await in Lunar

## ✅ Both Features Fully Implemented!

Both Decorators and Async/Await are **fully implemented and working** in Lunar. These features bring TypeScript-like functionality to Lunar.

---

## Decorators

Decorators provide a way to add metadata and modify the behavior of functions and classes at declaration time.

### Syntax

```lunar
@decoratorName
function myFunction(): void
end

@decoratorWithArgs("argument")
function myFunction(): void
end

@decorator1
@decorator2
function myFunction(): void
end
```

### How It Works

Decorators are syntactic sugar that transforms into function calls:

```lunar
@log
function greet(name: string): string
    return "Hello, " .. name
end
```

Compiles to:

```lua
function greet(name)
    return "Hello, " .. name
end
greet = log(greet)
```

### Features

#### ✅ Simple Decorators

```lunar
function log(fn: any): any
    print("Function decorated")
    return fn
end

@log
function myFunction(): void
    print("Hello")
end
```

#### ✅ Decorators with Arguments

```lunar
function memoize(cacheSize: number): any
    return function(fn: any): any
        print("Memoizing with cache size:", cacheSize)
        return fn
    end
end

@memoize(100)
function expensive(n: number): number
    return n * 2
end
```

Compiles to:
```lua
function expensive(n)
    return n * 2
end
expensive = memoize(100)(expensive)
```

#### ✅ Multiple Decorators

Decorators are applied **bottom-to-top**:

```lunar
@decorator1
@decorator2
@decorator3
function myFunction(): void
end
```

Compiles to:
```lua
function myFunction()
end
myFunction = decorator1(decorator2(decorator3(myFunction)))
```

#### ✅ Class Decorators

```lunar
function component(cls: any): any
    print("Component registered")
    return cls
end

@component
class Button
    label: string
end
```

Compiles to:
```lua
local Button = {}
-- class implementation
Button = component(Button)
```

### Common Use Cases

1. **Logging**: Track function calls
2. **Memoization**: Cache function results
3. **Deprecation Warnings**: Mark old APIs
4. **Authorization**: Check permissions before execution
5. **Rate Limiting**: Control function call frequency
6. **Dependency Injection**: Register services

### Examples

See `examples/decorators.lunar` for comprehensive examples including:
- Simple function decorators
- Decorators with arguments
- Multiple decorators
- Class decorators
- Validation decorators
- Timing decorators

---

## Async/Await

Async/await provides a clean syntax for working with asynchronous operations using Lua coroutines.

### Syntax

```lunar
async function fetchData(): string
    return "data"
end

async function processData(): string
    local data = await fetchData()
    return data
end
```

### How It Works

Async functions return Lua coroutines, and await uses coroutine.yield:

```lunar
async function fetchData(): string
    print("Fetching...")
    return "data"
end

async function processData(): string
    local result = await fetchData()
    return result
end
```

Compiles to:

```lua
function fetchData()
    return coroutine.create(function()
        print("Fetching...")
        return "data"
    end)
end

function processData()
    return coroutine.create(function()
        local result = coroutine.yield(fetchData())
        return result
    end)
end
```

### Features

#### ✅ Async Functions

Any function can be marked as async:

```lunar
async function fetchUser(id: number): string
    return "User: " .. tostring(id)
end
```

Returns a coroutine that can be resumed.

#### ✅ Await Expressions

Await suspends execution:

```lunar
async function loadData(): string
    local user = await fetchUser(123)
    local posts = await fetchPosts(123)
    return user .. " | " .. posts
end
```

#### ✅ Await in Conditionals

```lunar
async function getData(valid: boolean): string
    if valid then
        local data = await fetchData()
        return data
    else
        return "invalid"
    end
end
```

#### ✅ Await in Loops

```lunar
async function loadAll(count: number): string
    local results: string = ""
    for i = 1, count do
        local data = await fetchItem(i)
        results = results .. data
    end
    return results
end
```

#### ✅ Async with Decorators

Decorators work perfectly with async functions:

```lunar
function cache(fn: any): any
    return fn
end

@cache
async function fetchData(): string
    return "cached data"
end
```

### Running Async Functions

To execute an async function, you need to resume the coroutine:

```lunar
-- Simple execution
local co = fetchData()
local success, result = coroutine.resume(co)
print(result)

-- Or create an async runtime helper:
function runAsync(asyncFn: any): void
    local co = asyncFn()
    local success = true
    local result = nil

    while success do
        success, result = coroutine.resume(co, result)
        if not success or coroutine.status(co) == "dead" then
            break
        end
    end
end
```

### Common Use Cases

1. **API Calls**: HTTP requests
2. **Database Queries**: Async database operations
3. **File I/O**: Reading/writing files asynchronously
4. **Data Pipelines**: Sequential async operations
5. **Concurrent Operations**: Managing multiple async tasks
6. **Event Handling**: Async event processors

### Examples

See `examples/async_await.lunar` for comprehensive examples including:
- Simple async functions
- Multiple await expressions
- Async with conditionals and loops
- Error handling patterns
- Data pipeline patterns
- Practical async runtime helper

---

## Implementation Status

| Feature | Status | Notes |
|---------|--------|-------|
| **Decorators** | ✅ | Fully implemented |
| - Simple decorators | ✅ | Working |
| - Decorators with arguments | ✅ | Working |
| - Multiple decorators | ✅ | Working |
| - Class decorators | ✅ | Working |
| **Async/Await** | ✅ | Fully implemented |
| - Async functions | ✅ | Returns coroutines |
| - Await expressions | ✅ | Uses coroutine.yield |
| - Await in conditionals | ✅ | Working |
| - Await in loops | ✅ | Working |
| - Async + Decorators | ✅ | Working together |

---

## Tests

Both features have comprehensive test suites:

- **Decorators**: `internal/types/decorator_test.go` (8 tests, all passing)
- **Async/Await**: `internal/types/async_test.go` (6 tests, all passing)

Run tests:
```bash
go test ./internal/types -v -run Decorator
go test ./internal/types -v -run Async
```

---

## Quick Reference

### Decorator Patterns

```lunar
// Simple
@log
function foo(): void end

// With args
@memoize(100)
function bar(): void end

// Multiple
@auth
@rate
function baz(): void end

// Class
@component
class MyClass end
```

### Async/Await Patterns

```lunar
// Simple async
async function fetch(): string
    return "data"
end

// With await
async function process(): string
    local data = await fetch()
    return data
end

// Multiple awaits
async function loadAll(): string
    local a = await fetch1()
    local b = await fetch2()
    return a .. b
end

// With conditions
async function check(): string
    if condition then
        return await fetchData()
    end
end
```

---

## Next Steps

Both features are production-ready! You can:

1. Use decorators for:
   - Function wrapping and enhancement
   - Class registration and metadata
   - Authorization and validation

2. Use async/await for:
   - Asynchronous operations
   - Coroutine-based concurrency
   - Sequential async workflows

See the example files for more patterns and use cases!

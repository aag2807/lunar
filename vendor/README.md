# Lunar Vendor Libraries

Built-in libraries for Lunar, inspired by Odin's vendor system. These libraries are automatically available for import and will be bundled with your application.

## Usage

Import vendor libraries using the `vendor/` prefix:

```lunar
import { printf, println } from "vendor/fmt"
import { encode, decode } from "vendor/json"
import { get, post } from "vendor/http"
import { describe, it, expect } from "vendor/testing"
```

## Available Libraries

### fmt - Formatting & Output

Print and format functions for console output.

```lunar
import { printf, println, inspect } from "vendor/fmt"

printf("Hello %s!\n", "World")
println("Multiple", "arguments")

local data = { a = 1, b = 2 }
println(inspect(data))  -- { a = 1, b = 2 }
```

**Functions:**
- `printf(format, ...)` - Print formatted string to stdout
- `sprintf(format, ...) -> string` - Return formatted string
- `println(...)` - Print with newline
- `print(...)` - Print without newline
- `inspect(value, depth?) -> string` - Format value for debugging
- `errorf(format, ...)` - Print to stderr

### json - JSON Encoding/Decoding

Pure Lua JSON library for encoding and decoding.

```lunar
import { encode, decode } from "vendor/json"

local data = { name = "Lunar", version = 1 }
local jsonStr = encode(data)
-- {"name":"Lunar","version":1}

local parsed = decode('{"key":"value"}')
println(parsed.key)  -- value
```

**Functions:**
- `encode(value) -> string` - Encode value to JSON string
- `decode(str) -> any` - Decode JSON string to value

### http - HTTP Client

HTTP client built on LuaSocket. Requires `luarocks install luasocket`.

```lunar
import { get, post } from "vendor/http"

local response = get("https://api.example.com/data")
println(response.status)  -- 200
println(response.body)

local response = post("https://api.example.com/submit", "data=value")
```

**Functions:**
- `get(url, options?) -> Response` - GET request
- `post(url, body, options?) -> Response` - POST request
- `put(url, body, options?) -> Response` - PUT request
- `delete(url, options?) -> Response` - DELETE request
- `request(method, url, options?) -> Response` - Generic request
- `encodeURI(str) -> string` - URL encode
- `decodeURI(str) -> string` - URL decode
- `parseURL(url) -> table` - Parse URL components

**Response Object:**
```lunar
{
    status: number,
    body: string,
    headers: table
}
```

### testing - Test Framework

BDD-style testing framework with assertions.

```lunar
import { describe, it, expect, assert } from "vendor/testing"

describe("My Feature", function()
    it("should work correctly", function()
        local result = 1 + 1
        expect(result).toBe(2)
        expect(result).toBeGreaterThan(1)
    end)
end)
```

**Test Structure:**
- `describe(name, fn)` - Define a test suite
- `it(name, fn)` - Define a test case
- `beforeEach(fn)` - Run before each test
- `afterEach(fn)` - Run after each test

**Assertions:**
- `expect(value).toBe(expected)` - Strict equality
- `expect(value).toEqual(expected)` - Deep equality
- `expect(value).toBeTruthy()` - Truthy check
- `expect(value).toBeFalsy()` - Falsy check
- `expect(value).toBeNil()` - Nil check
- `expect(value).toBeType(type)` - Type check
- `expect(value).toContain(substr)` - String contains
- `expect(value).toBeGreaterThan(n)` - Number comparison
- `expect(value).toBeLessThan(n)` - Number comparison

**Simple Assertions:**
- `assert.equal(a, b, msg?)` - Assert equality
- `assert.notEqual(a, b, msg?)` - Assert inequality
- `assert.isTrue(value, msg?)` - Assert true
- `assert.isFalse(value, msg?)` - Assert false
- `assert.isNil(value, msg?)` - Assert nil
- `assert.isNotNil(value, msg?)` - Assert not nil
- `assert.fail(msg)` - Force failure

## Adding Custom Vendor Libraries

To add your own vendor library:

1. Create a directory under `vendor/` (e.g., `vendor/mylib/`)
2. Add a `.lua` implementation file (e.g., `mylib.lua`)
3. Add a `.d.lunar` declaration file for type definitions (e.g., `mylib.d.lunar`)
4. The library should return a table of exported functions

Example structure:
```
vendor/
  mylib/
    mylib.lua        -- Implementation
    mylib.d.lunar    -- Type declarations
```

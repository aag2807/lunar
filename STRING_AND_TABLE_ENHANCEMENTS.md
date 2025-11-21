# String and Table Enhancements in Lunar

This document describes the enhanced string features and improved spread operator support added to Lunar.

---

## String Enhancements

### 1. Enhanced Escape Sequences ✅

Lunar now supports a comprehensive set of escape sequences in string literals and template strings.

#### Basic Escape Sequences (Already Supported)
```lunar
local str1: string = "Hello\nWorld"  -- Newline
local str2: string = "Tab\there"      -- Tab
local str3: string = "Quote: \"Hi\"" -- Escaped quotes
local str4: string = 'Single\'s'      -- Single quote
local str5: string = "Back\\slash"    -- Backslash
```

#### New Escape Sequences ✅

**Control Characters:**
```lunar
local cr: string = "Line1\rLine2"        -- \r = Carriage return
local bs: string = "ABC\bD"              -- \b = Backspace
local ff: string = "Page1\fPage2"        -- \f = Form feed
local vt: string = "Line1\vLine2"        -- \v = Vertical tab
local null: string = "Text\0More"        -- \0 = Null character
```

**Hexadecimal Escapes:**
```lunar
-- \xHH format (2 hex digits)
local hex1: string = "\x48\x65\x6C\x6C\x6F"  -- "Hello"
local hex2: string = "\x41\x42\x43"          -- "ABC"
```

**Unicode Escapes:**
```lunar
-- \uHHHH format (4 hex digits)
local unicode1: string = "\u0048\u0065\u006C\u006C\u006F"  -- "Hello"
local unicode2: string = "\u03B1\u03B2\u03B3"              -- "αβγ" (Greek)

-- \u{HHHHHH} format (up to 6 hex digits, for any Unicode code point)
local emoji: string = "\u{1F4A9}"           -- 💩
local smiley: string = "\u{1F600}"          -- 😀
local heart: string = "\u{2764}"            -- ❤
local mix: string = "Hello \u{1F44B}"       -- "Hello 👋"
```

#### Works in All String Types

All escape sequences work in:
- Double-quoted strings: `"Hello\n"`
- Single-quoted strings: `'Hello\n'`
- Template strings: `` `Hello\n${name}` ``

```lunar
local name: string = "World"

-- All of these work:
local str1: string = "Hello\nWorld"
local str2: string = 'Hello\nWorld'
local str3: string = `Hello\n${name}`
local str4: string = `Unicode: \u{1F680} ${name}`
```

---

### 2. Lua-Style Long Strings ✅

Long strings provide a way to include multi-line text without processing escape sequences.

#### Basic Long String
```lunar
local long: string = [[
This is a long string
that spans multiple lines
without any escape sequences
\n \t \r are NOT processed
]]
print(long)
```

#### Long Strings with Nested Brackets

If you need to include `]]` in your string, use equals signs:

```lunar
-- Level 0: [[...]]
local basic: string = [[Simple long string]]

-- Level 1: [=[...]=]
local level1: string = [=[
This can contain [[brackets]] inside
]=]

-- Level 2: [==[...]==]
local level2: string = [==[
This can contain [=[ nested ]=] brackets
]==]

-- Any number of equals signs
local level3: string = [===[
Even more [[[ nesting ]]]
]===]
```

#### Key Features of Long Strings

**No Escape Processing:**
```lunar
local raw: string = [[
Backslash-n: \n
Backslash-t: \t
These are literal characters, not escapes!
]]
```

**Multi-line Without Concatenation:**
```lunar
-- Before: needed concatenation
local old: string = "Line 1\n" ..
                    "Line 2\n" ..
                    "Line 3"

-- Now: use long strings
local new: string = [[
Line 1
Line 2
Line 3
]]
```

**Embedding Code Samples:**
```lunar
local luaCode: string = [[
function hello()
    print("Hello, World!")
end
]]

local json: string = [[
{
    "name": "Lunar",
    "version": "1.0"
}
]]
```

---

### 3. Combining Features

You can combine long strings with template interpolation:

```lunar
local code: string = [[
function greet(name)
    print("Hello, " .. name)
end
]]

local message: string = `Code sample:\n${code}`
print(message)
```

---

## Table Enhancements

### Spread Operator for Arrays ✅

The spread operator (`...`) now fully supports array literals, allowing you to merge and expand arrays inline.

#### Basic Array Spreading

```lunar
local arr1: number[] = {1, 2, 3}
local arr2: number[] = {4, 5, 6}

-- Merge arrays
local merged: number[] = {...arr1, ...arr2}
-- Result: {1, 2, 3, 4, 5, 6}
```

#### Mixing Spread with Literal Elements

```lunar
local middle: number[] = {4, 5, 6}

-- Add elements before and after
local expanded: number[] = {1, 2, 3, ...middle, 7, 8, 9}
-- Result: {1, 2, 3, 4, 5, 6, 7, 8, 9}
```

#### Multiple Spreads

```lunar
local a: number[] = {1, 2}
local b: number[] = {3, 4}
local c: number[] = {5, 6}

local all: number[] = {...a, ...b, ...c}
-- Result: {1, 2, 3, 4, 5, 6}
```

#### Empty Array Spread

```lunar
local empty: number[] = {}
local nums: number[] = {...empty, 1, 2, 3}
-- Result: {1, 2, 3}
```

#### Nested Spreading

```lunar
local inner: number[] = {3, 4}
local outer: number[] = {1, 2, ...inner, 5, 6}
-- Result: {1, 2, 3, 4, 5, 6}
```

---

### Spread Operator for Tables (Experimental) ⚠️

**Status**: Code generation is implemented, but type checker support is pending.

#### What Works (Without Type Annotations)

```lunar
-- Object merging
local obj1 = {name = "Alice", age = 30}
local obj2 = {city = "NYC", country = "USA"}
local merged = {...obj1, ...obj2}
-- Result: {name = "Alice", age = 30, city = "NYC", country = "USA"}

-- Override properties
local base = {x = 10, y = 20, z = 30}
local override = {...base, y = 999}
-- Result: {x = 10, y = 999, z = 30}

-- Multiple spreads with explicit keys
local part1 = {a = "A", b = "B"}
local part2 = {c = "C", d = "D"}
local combined = {...part1, key = "value", ...part2}
-- Result: {a = "A", b = "B", key = "value", c = "C", d = "D"}
```

#### Current Limitations

The type checker doesn't yet support inferring table types from spread expressions. You can use table spreading if you:

1. Don't use explicit type annotations
2. Use `any` type: `local merged: any = {...obj1, ...obj2}`
3. Run without full type checking (when available)

**Type checker support for table spreading is planned for a future update.**

---

## Generated Lua Code

### How Escape Sequences Are Compiled

Lunar processes escape sequences at compile time and properly escapes them in the generated Lua output:

```lunar
-- Input:
local str: string = "Hello\nWorld\t\u{1F600}"

-- Generated Lua:
local str = "Hello\nWorld\t😀"
```

### How Long Strings Are Compiled

Long strings are preserved as-is in the Lua output:

```lunar
-- Input:
local code: string = [[
function test()
    return 42
end
]]

-- Generated Lua:
local code = "\nfunction test()\n    return 42\nend\n"
```

### How Array Spread Is Compiled

```lunar
-- Input:
local merged: number[] = {1, ...arr, 2}

-- Generated Lua:
local merged = (function()
    local __temp = {}
    table.insert(__temp, 1)
    for _, __v in ipairs(arr) do
        table.insert(__temp, __v)
    end
    table.insert(__temp, 2)
    return __temp
end)()
```

### How Table Spread Is Compiled

```lunar
-- Input:
local merged = {...obj1, key = "val", ...obj2}

-- Generated Lua:
local merged = (function()
    local __temp = {}
    for __k, __v in pairs(obj1) do
        __temp[__k] = __v
    end
    __temp["key"] = "val"
    for __k, __v in pairs(obj2) do
        __temp[__k] = __v
    end
    return __temp
end)()
```

---

## Examples

### Example 1: Multi-language String Support

```lunar
local greeting: string = "\u{1F44B} Hello! \u{4F60}\u{597D}!"
print(greeting)
-- Output: 👋 Hello! 你好!
```

### Example 2: Embedded SQL Query

```lunar
local tableName: string = "users"
local query: string = [[
SELECT id, name, email
FROM users
WHERE active = 1
ORDER BY created_at DESC
LIMIT 100
]]

local fullQuery: string = `Query for ${tableName}:\n${query}`
print(fullQuery)
```

### Example 3: Array Utilities

```lunar
function flatten<T>(arrays: T[][]): T[]
    local result: T[] = {}
    for _, arr in ipairs(arrays) do
        result = {...result, ...arr}
    end
    return result
end

local arrays: number[][] = {{1, 2}, {3, 4}, {5, 6}}
local flat: number[] = flatten(arrays)
-- Result: {1, 2, 3, 4, 5, 6}
```

### Example 4: Configuration Merging

```lunar
local defaults = {
    timeout = 30,
    retries = 3,
    debug = false
}

local userConfig = {
    timeout = 60,
    debug = true
}

-- Merge with user config overriding defaults
local config = {...defaults, ...userConfig}
-- Result: {timeout = 60, retries = 3, debug = true}
```

---

## Summary

### String Enhancements ✅

| Feature | Status | Example |
|---------|--------|---------|
| Enhanced escape sequences | ✅ Implemented | `"\r\b\f\v\0"` |
| Hexadecimal escapes | ✅ Implemented | `"\x48\x65\x6C\x6C\x6F"` |
| Unicode escapes (4-digit) | ✅ Implemented | `"\u0048\u0065"` |
| Unicode escapes (variable) | ✅ Implemented | `"\u{1F600}"` |
| Lua-style long strings | ✅ Implemented | `[[...]]` |
| Nested bracket levels | ✅ Implemented | `[=[...]=]` |

### Table Enhancements

| Feature | Status | Example |
|---------|--------|---------|
| Array spread | ✅ Fully Implemented | `{...arr1, ...arr2}` |
| Table spread (codegen) | ✅ Implemented | `{...obj1, key = val}` |
| Table spread (types) | ⚠️ Pending | Type checker needs updates |

---

## Testing

Run the test files to see these features in action:

```bash
# Test string enhancements
lunar test/string_enhancements.lunar
lua test/string_enhancements.lua

# Test array spreading
lunar test/array_spread.lunar
lua test/array_spread.lua
```

---

## Future Enhancements

- **Type checker support for table spread**: Full type inference for spread in table contexts
- **Spread in function calls**: Already supported, `func(...args)`
- **Rest parameters**: Already supported, `function fn(...args: string[])`
- **Spread in other contexts**: Potential future expansion

---

## References

- **Lexer Implementation**: `internal/lexer/lexer.go` (lines 270-475)
- **Escape Sequence Parsing**: `internal/lexer/lexer.go` (lines 462-507)
- **Long String Parsing**: `internal/lexer/lexer.go` (lines 414-476)
- **Code Generator**: `internal/codegen/generator.go` (lines 985-1054, 1512-1540)
- **Spread in Tables**: `internal/codegen/generator.go` (lines 995-1054)

---

## Migration Guide

If you're upgrading from an older version of Lunar:

1. **Escape sequences**: No changes needed, all existing code remains compatible
2. **Long strings**: New feature, opt-in usage
3. **Array spread**: If you were working around lack of spread support, you can now simplify your code
4. **Table spread**: Use with caution until type checker support is added

---

**Version**: Added in Lunar v1.1
**Author**: Lunar Development Team
**Last Updated**: 2025-01-20

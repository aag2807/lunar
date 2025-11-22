

# Lunar Standard Library Declarations

This directory contains type declarations for Lua 5.1's standard library. These files provide type safety for all built-in Lua functions and modules.

## Available Libraries

### Lua Standard Library

- **lua.d.lunar** - Core global functions (print, tonumber, etc.) ✓ Working
- **math.d.lunar** - Mathematical functions (math.sin, math.random, etc.) ✓ Working
- **io.d.lunar** - File I/O operations (io.open, io.read, etc.) ✓ Working
- **os.d.lunar** - OS facilities (os.time, os.execute, etc.) ✓ Working
- **string.d.lunar** - String manipulation functions (string.sub, string.upper, etc.) ✓ Working
- **table.d.lunar** - Table manipulation functions (table.insert, table.concat, etc.) ✓ Working
- **coroutine.d.lunar** - Coroutine functions (coroutine.create, coroutine.resume, etc.) ✓ Working
- **debug.d.lunar** - Debug introspection functions (debug.getinfo, debug.traceback, etc.) ✓ Working
- **package.d.lunar** - Module loading and package management (require, package.loaded, etc.) ✓ Working

### Vendor Libraries (LuaRocks)

- **lfs.d.lunar** - LuaFileSystem - File system operations (directory iteration, attributes, etc.)
- **crypto.d.lunar** - LuaCrypto - Cryptographic functions (hashing, encryption, HMAC, etc.)
- **date.d.lunar** - LuaDate - Advanced date/time utilities (parsing, formatting, arithmetic)
- **lpeg.d.lunar** - LPeg - Parsing Expression Grammars (pattern matching and parsing)
- **rex_pcre.d.lunar** - Rex PCRE - Perl Compatible Regular Expressions

## Context-Aware Keywords

Lunar now supports **context-aware keywords** for `string`, `table`, and `type`. These keywords work as:
- **Type names** in type annotation contexts: `local x: string = "hello"`
- **Identifiers** in value contexts: `string.len("hello")` or `local myString = "test"`

This means you get full type safety for both:
- Primitive types (`string`, `number`, `boolean`, etc.)
- Standard library modules (`string.*`, `table.*` functions)

All standard library declarations are now fully functional!

## Usage

### Option 1: Copy to Your Project (Recommended)

Copy the declaration files you need into your project directory:

```bash
cp stdlib/*.d.lunar my_project/
cd my_project
lunar my_code.lunar  # Declarations auto-loaded!
```

### Option 2: Use from stdlib Directory

Place your code in the stdlib directory (or create a symlink):

```bash
cd lunar/stdlib
lunar ../my_code.lunar  # Will find stdlib declarations
```

### Option 3: Global Installation (Advanced)

Create a global declarations directory and configure your environment to always include it.

## Examples

### Using Math Library (Working)

```lunar
-- math.d.lunar provides types automatically
local angle: number = math.pi / 4
local sine: number = math.sin(angle)
local rounded: number = math.floor(3.7)

-- Constants are typed too!
local pi: number = math.pi  -- ✓
-- local wrong: string = math.pi  -- ✗ Type error!
```

### Using I/O Library (Working)

```lunar
-- io.d.lunar provides types automatically
local file: File | nil = io.open("data.txt", "r")
if file ~= nil then
    local content: string = file:read("*all")
    file:close()
end
```

### Using String Library (Now Working!)

```lunar
-- string.d.lunar provides types automatically
local str: string = "Hello, World!"
local len: number = string.len(str)
local upper: string = string.upper(str)
local sub: string = string.sub(str, 1, 5)

-- Context-aware: 'string' works both as a type and module name!
```

### Using Table Library (Now Working!)

```lunar
-- table.d.lunar provides types automatically
local tbl: any = {1, 2, 3}
table.insert(tbl, 4)
table.sort(tbl)
local result: string = table.concat(tbl, ", ")

-- Context-aware: 'table' works both as a type and module name!
```

## Coverage

These declarations cover the most commonly used functions from Lua 5.1's standard library. They provide:

- **Type safety** for function parameters and return values
- **IntelliSense-style** code completion hints
- **Compile-time error checking** for API misuse
- **Self-documenting** code with clear type signatures

## Compatibility

- **Lua 5.1** - Full coverage
- **Lua 5.2/5.3** - Most functions work, some newer features not included
- **LuaJIT** - Compatible with LuaJIT 2.x standard library

## LuaRocks Integration

Lunar now supports automatic installation and type generation for LuaRocks packages!

### Quick Start

```bash
# Initialize LuaRocks configuration
lunar --rocks-init

# Install a package and auto-generate types
lunar --rocks-install lfs@1.8.0

# List installed packages
lunar --rocks-list

# Search for packages
lunar --rocks-search "json"

# Generate types for an already installed package
lunar --rocks-types luasocket
```

### Using Vendor Libraries

Once installed, vendor libraries work just like standard library modules:

```lunar
-- File system operations with LuaFileSystem
local lfs = require("lfs")
local attr: FileAttributes = lfs.attributes("/path/to/file")
print("File size:", attr.size)

-- Iterate directory
for file in lfs.dir(".") do
    print(file)
end
```

```lunar
-- Cryptography with LuaCrypto
local crypto = require("crypto")
local hash: string = crypto.digest("sha256", "Hello, World!")
local hmac: string = crypto.hmac("sha256", "secret_key", "message")
```

```lunar
-- Date manipulation with LuaDate
local date = require("date")
local now: Date = date.now()
local tomorrow: Date = now:adddays(1)
print("Tomorrow:", tomorrow:fmt("%Y-%m-%d"))
```

```lunar
-- Regular expressions with Rex PCRE
local rex = require("rex_pcre")
local match_start, match_end = rex.find("hello world", "w\\w+")
print("Found:", match_start, match_end)
```

### Managing Dependencies

Create a `lunarocks.json` file to manage project dependencies:

```json
{
  "dependencies": {
    "lfs": "1.8.0",
    "luasocket": "3.0.0",
    "luasec": "1.0.0"
  },
  "devDependencies": {
    "busted": "2.0.0"
  }
}
```

Then install all dependencies:

```bash
lunar --rocks-deps
```

This will:
1. Install all packages from `lunarocks.json`
2. Auto-generate `.d.lunar` type definition files
3. Place type definitions in the `./types` directory

## Extending

To add your own functions or modules:

1. Create a new `.d.lunar` file
2. Use `declare interface` for modules
3. Use `declare function` for global functions
4. Place in your project directory

Example:

```lunar
-- mylib.d.lunar
declare interface MyLib
    doSomething: function(x: number): string
end

declare const mylib: MyLib
```

## Contributing

If you find missing functions or incorrect signatures, please update the appropriate `.d.lunar` file. These declarations benefit the entire Lunar community!

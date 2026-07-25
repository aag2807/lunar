# Lunar

**A statically-typed superset of Lua that compiles to clean, efficient Lua code.**

Lunar adds modern type safety and programming features to Lua while maintaining 100% compatibility with existing Lua code and libraries.

```lunar
-- Type-safe Lunar code
function calculateTotal(price: number, quantity: number): number
    return price * quantity
end

class ShoppingCart
    private items: number[]

    constructor()
        self.items = {}
    end

    addItem(price: number): void
        table.insert(self.items, price)
    end

    getTotal(): number
        local total: number = 0
        for _, price in ipairs(self.items) do
            total = total + price
        end
        return total
    end
end
```

## Features

### Type System
✅ **Type Safety** - Catch errors at compile time, not runtime
✅ **Classes & OOP** - Modern object-oriented programming with inheritance
✅ **Interfaces** - Define contracts and ensure implementation
✅ **Enums** - Type-safe enumeration values
✅ **Generics** - Write reusable, type-safe code
✅ **Union Types** - Flexible type combinations (`string | number`)
✅ **Intersection Types** - Combine multiple types (`T1 & T2`)
✅ **Pattern Matching** - Powerful match expressions with discriminated unions, guards, and destructuring
✅ **Optional Parameters** - Optional function parameters (`param: Type?`)
✅ **Multiple Return Values** - Tuple types for multi-value returns (`(boolean, string)`)
✅ **Advanced Types** - Mapped types, conditional types, template literal types
✅ **Type Guards** - Runtime type checking (`value is Type`)
✅ **Method Overloading** - Multiple function signatures with automatic resolution
✅ **Constructor Parameter Properties** - TypeScript-style shorthand for class properties
✅ **Readonly Properties** - Immutable properties that can only be set in constructors

## Language Reference

### Keywords

Lunar includes the following keywords:

**Type & Declaration Keywords:**
- `class` - Define a class
- `interface` - Define an interface contract
- `enum` - Define an enumeration
- `type` - Define a type alias
- `declare` - Declare types for external code (in `.d.lunar` files)
- `extends` - Class/interface inheritance
- `implements` - Implement an interface

**Access Modifiers:**
- `public` - Accessible from anywhere (default)
- `private` - Only accessible within the class
- `protected` - Accessible within class and subclasses
- `static` - Static method (in interfaces, for autocomplete)
- `readonly` - Property can only be assigned in constructor
- `abstract` - Abstract class or method

**Function & Variable Keywords:**
- `function` - Define a function
- `local` - Local variable (Lua standard)
- `const` - Constant variable (enforced at compile-time)
- `return` - Return from function
- `async` - Async function (planned feature)
- `await` - Await async operation (planned feature)

**Control Flow:**
- `if`, `then`, `else`, `elseif` - Conditional statements
- `while`, `do` - While loop
- `for`, `in` - For loop
- `break` - Break from loop
- `match`, `with` - Pattern matching
- `end` - End block

**Module System:**
- `import` - Import from module
- `export` - Export from module
- `from` - Specify module source
- `namespace` - Namespace declaration (planned)

**Type System:**
- `is` - Type guard (`value is Type`)
- `as` - Type assertion (`value as Type`)
- `keyof` - Extract keys of type
- `typeof` - Get type of value

**Other:**
- `constructor` - Class constructor
- `self` - Reference to current instance
- `super` - Reference to parent class
- `get`, `set` - Property accessors (planned)

### Primitive Types

**Basic Types:**
- `number` - Numeric values (integers and floats)
- `string` - Text values
- `boolean` - True or false values
- `nil` - Null/undefined value
- `void` - No return value (for functions)
- `any` - Any type (disables type checking)
- `unknown` - Unknown type (requires type narrowing)
- `never` - Type that never occurs

**Literal Types:**
- `true`, `false` - Boolean literals
- `42`, `3.14` - Number literals
- `"hello"` - String literals

**Composite Types:**
- `table` - Lua table type
- `table<K, V>` - Generic table with key and value types
- `Type[]` - Array type
- `(Type1, Type2)` - Tuple type (multiple return values)
- `Type1 | Type2` - Union type (can be Type1 OR Type2)
- `Type1 & Type2` - Intersection type (combines Type1 AND Type2)
- `Type?` - Optional type (Type OR nil)
- `function(params): ReturnType` - Function type

**Advanced Types:**
- `Pick<T, K>` - Pick specific properties from type
- `Omit<T, K>` - Omit specific properties from type
- `Partial<T>` - Make all properties optional
- `Required<T>` - Make all properties required
- `Record<K, V>` - Object with keys of type K and values of type V
- `Readonly<T>` - Make all properties readonly
- `keyof T` - Union of property names
- `typeof value` - Get type of a value
- Template literal types - `` `prefix_${T}` ``
- Conditional types - `T extends U ? X : Y`
- Mapped types - `{ [K in T]: U }`

### Operators

**Arithmetic:**
- `+` - Addition
- `-` - Subtraction
- `*` - Multiplication
- `/` - Division
- `//` - Integer division (floor division)
- `%` - Modulo
- `^` - Exponentiation (Lua standard)

**Comparison:**
- `==` - Equal to
- `~=` - Not equal to (Lua style)
- `!=` - Not equal to (alternative)
- `<` - Less than
- `>` - Greater than
- `<=` - Less than or equal
- `>=` - Greater than or equal

**Logical:**
- `and` - Logical AND
- `or` - Logical OR
- `not` - Logical NOT

**Bitwise:**
- `&` - Bitwise AND
- `|` - Bitwise OR
- `~` - Bitwise XOR (binary), Bitwise NOT (unary), as in Lua
- `^` - Exponentiation, **not** XOR (Lua semantics)
- `<<` - Left shift
- `>>` - Right shift

**Other:**
- `..` - String concatenation
- `...` - Variadic (spread) operator
- `#` - Length operator
- `?.` - Optional chaining
- `??` - Nullish coalescing
- `|>` - Pipe operator

### Comments

```lunar
-- Single line comment

--[[
    Multi-line comment
    Can span multiple lines
]]

--[[ Inline comment ]] local x: number = 42
```

For complete language documentation, see:
- **[Compiled Output Guide](docs/COMPILED_OUTPUT.md)** - How Lunar compiles to Lua
- **[Language Specification](LANGUAGE_SPEC.md)** - Complete language reference

### Developer Experience
✅ **Bundler** - Bundle all dependencies into a single file with `--bundle`
✅ **Run Mode** - Compile and execute with `--run`
✅ **Modern Test Runner** - Colored output, test filtering, timing, pass/fail statistics
✅ **Watch Mode** - Auto-recompile on file changes with `--watch`
✅ **REPL** - Interactive mode for rapid prototyping with `--repl`
✅ **Language Server Protocol (LSP)** - Full IDE integration with diagnostics, hover, completions, go-to-definition
✅ **Editor Integrations** - VS Code extension and Neovim plugin included
✅ **Code Formatter** - Automatic code formatting with `--format`
✅ **Linter** - Best practices checking with `--lint`
✅ **Source Maps** - Debug with original Lunar source line numbers (Source Map v3)
✅ **Excellent Error Messages** - Clear, helpful errors with source context

### Lua Targets
✅ **Multi-Target Support** - Compile for Lua 5.1, 5.2, 5.3, 5.4, or LuaJIT
✅ **Automatic Compatibility** - Integer division (`//`) and bitwise ops auto-convert per target
✅ **Bitwise Operators** - `&`, `|`, `~`, `<<`, `>>` work across all Lua versions
✅ **Tree Shaking** - Remove unused exports from bundles

### Lua Compatibility
✅ **Context-Aware Keywords** - `string`, `table`, `type` work as both types and identifiers
✅ **Declaration Files** - Type definitions for existing Lua libraries (`.d.lunar`)
✅ **Complete Standard Library Types** - Full type coverage for all Lua stdlib modules
✅ **Vendor Libraries** - Built-in libraries for testing, JSON, HTTP, and formatting
✅ **Clean Lua Output** - Generates readable, efficient Lua code
✅ **100% Lua Compatible** - Use any Lua library seamlessly

## Installation

### Prerequisites
- Go 1.16 or higher

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/lunar.git
cd lunar

# Build the compiler
go build -o lunar ./cmd/lunar

# Build the LSP server (for IDE integration)
go build -o lunar-lsp ./cmd/lunar-lsp

# Build the declaration generator (optional)
go build -o lunar2decl ./cmd/lunar2decl

# Add to your PATH (optional)
sudo cp lunar lunar-lsp lunar2decl /usr/local/bin/
```

### Using Make (recommended)

```bash
# Build both tools
make build

# Install to /usr/local/bin
make install

# Run tests
make test

# Clean build artifacts
make clean
```

## Quick Start

### 1. Create a Lunar file

```lunar
-- hello.lunar
function greet(name: string): string
    return "Hello, " .. name .. "!"
end

local message: string = greet("World")
print(message)
```

### 2. Compile to Lua

```bash
lunar hello.lunar
```

This generates `hello.lua`:

```lua
function greet(name)
    return "Hello, " .. name .. "!"
end

local message = greet("World")
print(message)
```

### 3. Run the Lua code

```bash
lua hello.lua
```

## Usage

### Basic Compilation

```bash
# Compile with type checking (default)
lunar input.lunar

# Compile without type checking
lunar --no-typecheck input.lunar

# Generate source map for debugging
lunar --source-map input.lunar

# Specify output file
lunar -o output.lua input.lunar

# Target specific Lua version (default: lua53)
lunar --target luajit input.lunar
lunar --target lua51 input.lunar
lunar --target lua54 input.lunar

# Bundle all dependencies into a single file
lunar --bundle main.lunar

# Bundle and run immediately
lunar --bundle --run main.lunar

# Watch mode with bundling and auto-run
lunar --bundle --watch --run main.lunar

# Compile and run (without bundling)
lunar --run input.lunar
```

### Project Management

```bash
# Create a new project from template
lunar --template basic create my-project
lunar --template cli create my-cli-app
lunar --template web create my-api
lunar --template library create my-lib

# List available templates
lunar create list

# Initialize lunar.json in existing project
lunar init
lunar init -y                    # Skip prompts
lunar init --name myproject      # Set project name
lunar init --strict              # Enable strict mode

# Manage dependencies
lunar add mypackage              # Add package
lunar add user/repo              # Add from GitHub
lunar add mypackage --dev        # Add dev dependency
lunar remove mypackage           # Remove package
lunar install                    # Install all dependencies

# Run scripts from lunar.json
lunar run dev
lunar run build
lunar run test
```

### Testing

```bash
# Run tests in a directory
lunar --test ./tests

# Run tests with filtering
lunar --test ./tests --filter "Math"

# Run tests with coverage (experimental)
lunar --test --coverage ./tests

# Watch tests and re-run on changes
lunar --test --test-watch ./tests
```

### Code Quality

```bash
# Format code (print to stdout)
lunar --format input.lunar

# Format code (write back to file)
lunar --format-write input.lunar

# Lint code for best practices
lunar --lint input.lunar
```

### Advanced Options

```bash
# Combine options (note: flags must come before filename)
lunar --source-map --target luajit -o output.lua input.lunar

# Start interactive REPL
lunar --repl

# Advanced compiler features
lunar --wasm input.lunar         # Compile to WebAssembly
lunar --jit-hints input.lunar    # Add JIT optimization hints
lunar --plugin-load ./plugin.so  # Load compiler plugin

# Show version
lunar --version

# Show help
lunar --help
```

### REPL Mode

Lunar includes an interactive REPL for rapid prototyping and experimentation:

```bash
lunar --repl
```

```
Lunar REPL - Interactive Mode
Version 1.5.0
Type 'exit' or 'quit' to exit, 'help' for commands

>>> local x: number = 42
>>> print(x * 2)
84
>>> function greet(name: string): string
...     return "Hello, " .. name
... end
>>> greet("Lunar")
Hello, Lunar
```

## Documentation

- **[Language Specification](LANGUAGE_SPEC.md)** - Complete language reference
- **[Compiled Output Guide](docs/COMPILED_OUTPUT.md)** - How Lunar features compile to Lua
- **[Standard Library](stdlib/README.md)** - Type declarations for Lua stdlib
- **[Declaration Generator](cmd/lunar2decl/README.md)** - Generate `.d.lunar` files
- **[Examples](examples/)** - Sample code and use cases

## Examples

### Type Safety

```lunar
function divide(a: number, b: number): number
    if b == 0 then
        error("Division by zero")
    end
    return a / b
end

-- Type error caught at compile time!
-- divide("10", 5)  -- Error: cannot pass string to number parameter
```

### Classes and Inheritance

```lunar
class Animal
    protected name: string

    constructor(name: string)
        self.name = name
    end

    speak(): void
        print("Some sound")
    end
end

class Dog extends Animal
    constructor(name: string)
        super(name)
    end

    speak(): void
        print(self.name .. " says: Woof!")
    end
end

local dog: Dog = Dog("Buddy")
dog.speak()  -- Outputs: Buddy says: Woof!
```

### Generics

```lunar
class Box<T>
    private value: T

    constructor(value: T)
        self.value = value
    end

    getValue(): T
        return self.value
    end

    setValue(value: T): void
        self.value = value
    end
end

local numberBox: Box<number> = Box<number>(42)
local stringBox: Box<string> = Box<string>("hello")
```

### Method Overloading

```lunar
-- Multiple signatures for the same function
function add(a: number, b: number): number
    return a + b
end

function add(a: string, b: string): string
    return a .. b
end

local sum: number = add(1, 2)      -- Returns 3
local concat: string = add("a", "b")  -- Returns "ab"
```

### Pattern Matching

```lunar
-- Powerful pattern matching for discriminated unions
function handleResult(result: any): string
    return match result with
        | { type: "success", value: v } -> "Got value: " .. tostring(v)
        | { type: "error", message: msg } -> "Error: " .. msg
        | _ -> "Unknown result"
    end
end

-- Guards for conditional matching
function classify(n: number): string
    return match n with
        | x when x > 100 -> "large"
        | x when x > 10 -> "medium"
        | x when x > 0 -> "small"
        | 0 -> "zero"
        | _ -> "negative"
    end
end

-- Literal patterns and wildcards
function describe(value: any): string
    return match value with
        | 0 -> "zero"
        | 42 -> "the answer"
        | "hello" -> "greeting"
        | true -> "yes"
        | nil -> "nothing"
        | _ -> "something else"
    end
end
```

### Optional Parameters

```lunar
-- Optional parameters use Type? syntax
function greet(name: string, title: string?): string
    if title ~= nil then
        return "Hello, " .. title .. " " .. name
    else
        return "Hello, " .. name
    end
end

print(greet("Alice", nil))      -- "Hello, Alice"
print(greet("Bob", "Dr."))      -- "Hello, Dr. Bob"
```

### Multiple Return Values

```lunar
-- Functions can return tuples
function divide(a: number, b: number): (boolean, number | string)
    if b == 0 then
        return false, "Division by zero"
    end
    return true, a / b
end

local success: boolean, result: number | string = divide(10, 2)
print(success, result)  -- true, 5

local success2: boolean, error: number | string = divide(10, 0)
print(success2, error)  -- false, "Division by zero"
```

### Constructor Parameter Properties

```lunar
-- TypeScript-style shorthand for declaring class properties
class Person
    function constructor(
        public name: string,
        private readonly id: number,
        protected age: number
    )
        -- Properties are automatically created and assigned
    end

    function getId(): number
        return self.id
    end
end

local person = Person("Alice", 123, 30)
print(person.name)  -- "Alice" (public)
-- person.id = 456  -- Error: cannot assign to readonly property
```

### Bitwise Operations (Multi-Target)

```lunar
-- Bitwise operators work across all Lua versions!
local flags: number = 0x0F  -- 0b00001111

-- Bitwise AND, OR, XOR
local masked: number = flags & 0x0C     -- 0b00001100
local combined: number = flags | 0x30   -- 0b00111111
local flipped: number = flags ~ 0xFF     -- 0b11110000 (binary ~ is XOR)

-- Bitwise NOT
local inverted: number = ~flags         -- 0b11110000

-- Bit shifts
local shifted_left: number = flags << 2 -- 0b00111100
local shifted_right: number = flags >> 2 -- 0b00000011

-- Compiles to native operators on Lua 5.3/5.4
-- Auto-converts to bit32.* on Lua 5.2
-- Auto-converts to bit.* on LuaJIT/Lua 5.1
```

### Bundling Multiple Files

```lunar
-- utils.lunar
export function greet(name: string): string
    return "Hello, " .. name .. "!"
end

export function add(a: number, b: number): number
    return a + b
end
```

```lunar
-- main.lunar
import { greet, add } from "./utils"

local message: string = greet("World")
print(message)

local sum: number = add(10, 20)
print("Sum: " .. tostring(sum))
```

```bash
# Bundle and run
lunar --bundle --run main.lunar
```

### Using Lua Libraries with Type Safety

```lunar
-- Copy stdlib declarations to your project
-- cp stdlib/*.d.lunar .

-- Now use Lua stdlib with full type safety!
function calculateCircleArea(radius: number): number
    local area: number = math.pi * math.pow(radius, 2)
    return math.floor(area * 100) / 100
end

local result: number = calculateCircleArea(5.0)
print(result)
```

## Declaration Files

Create type definitions for existing Lua libraries:

### Manual Creation
```lunar
-- socket.d.lunar
declare interface Socket
    connect: function(host: string, port: number): boolean
    send: function(data: string): boolean
    receive: function(): string | nil
    close: function(): void
end

declare function socket_connect(host: string, port: number): Socket end
```

### Auto-Generate from Lua Code
```bash
# Generate declarations from existing Lua files
lunar2decl mylib.lua

# This creates mylib.d.lunar with function signatures
# Manually refine the types for better type safety
```

## Error Messages

Lunar provides clear, helpful error messages with source context:

```
test.lunar: Type errors found:

  Error 1: test.lunar:4:2
  Cannot assign type 'number' to variable of type 'string'

     2 |
     3 | function calculateArea(width: number, height: number): number
     4 | 	local area: string = width * height
       |  ^
     5 | 	return area
```

## Standard Library Support

Lunar includes complete type declarations for Lua 5.1 standard library:

- ✅ **lua.d.lunar** - Core globals (print, tostring, tonumber, pcall, etc.)
- ✅ **math.d.lunar** - Math functions (sin, cos, random, floor, etc.)
- ✅ **string.d.lunar** - String manipulation (find, match, format, etc.)
- ✅ **table.d.lunar** - Table functions (insert, remove, sort, concat)
- ✅ **io.d.lunar** - File I/O (open, read, write, etc.)
- ✅ **os.d.lunar** - OS facilities (time, execute, date, etc.)
- ✅ **coroutine.d.lunar** - Coroutine support (create, resume, yield)
- ✅ **debug.d.lunar** - Debug facilities (getinfo, traceback, etc.)
- ✅ **package.d.lunar** - Module loading (path, loaded, etc.)

### Built-in Vendor Libraries

Lunar includes ready-to-use vendor libraries for common tasks:

```lunar
-- Testing framework (BDD-style with assertions)
import { describe, it, expect } from "vendor/testing"

describe("Math operations", function()
    it("should add numbers correctly", function()
        expect(1 + 1).toBe(2)
        expect(10).toBeGreaterThan(5)
    end)
end)
```

```lunar
-- JSON encoding/decoding
import { encode, decode } from "vendor/json"

local data = { name = "Lunar", version = "1.5" }
local json_str: string = encode(data)
local parsed: any = decode(json_str)
print(parsed.name)  -- "Lunar"
```

```lunar
-- HTTP client (uses curl, no external dependencies)
import { get, post, json } from "vendor/http"

local response: HttpResponse = get("https://api.example.com/data")
print(response.status)  -- 200
print(response.body)

-- JSON request helper
local data = { user = "alice" }
local response = json("POST", "https://api.example.com/users", data)
```

```lunar
-- Formatting and output utilities
import { printf, sprintf, inspect } from "vendor/fmt"

printf("Hello %s!\n", "World")
local msg: string = sprintf("Result: %d", 42)

local data = { x = 1, y = 2 }
print(inspect(data))  -- Pretty-printed table
```

See [vendor/README.md](vendor/README.md) for complete documentation.

## Project Structure

```
lunar/
├── cmd/
│   ├── lunar/          # Main compiler
│   ├── lunar-lsp/      # Language Server Protocol server
│   └── lunar2decl/     # Declaration generator tool
├── internal/
│   ├── lexer/          # Tokenization
│   ├── parser/         # AST construction
│   ├── types/          # Type checking
│   ├── codegen/        # Lua code generation
│   ├── bundler/        # Module bundler
│   ├── config/         # Configuration loader
│   ├── ast/            # AST definitions
│   ├── lsp/            # LSP implementation
│   ├── formatter/      # Code formatter
│   └── linter/         # Code linter
├── editors/
│   └── nvim/           # Neovim plugin
├── stdlib/             # Standard library declarations
├── examples/           # Example code
└── README.md           # This file
```

## Roadmap

### v1.0 ✅
- [x] Complete type system
- [x] Classes, interfaces, enums
- [x] Generics
- [x] Union and literal types
- [x] Declaration files
- [x] Standard library declarations
- [x] Improved error messages
- [x] Declaration generator tool

### v1.1 ✅
- [x] Context-aware keywords (full string/table stdlib support)
- [x] Source maps for debugging

### v1.2 ✅
- [x] Enhanced error suggestions ("Did you mean...?")
- [x] Comprehensive stdlib coverage (coroutine, debug, package)
- [x] Integration tests

### v1.3 ✅
- [x] Method overloading with automatic resolution
- [x] Constructor parameter properties (TypeScript-style)
- [x] Enhanced readonly property enforcement
- [x] Code formatter (`--format`, `--format-write`)
- [x] Linter (`--lint`)
- [x] Language Server Protocol (LSP) implementation
- [x] Neovim plugin

### v1.4 ✅
- [x] Webpack-like bundler (`--bundle`) for multi-file projects
- [x] Run mode (`--run`) to execute after compilation
- [x] Watch mode with bundling and auto-run
- [x] Topological sort for correct dependency order
- [x] Module system with internal `__require` for bundled code

### v1.5 (Current) ✅
- [x] Watch all dependencies (not just entry file)
- [x] Project configuration via `lunar.config.json`
- [x] Path aliases (`@/utils` -> `src/utils`)
- [x] Index file resolution (`./utils` -> `./utils/index.lunar`)
- [x] Auto-create output directories
- [x] Tree shaking (remove unused exports)
- [x] Built-in vendor libraries (testing, json, http, fmt)
- [x] Test runner (`--test` for discovering and running tests)
- [x] Multi-target support (`--target lua51/lua52/lua53/lua54/luajit`)
- [x] Integer division compatibility (auto-convert `//` to `math.floor`)
- [x] Bitwise operators (`&`, `|`, `^`, `~`, `<<`, `>>`) with target-specific conversion
- [x] Variadic function support in type system
- [x] Optional parameters (`param: Type?`)
- [x] Multiple return values with tuple types
- [x] REPL (interactive mode)
- [x] Advanced type features (mapped, conditional, template literal types)
- [x] VS Code extension and Neovim plugin

### v1.6 (Current) ✅
- [x] Better test runner with colored output
- [x] Test filtering by pattern (`--filter`)
- [x] Test timing and statistics
- [x] Pattern matching with discriminated unions, guards, and destructuring
- [x] Optional chaining (`?.`) and nullish coalescing (`??`)
- [x] Pipe operator (`|>`) for functional programming
- [x] Complete LSP features (find references, rename, code actions, inlay hints)
- [x] Package manager (`lunar add`, `lunar remove`, `lunar install`)
- [x] Project scaffolding tool (`lunar create` with templates)
- [x] Test coverage infrastructure (debug hook-based tracking)
- [x] Advanced compiler features (WASM, JIT hints, plugin system)

### v2.0 (Future)
- [ ] Null coalescing assignment (`??=`)
- [ ] Incremental compilation and caching
- [ ] Performance optimizations (dead code elimination, minification)
- [ ] REPL improvements (LSP-powered auto-completion, persistent history)
- [ ] Documentation generator (`--docs` flag)
- [ ] Package registry and publishing
- [ ] Source map debugging improvements

## Configuration

Lunar supports project configuration via `lunar.config.json`:

```json
{
  "compilerOptions": {
    "strict": false,
    "noTypeCheck": false,
    "sourceMap": false,
    "bundle": true,
    "treeShake": true,
    "target": "lua53",
    "baseUrl": ".",
    "paths": {
      "@utils": "./src/utils",
      "@/*": "./src/*"
    }
  },
  "outDir": "dist",
  "outFile": "dist/bundle.lua",
  "include": ["**/*.lunar"],
  "exclude": ["node_modules"]
}
```

### Configuration Options

| Option | Description |
|--------|-------------|
| `compilerOptions.target` | Target Lua version: `lua51`, `lua52`, `lua53`, `lua54`, `luajit` |
| `compilerOptions.bundle` | Enable bundling by default |
| `compilerOptions.treeShake` | Remove unused exports from bundles |
| `compilerOptions.sourceMap` | Generate source maps |
| `compilerOptions.paths` | Path aliases for imports |
| `compilerOptions.baseUrl` | Base directory for path resolution |
| `outDir` | Output directory for compiled files |
| `outFile` | Single output file (for bundling) |

## IDE Integration

Lunar includes a full Language Server Protocol (LSP) implementation for IDE integration.

### Current LSP Features
- ✅ Real-time diagnostics (type errors, parse errors)
- ✅ Go to definition
- ✅ Hover type information
- ✅ Auto-completion (variables, functions, classes, keywords)
- ✅ Find references (find all usages of a symbol)
- ✅ Rename symbol (rename across all files)
- ✅ Code actions (quick fixes for undefined variables, type mismatches, missing imports)
- ✅ Inlay hints (type annotations for inferred types)

### Neovim

A ready-to-use Neovim plugin is included in `editors/nvim/`:

```lua
-- Using lazy.nvim
{
  dir = "/path/to/lunar/editors/nvim",
  ft = "lunar",
  config = function()
    require("lunar").setup()
  end,
}
```

See [editors/nvim/README.md](editors/nvim/README.md) for full installation instructions.

### VS Code

A VS Code extension is included in `editors/vscode/`:

1. Copy or symlink `editors/vscode/` to `~/.vscode/extensions/lunar/`
2. Reload VS Code
3. Open any `.lunar` file to activate

### Other Editors

The `lunar-lsp` binary can be used with any editor that supports LSP:

```bash
# Build the LSP server
go build -o lunar-lsp ./cmd/lunar-lsp

# Configure your editor to use it for .lunar files
```

### Documentation
- **[LSP Design Document](docs/LSP_DESIGN.md)** - Architecture and implementation details
- **[Neovim Plugin](editors/nvim/README.md)** - Installation and configuration
- **[VS Code Extension](editors/vscode/README.md)** - Installation and usage

## Contributing

Contributions are welcome! Areas where help is especially appreciated:

- Additional standard library declarations
- Bug fixes and error reporting
- Documentation improvements
- Example code and tutorials
- Testing on different platforms

## License

[MIT License](LICENSE)

## Why Lunar?

**For Lua Developers:**
- Add type safety to catch bugs early
- Better IDE support and autocompletion
- Modern OOP features while keeping Lua's simplicity
- No runtime overhead - compiles to clean Lua

**For TypeScript/Typed Language Developers:**
- Familiar syntax and type system
- Target embedded systems and game engines that use Lua
- Lightweight and fast compilation
- Full interop with existing Lua ecosystem

## Comparison with Lua

| Feature | Lua | Lunar |
|---------|-----|-------|
| Static typing | ❌ | ✅ |
| Classes/OOP | Manual (metatables) | ✅ Built-in |
| Interfaces | ❌ | ✅ |
| Generics | ❌ | ✅ |
| Compile-time errors | ❌ | ✅ |
| Runtime performance | ⚡ Fast | ⚡ Fast (same) |
| Lua compatibility | ✅ | ✅ |
| Learning curve | Easy | Easy-Medium |

## Acknowledgments

Inspired by TypeScript, with design principles adapted for the Lua ecosystem.

---

**[Get Started Now](#quick-start)** | **[View Examples](examples/)** | **[Read the Spec](LANGUAGE_SPEC.md)**

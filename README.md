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
✅ **Method Overloading** - Multiple function signatures with automatic resolution
✅ **Constructor Parameter Properties** - TypeScript-style shorthand for class properties
✅ **Readonly Properties** - Immutable properties that can only be set in constructors

### Developer Experience
✅ **Bundler** - Bundle all dependencies into a single file with `--bundle`
✅ **Run Mode** - Compile and execute with `--run`
✅ **Language Server Protocol (LSP)** - Full IDE integration with diagnostics, hover, completions
✅ **Neovim Plugin** - Ready-to-use editor integration
✅ **Code Formatter** - Automatic code formatting with `--format`
✅ **Linter** - Best practices checking with `--lint`
✅ **Source Maps** - Debug with original Lunar source line numbers (Source Map v3)
✅ **Excellent Error Messages** - Clear, helpful errors with "Did you mean?" suggestions

### Lua Compatibility
✅ **Context-Aware Keywords** - `string`, `table`, `type` work as both types and identifiers
✅ **Declaration Files** - Type definitions for existing Lua libraries (`.d.lunar`)
✅ **Complete Standard Library Types** - Full type coverage including `string.*` and `table.*`
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

```bash
# Compile with type checking (default)
lunar input.lunar

# Compile without type checking
lunar --no-typecheck input.lunar

# Generate source map for debugging
lunar --source-map input.lunar

# Specify output file
lunar -o output.lua input.lunar

# Bundle all dependencies into a single file
lunar --bundle main.lunar

# Bundle and run immediately
lunar --bundle --run main.lunar

# Watch mode with bundling and auto-run
lunar --bundle --watch --run main.lunar

# Compile and run (without bundling)
lunar --run input.lunar

# Format code (print to stdout)
lunar --format input.lunar

# Format code (write back to file)
lunar --format-write input.lunar

# Lint code for best practices
lunar --lint input.lunar

# Combine options (note: flags must come before filename)
lunar --source-map -o output.lua input.lunar

# Show version
lunar --version

# Show help
lunar --help
```

## Documentation

- **[Language Specification](LANGUAGE_SPEC.md)** - Complete language reference
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
declare interface Socket {
    connect: function(host: string, port: number): boolean
    send: function(data: string): boolean
    receive: function(): string | nil
    close: function(): void
}
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

Lunar includes type declarations for Lua 5.1 standard library:

- ✅ **lua.d.lunar** - Core globals (print, tostring, tonumber, etc.)
- ✅ **math.d.lunar** - Math functions (sin, cos, random, floor, etc.)
- ✅ **io.d.lunar** - File I/O (open, read, write, etc.)
- ✅ **os.d.lunar** - OS facilities (time, execute, date, etc.)
- ⚠️ **string/table** - Currently limited due to keyword conflicts (v1.1)

Simply copy the declarations to your project directory for automatic type checking!

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

### v1.4 (Current) ✅
- [x] Webpack-like bundler (`--bundle`) for multi-file projects
- [x] Run mode (`--run`) to execute after compilation
- [x] Watch mode with bundling and auto-run
- [x] Topological sort for correct dependency order
- [x] Module system with internal `__require` for bundled code

### v2.0 (Future)
- [ ] Package manager integration
- [ ] Incremental compilation
- [ ] More LSP features (find references, rename, code actions)
- [ ] VS Code extension
- [ ] Performance optimizations

## IDE Integration

Lunar includes a full Language Server Protocol (LSP) implementation for IDE integration.

### Current Features
- Real-time diagnostics (type errors, parse errors)
- Go to definition
- Hover type information
- Auto-completion (variables, functions, classes, keywords)

### Neovim

A ready-to-use Neovim plugin is included:

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

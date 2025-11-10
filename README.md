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

✅ **Type Safety** - Catch errors at compile time, not runtime
✅ **Classes & OOP** - Modern object-oriented programming with inheritance
✅ **Interfaces** - Define contracts and ensure implementation
✅ **Enums** - Type-safe enumeration values
✅ **Generics** - Write reusable, type-safe code
✅ **Union Types** - Flexible type combinations (`string | number`)
✅ **Declaration Files** - Type definitions for existing Lua libraries (`.d.lunar`)
✅ **Standard Library Types** - Built-in declarations for Lua 5.1 stdlib
✅ **Excellent Error Messages** - Clear, helpful errors with source context
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

# Build the declaration generator (optional)
go build -o lunar2decl ./cmd/lunar2decl

# Add to your PATH (optional)
sudo cp lunar /usr/local/bin/
sudo cp lunar2decl /usr/local/bin/
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
lunar input.lunar --no-type-check

# Specify output file
lunar input.lunar -o output.lua

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
│   └── lunar2decl/     # Declaration generator tool
├── internal/
│   ├── lexer/          # Tokenization
│   ├── parser/         # AST construction
│   ├── types/          # Type checking
│   ├── codegen/        # Lua code generation
│   └── ast/            # AST definitions
├── stdlib/             # Standard library declarations
├── examples/           # Example code
└── README.md           # This file
```

## Roadmap

### v1.0 (Current) ✅
- [x] Complete type system
- [x] Classes, interfaces, enums
- [x] Generics
- [x] Union and literal types
- [x] Declaration files
- [x] Standard library declarations
- [x] Improved error messages
- [x] Declaration generator tool

### v1.1 (Planned)
- [ ] Context-aware keywords (full string/table stdlib support)
- [ ] Enhanced error suggestions ("Did you mean...?")
- [ ] More comprehensive stdlib coverage
- [ ] Performance optimizations

### v2.0 (Future)
- [ ] Language Server Protocol (LSP) for IDE integration
- [ ] Source maps for debugging
- [ ] Package manager integration
- [ ] Code formatter

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
